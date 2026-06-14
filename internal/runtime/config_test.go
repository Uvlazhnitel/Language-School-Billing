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
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("SESSION_SECRET", "session-secret")

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
	if cfg.AdminUsername != "admin" {
		t.Fatalf("AdminUsername = %q, want %q", cfg.AdminUsername, "admin")
	}
	if cfg.AdminPassword != "secret" {
		t.Fatalf("AdminPassword = %q, want %q", cfg.AdminPassword, "secret")
	}
	if cfg.SessionSecret != "session-secret" {
		t.Fatalf("SessionSecret = %q, want %q", cfg.SessionSecret, "session-secret")
	}
}

func TestLoadConfigUsesServerLayoutByDefault(t *testing.T) {
	home := t.TempDir()

	cfg := LoadConfig(home)

	wantBase := DefaultServerBaseDir
	if cfg.BaseDir != wantBase {
		t.Fatalf("BaseDir = %q, want %q", cfg.BaseDir, wantBase)
	}
	if cfg.DataDir != filepath.Join(wantBase, "data") {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, filepath.Join(wantBase, "data"))
	}
	if cfg.BackupsDir != filepath.Join(wantBase, "backups") {
		t.Fatalf("BackupsDir = %q, want %q", cfg.BackupsDir, filepath.Join(wantBase, "backups"))
	}
	if cfg.InvoicesDir != filepath.Join(wantBase, "invoices") {
		t.Fatalf("InvoicesDir = %q, want %q", cfg.InvoicesDir, filepath.Join(wantBase, "invoices"))
	}
}
