package runtime

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"langschool/internal/infra"
)

func TestFullBackupNowCreatesArchiveWithManifestAndInvoices(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "app.sqlite")
	invoicesDir := filepath.Join(root, "invoices")
	backupsDir := filepath.Join(root, "backups")

	mustCreateSQLiteDB(t, dbPath)
	if err := os.MkdirAll(filepath.Join(invoicesDir, "2026", "06"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invoicesDir, "2026", "06", "invoice-1.pdf"), []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath, err := FullBackupNow(dbPath, invoicesDir, backupsDir)
	if err != nil {
		t.Fatalf("FullBackupNow returned error: %v", err)
	}
	if !strings.HasSuffix(archivePath, ".tar.gz") {
		t.Fatalf("archive path = %q, want .tar.gz suffix", archivePath)
	}

	entries := untarEntries(t, archivePath)
	assertContains(t, entries, "data/app.sqlite")
	assertContains(t, entries, "invoices/2026/06/invoice-1.pdf")
	assertContains(t, entries, "manifest.json")
}

func TestCleanupOldFullBackupsKeepsNewestOnly(t *testing.T) {
	backupsDir := t.TempDir()
	for i := 0; i < 9; i++ {
		name := filepath.Join(backupsDir, "full-"+time.Date(2026, 6, 1, 10, 0, i, 0, time.UTC).Format("20060102-150405")+".tar.gz")
		if err := os.WriteFile(name, []byte("backup"), 0o644); err != nil {
			t.Fatal(err)
		}
		modTime := time.Date(2026, 6, 1, 10, 0, i, 0, time.UTC)
		if err := os.Chtimes(name, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	if err := CleanupOldFullBackups(backupsDir, 8); err != nil {
		t.Fatalf("CleanupOldFullBackups returned error: %v", err)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "full-") && strings.HasSuffix(entry.Name(), ".tar.gz") {
			count++
		}
	}
	if count != 8 {
		t.Fatalf("full backup count = %d, want 8", count)
	}
}

func TestCleanupOldDBBackupsKeepsNewestOnly(t *testing.T) {
	backupsDir := t.TempDir()
	for i := 0; i < 32; i++ {
		name := filepath.Join(backupsDir, "app-"+time.Date(2026, 6, 1, 10, 0, i, 0, time.UTC).Format("20060102-150405")+".sqlite")
		if err := os.WriteFile(name, []byte("backup"), 0o644); err != nil {
			t.Fatal(err)
		}
		modTime := time.Date(2026, 6, 1, 10, 0, i, 0, time.UTC)
		if err := os.Chtimes(name, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	if err := CleanupOldDBBackups(backupsDir, 30); err != nil {
		t.Fatalf("CleanupOldDBBackups returned error: %v", err)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "app-") && strings.HasSuffix(entry.Name(), ".sqlite") {
			count++
		}
	}
	if count != 30 {
		t.Fatalf("db backup count = %d, want 30", count)
	}
}

func TestRestoreFullBackupRestoresDBAndInvoices(t *testing.T) {
	root := t.TempDir()
	backupsDir := filepath.Join(root, "backups")
	liveInvoicesDir := filepath.Join(root, "invoices")
	dbPath := filepath.Join(root, "app.sqlite")

	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(liveInvoicesDir, "2026", "06"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustCreateSQLiteDB(t, dbPath)
	if err := os.WriteFile(filepath.Join(liveInvoicesDir, "2026", "06", "invoice-1.pdf"), []byte("original-pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath, err := FullBackupNow(dbPath, liveInvoicesDir, backupsDir)
	if err != nil {
		t.Fatalf("FullBackupNow returned error: %v", err)
	}
	archivedDB := untarFile(t, archivePath, "data/app.sqlite")

	replacementDB := filepath.Join(root, "replacement.sqlite")
	mustCreateSQLiteDB(t, replacementDB)
	if err := os.WriteFile(replacementDB+"-note", []byte("marker"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(replacementDB, dbPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(liveInvoicesDir, "2026", "06", "invoice-1.pdf"), []byte("mutated-pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(liveInvoicesDir, "2026", "06", "invoice-2.pdf"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}

	preRestoreArchive, err := RestoreFullBackup(archivePath, dbPath, liveInvoicesDir, backupsDir)
	if err != nil {
		t.Fatalf("RestoreFullBackup returned error: %v", err)
	}
	if preRestoreArchive == "" {
		t.Fatal("RestoreFullBackup did not create a pre-restore archive")
	}

	restoredDB, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredDB) != string(archivedDB) {
		t.Fatal("restored DB bytes do not match archived DB bytes")
	}

	restoredInvoice, err := os.ReadFile(filepath.Join(liveInvoicesDir, "2026", "06", "invoice-1.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredInvoice) != "original-pdf" {
		t.Fatalf("restored invoice content = %q, want original-pdf", restoredInvoice)
	}
	if _, err := os.Stat(filepath.Join(liveInvoicesDir, "2026", "06", "invoice-2.pdf")); !os.IsNotExist(err) {
		t.Fatalf("invoice-2.pdf should not exist after restore, got err=%v", err)
	}
}

func mustCreateSQLiteDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := infra.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ent.Close(); err != nil {
		t.Fatal(err)
	}
}

func untarEntries(t *testing.T, archivePath string) []string {
	t.Helper()

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	var names []string
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		names = append(names, strings.TrimSuffix(header.Name, "/"))
	}
	return names
}

func untarFile(t *testing.T, archivePath, want string) []byte {
	t.Helper()

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if strings.TrimSuffix(header.Name, "/") != want {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	t.Fatalf("missing %q in archive", want)
	return nil
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("missing %q in archive entries: %v", want, values)
}
