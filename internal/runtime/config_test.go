package runtime

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigUsesEnvironmentOverrides(t *testing.T) {
	home := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "data-store")
	backupsDir := filepath.Join(t.TempDir(), "backup-store")
	invoicesDir := filepath.Join(t.TempDir(), "invoice-store")

	t.Setenv("APP_DATA_DIR", dataDir)
	t.Setenv("BACKUPS_DIR", backupsDir)
	t.Setenv("INVOICES_DIR", invoicesDir)
	t.Setenv("LS_FONTS_DIR", "/fonts")
	t.Setenv("APP_BASE_URL", "https://example.test")

	cfg := LoadConfig(home)

	if cfg.DataDir != dataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, dataDir)
	}
	if cfg.BackupsDir != backupsDir {
		t.Fatalf("BackupsDir = %q, want %q", cfg.BackupsDir, backupsDir)
	}
	if cfg.InvoicesDir != invoicesDir {
		t.Fatalf("InvoicesDir = %q, want %q", cfg.InvoicesDir, invoicesDir)
	}
	if cfg.FontsDir != "/fonts" {
		t.Fatalf("FontsDir = %q, want %q", cfg.FontsDir, "/fonts")
	}
	if cfg.BaseURL != "https://example.test" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.test")
	}
}

func TestLoadConfigFallsBackToDesktopLayout(t *testing.T) {
	home := t.TempDir()

	cfg := LoadConfig(home)

	wantBase := filepath.Join(home, AppDirName)
	if cfg.BaseDir != wantBase {
		t.Fatalf("BaseDir = %q, want %q", cfg.BaseDir, wantBase)
	}
	if cfg.DataDir != filepath.Join(wantBase, "Data") {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, filepath.Join(wantBase, "Data"))
	}
	if cfg.BackupsDir != filepath.Join(wantBase, "Backups") {
		t.Fatalf("BackupsDir = %q, want %q", cfg.BackupsDir, filepath.Join(wantBase, "Backups"))
	}
	if cfg.InvoicesDir != filepath.Join(wantBase, "Invoices") {
		t.Fatalf("InvoicesDir = %q, want %q", cfg.InvoicesDir, filepath.Join(wantBase, "Invoices"))
	}
}
