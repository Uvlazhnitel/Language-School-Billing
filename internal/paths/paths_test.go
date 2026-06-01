package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCreatesCapitalizedDirectories(t *testing.T) {
	base := filepath.Join(t.TempDir(), "StudentDesk")

	dirs, err := Ensure(base)
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}

	if dirs.Base != base {
		t.Fatalf("Base = %q, want %q", dirs.Base, base)
	}

	want := map[string]string{
		"Data":     filepath.Join(base, "Data"),
		"Backups":  filepath.Join(base, "Backups"),
		"Invoices": filepath.Join(base, "Invoices"),
		"Exports":  filepath.Join(base, "Exports"),
	}

	got := map[string]string{
		"Data":     dirs.Data,
		"Backups":  dirs.Backups,
		"Invoices": dirs.Invoices,
		"Exports":  dirs.Exports,
	}

	for name, wantPath := range want {
		if got[name] != wantPath {
			t.Fatalf("%s path = %q, want %q", name, got[name], wantPath)
		}
		if info, err := os.Stat(wantPath); err != nil {
			t.Fatalf("%s directory missing: %v", name, err)
		} else if !info.IsDir() {
			t.Fatalf("%s path is not a directory: %q", name, wantPath)
		}
	}
}

func TestEnsureMigratesLegacyLowercaseDirectories(t *testing.T) {
	base := filepath.Join(t.TempDir(), "StudentDesk")
	legacyInvoices := filepath.Join(base, "invoices", "2026", "06")
	legacyData := filepath.Join(base, "data")

	if err := os.MkdirAll(legacyInvoices, 0o755); err != nil {
		t.Fatalf("MkdirAll legacy invoices: %v", err)
	}
	if err := os.MkdirAll(legacyData, 0o755); err != nil {
		t.Fatalf("MkdirAll legacy data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyInvoices, "invoice.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatalf("WriteFile legacy invoice: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyData, "app.sqlite"), []byte("db"), 0o644); err != nil {
		t.Fatalf("WriteFile legacy db: %v", err)
	}

	dirs, err := Ensure(base)
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dirs.Invoices, "2026", "06", "invoice.pdf")); err != nil {
		t.Fatalf("expected migrated invoice file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirs.Data, "app.sqlite")); err != nil {
		t.Fatalf("expected migrated db file: %v", err)
	}
	if dirs.Invoices != filepath.Join(base, "Invoices") {
		t.Fatalf("Invoices path = %q, want %q", dirs.Invoices, filepath.Join(base, "Invoices"))
	}
	if dirs.Data != filepath.Join(base, "Data") {
		t.Fatalf("Data path = %q, want %q", dirs.Data, filepath.Join(base, "Data"))
	}
}
