package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"langschool/internal/paths"
)

func TestBackupBeforeMigrationCreatesCopyForExistingDB(t *testing.T) {
	base := filepath.Join(t.TempDir(), "LangSchool")
	dirs, err := paths.Ensure(base)
	if err != nil {
		t.Fatalf("paths.Ensure: %v", err)
	}

	dbPath := filepath.Join(dirs.Data, "app.sqlite")
	original := []byte("important business data")
	if err := os.WriteFile(dbPath, original, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	app := &App{dirs: dirs, appDBPath: dbPath}
	backupPath, err := app.backupBeforeMigration()
	if err != nil {
		t.Fatalf("backupBeforeMigration: %v", err)
	}
	if backupPath == "" {
		t.Fatal("expected backup path for existing database")
	}
	if !strings.Contains(filepath.Base(backupPath), "pre-migration-") {
		t.Fatalf("expected pre-migration prefix in backup filename, got %q", filepath.Base(backupPath))
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("ReadFile backup: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("expected backup contents %q, got %q", string(original), string(got))
	}
}

func TestBackupBeforeMigrationSkipsWhenDatabaseDoesNotExist(t *testing.T) {
	base := filepath.Join(t.TempDir(), "LangSchool")
	dirs, err := paths.Ensure(base)
	if err != nil {
		t.Fatalf("paths.Ensure: %v", err)
	}

	app := &App{
		dirs:      dirs,
		appDBPath: filepath.Join(dirs.Data, "app.sqlite"),
	}

	backupPath, err := app.backupBeforeMigration()
	if err != nil {
		t.Fatalf("backupBeforeMigration: %v", err)
	}
	if backupPath != "" {
		t.Fatalf("expected no backup path for missing database, got %q", backupPath)
	}
}

func TestBackupBeforeMigrationFailsWhenBackupCannotBeCreated(t *testing.T) {
	dbDir := t.TempDir()
	dbPath := filepath.Join(dbDir, "app.sqlite")
	if err := os.WriteFile(dbPath, []byte("important business data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	app := &App{
		dirs: paths.Dirs{
			Backups: filepath.Join(t.TempDir(), "missing", "backups"),
		},
		appDBPath: dbPath,
	}

	backupPath, err := app.backupBeforeMigration()
	if err == nil {
		t.Fatal("expected backupBeforeMigration to fail when backup directory is unavailable")
	}
	if backupPath != "" {
		t.Fatalf("expected empty backup path on failure, got %q", backupPath)
	}
}

func TestCleanupOldPreMigrationBackupsKeepsNewestOnly(t *testing.T) {
	base := filepath.Join(t.TempDir(), "LangSchool")
	dirs, err := paths.Ensure(base)
	if err != nil {
		t.Fatalf("paths.Ensure: %v", err)
	}

	manualBackup := filepath.Join(dirs.Backups, "app-manual.sqlite")
	if err := os.WriteFile(manualBackup, []byte("manual"), 0o644); err != nil {
		t.Fatalf("WriteFile manual backup: %v", err)
	}

	keptNames := make([]string, 0, 30)
	for i := 0; i < 35; i++ {
		name := filepath.Join(dirs.Backups, "pre-migration-20260511-120000.sqlite")
		if i > 0 {
			name = filepath.Join(dirs.Backups, "pre-migration-"+time.Date(2026, 5, 11, 12, 0, i, 0, time.UTC).Format("20060102-150405")+".sqlite")
		}
		if err := os.WriteFile(name, []byte("auto"), 0o644); err != nil {
			t.Fatalf("WriteFile auto backup: %v", err)
		}
		modTime := time.Date(2026, 5, 11, 12, 0, i, 0, time.UTC)
		if err := os.Chtimes(name, modTime, modTime); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
		if i >= 5 {
			keptNames = append(keptNames, filepath.Base(name))
		}
	}

	app := &App{dirs: dirs}
	if err := app.cleanupOldPreMigrationBackups(30); err != nil {
		t.Fatalf("cleanupOldPreMigrationBackups: %v", err)
	}

	entries, err := os.ReadDir(dirs.Backups)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	autoNames := make([]string, 0, 30)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "pre-migration-") {
			autoNames = append(autoNames, entry.Name())
		}
	}

	if len(autoNames) != 30 {
		t.Fatalf("expected 30 automatic backups after cleanup, got %d", len(autoNames))
	}

	for _, expected := range keptNames {
		if !slices.Contains(autoNames, expected) {
			t.Fatalf("expected newest backup %q to be kept", expected)
		}
	}

	if _, err := os.Stat(manualBackup); err != nil {
		t.Fatalf("expected manual backup to remain untouched: %v", err)
	}
}

func TestCleanupOldPreMigrationBackupsFailsWhenBackupsPathIsFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "backups-file")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	app := &App{
		dirs: paths.Dirs{
			Backups: filePath,
		},
	}

	if err := app.cleanupOldPreMigrationBackups(30); err == nil {
		t.Fatal("expected cleanupOldPreMigrationBackups to fail when backups path is not a directory")
	}
}
