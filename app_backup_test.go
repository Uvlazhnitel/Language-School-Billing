package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
