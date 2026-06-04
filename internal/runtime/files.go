package runtime

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"langschool/internal/paths"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const (
	DBBackupLimit   = 30
	FullBackupLimit = 8
)

type FullBackupManifest struct {
	CreatedAt         string `json:"createdAt"`
	DBPath            string `json:"dbPath"`
	InvoicesPath      string `json:"invoicesPath"`
	InvoiceFilesCount int    `json:"invoiceFilesCount"`
	AppVersion        string `json:"appVersion"`
}

func BackupNow(dbPath, backupsDir string) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("db path is empty")
	}
	ts := time.Now().Format("20060102-150405")
	dst := filepath.Join(backupsDir, fmt.Sprintf("app-%s.sqlite", ts))
	if err := copyFile(dbPath, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func CleanupOldDBBackups(backupsDir string, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("backup retention limit must be positive")
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return err
	}

	type backupFile struct {
		path    string
		modTime time.Time
	}

	backups := make([]backupFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "app-") || !strings.HasSuffix(name, ".sqlite") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		backups = append(backups, backupFile{
			path:    filepath.Join(backupsDir, name),
			modTime: info.ModTime(),
		})
	}

	if len(backups) <= limit {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	for _, backup := range backups[limit:] {
		if err := os.Remove(backup.path); err != nil {
			return err
		}
	}

	return nil
}

func BackupBeforeMigration(dbPath, backupsDir string) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("db path is empty")
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("db path points to a directory: %s", dbPath)
	}

	ts := time.Now().Format("20060102-150405")
	dst := filepath.Join(backupsDir, fmt.Sprintf("pre-migration-%s.sqlite", ts))
	if err := copyFile(dbPath, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func CleanupOldPreMigrationBackups(backupsDir string, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("backup retention limit must be positive")
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return err
	}

	type backupFile struct {
		path    string
		modTime time.Time
	}

	backups := make([]backupFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "pre-migration-") || !strings.HasSuffix(name, ".sqlite") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		backups = append(backups, backupFile{
			path:    filepath.Join(backupsDir, name),
			modTime: info.ModTime(),
		})
	}

	if len(backups) <= limit {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	for _, backup := range backups[limit:] {
		if err := os.Remove(backup.path); err != nil {
			return err
		}
	}

	return nil
}

func ResolveFontsDir(cfg Config, dirs paths.Dirs) (string, error) {
	var candidates []string

	if cfg.FontsDir != "" {
		candidates = append(candidates, cfg.FontsDir)
	}

	if dirs.Base != "" {
		candidates = append(candidates, filepath.Join(dirs.Base, "Fonts"))
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "Fonts"),
			filepath.Join(exeDir, "fonts"),
		)
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "Fonts"),
			filepath.Join(wd, "fonts"),
		)
	}

	for _, candidate := range candidates {
		if dirHasFonts(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("DejaVuSans.ttf & DejaVuSans-Bold.ttf not found in any known location; set LS_FONTS_DIR or place fonts into ~/StudentDesk/Fonts or ./Fonts")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func FullBackupNow(dbPath, invoicesDir, backupsDir string) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("db path is empty")
	}
	if invoicesDir == "" {
		return "", fmt.Errorf("invoices dir is empty")
	}
	if backupsDir == "" {
		return "", fmt.Errorf("backups dir is empty")
	}

	timestamp := time.Now().Format("20060102-150405")
	baseName := fmt.Sprintf("full-%s", timestamp)
	stageDir, err := os.MkdirTemp(backupsDir, baseName+"-stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stageDir)

	dataDir := filepath.Join(stageDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	if err := snapshotSQLite(dbPath, filepath.Join(dataDir, "app.sqlite")); err != nil {
		return "", err
	}

	invoicesStageDir := filepath.Join(stageDir, "invoices")
	invoiceCount, err := copyDir(invoicesDir, invoicesStageDir)
	if err != nil {
		return "", err
	}

	manifest := FullBackupManifest{
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		DBPath:            dbPath,
		InvoicesPath:      invoicesDir,
		InvoiceFilesCount: invoiceCount,
		AppVersion:        "dev",
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stageDir, "manifest.json"), manifestData, 0o644); err != nil {
		return "", err
	}

	tmpArchive, err := os.CreateTemp(backupsDir, baseName+"-*.tar.gz")
	if err != nil {
		return "", err
	}
	tmpArchivePath := tmpArchive.Name()
	if err := writeTarGz(tmpArchive, stageDir); err != nil {
		_ = tmpArchive.Close()
		_ = os.Remove(tmpArchivePath)
		return "", err
	}
	if err := tmpArchive.Close(); err != nil {
		_ = os.Remove(tmpArchivePath)
		return "", err
	}

	finalPath := filepath.Join(backupsDir, baseName+".tar.gz")
	if err := os.Rename(tmpArchivePath, finalPath); err != nil {
		_ = os.Remove(tmpArchivePath)
		return "", err
	}
	if err := os.Chmod(finalPath, 0o644); err != nil {
		return "", err
	}
	return finalPath, nil
}

func RestoreFullBackup(archivePath, dbPath, invoicesDir, backupsDir string) (string, error) {
	if archivePath == "" {
		return "", fmt.Errorf("archive path is empty")
	}
	if dbPath == "" {
		return "", fmt.Errorf("db path is empty")
	}
	if invoicesDir == "" {
		return "", fmt.Errorf("invoices dir is empty")
	}
	if backupsDir == "" {
		return "", fmt.Errorf("backups dir is empty")
	}

	stageDir, err := os.MkdirTemp(backupsDir, "restore-stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stageDir)

	if err := extractTarGz(archivePath, stageDir); err != nil {
		return "", err
	}

	restoredDB := filepath.Join(stageDir, "data", "app.sqlite")
	restoredInvoices := filepath.Join(stageDir, "invoices")
	if _, err := os.Stat(restoredDB); err != nil {
		return "", fmt.Errorf("backup archive missing data/app.sqlite: %w", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, "manifest.json")); err != nil {
		return "", fmt.Errorf("backup archive missing manifest.json: %w", err)
	}
	info, err := os.Stat(restoredInvoices)
	if err != nil {
		return "", fmt.Errorf("backup archive missing invoices directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("backup archive invoices path is not a directory")
	}

	preRestoreBackup := ""
	if fileExists(dbPath) {
		preRestoreBackup, err = FullBackupNow(dbPath, invoicesDir, backupsDir)
		if err != nil {
			return "", fmt.Errorf("pre-restore backup failed: %w", err)
		}
		if err := CleanupOldFullBackups(backupsDir, FullBackupLimit); err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(invoicesDir), 0o755); err != nil {
		return "", err
	}

	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	dbTmp := dbPath + ".restore-tmp"
	if err := copyFile(restoredDB, dbTmp); err != nil {
		return "", err
	}
	if err := os.Rename(dbTmp, dbPath); err != nil {
		_ = os.Remove(dbTmp)
		return "", err
	}

	if err := replaceDirContents(invoicesDir, restoredInvoices); err != nil {
		return "", err
	}

	return preRestoreBackup, nil
}

func CleanupOldFullBackups(backupsDir string, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("backup retention limit must be positive")
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return err
	}

	type backupFile struct {
		path    string
		modTime time.Time
	}

	backups := make([]backupFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "full-") || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		backups = append(backups, backupFile{
			path:    filepath.Join(backupsDir, name),
			modTime: info.ModTime(),
		})
	}

	if len(backups) <= limit {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	for _, backup := range backups[limit:] {
		if err := os.Remove(backup.path); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, os.MkdirAll(dst, 0o755)
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("source is not a directory: %s", src)
	}

	count := 0
	err = filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		count++
		return copyFile(path, target)
	})
	return count, err
}

func snapshotSQLite(dbPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dstPath)

	db, err := sql.Open("sqlite3", buildBackupDSN(dbPath))
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, "VACUUM INTO "+quoteSQLiteLiteral(dstPath)); err != nil {
		return err
	}
	return nil
}

func buildBackupDSN(dbPath string) string {
	return fmt.Sprintf("file:%s?_fk=1&_busy_timeout=5000&cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=FULL", dbPath)
}

func quoteSQLiteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func writeTarGz(out *os.File, root string) error {
	gzipWriter := gzip.NewWriter(out)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

func extractTarGz(archivePath, dst string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		targetPath := filepath.Join(dst, filepath.FromSlash(header.Name))
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget != dst && !strings.HasPrefix(cleanTarget, dst+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes target dir: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
	}
}

func replaceDirContents(dst, src string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}

	_, err = copyDir(src, dst)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirHasFonts(dir string) bool {
	return fileExists(filepath.Join(dir, "DejaVuSans.ttf")) &&
		fileExists(filepath.Join(dir, "DejaVuSans-Bold.ttf"))
}
