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
