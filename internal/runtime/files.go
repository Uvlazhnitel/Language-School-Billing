package runtime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"langschool/internal/paths"
)

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirHasFonts(dir string) bool {
	return fileExists(filepath.Join(dir, "DejaVuSans.ttf")) &&
		fileExists(filepath.Join(dir, "DejaVuSans-Bold.ttf"))
}
