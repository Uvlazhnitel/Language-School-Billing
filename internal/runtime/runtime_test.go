package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
)

func TestStartBootstrapsRuntimeAndSettings(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()

	rt, err := Start(ctx, Config{
		BaseDir:     base,
		DataDir:     filepath.Join(base, "Data"),
		BackupsDir:  filepath.Join(base, "Backups"),
		InvoicesDir: filepath.Join(base, "Invoices"),
		ExportsDir:  filepath.Join(base, "Exports"),
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})

	if rt.DB == nil || rt.DB.Ent == nil {
		t.Fatal("expected initialized database client")
	}
	if rt.Attendance == nil || rt.Invoice == nil || rt.Payment == nil {
		t.Fatal("expected initialized services")
	}
	if rt.AppDBPath != filepath.Join(rt.Dirs.Data, "app.sqlite") {
		t.Fatalf("AppDBPath = %q, want %q", rt.AppDBPath, filepath.Join(rt.Dirs.Data, "app.sqlite"))
	}

	st, err := rt.DB.Ent.Settings.Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Settings query failed: %v", err)
	}
	if st.OrgName != DefaultSchoolDisplayName {
		t.Fatalf("OrgName = %q, want %q", st.OrgName, DefaultSchoolDisplayName)
	}
	if st.Address != DefaultSchoolAddress {
		t.Fatalf("Address = %q, want %q", st.Address, DefaultSchoolAddress)
	}
}

func TestStartHonorsConfiguredDirectories(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := Config{
		BaseDir:     filepath.Join(root, "base"),
		DataDir:     filepath.Join(root, "custom-data"),
		BackupsDir:  filepath.Join(root, "custom-backups"),
		InvoicesDir: filepath.Join(root, "custom-invoices"),
		ExportsDir:  filepath.Join(root, "custom-exports"),
	}

	rt, err := Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})

	if rt.Dirs.Data != cfg.DataDir {
		t.Fatalf("Dirs.Data = %q, want %q", rt.Dirs.Data, cfg.DataDir)
	}
	if rt.Dirs.Backups != cfg.BackupsDir {
		t.Fatalf("Dirs.Backups = %q, want %q", rt.Dirs.Backups, cfg.BackupsDir)
	}
	if rt.Dirs.Invoices != cfg.InvoicesDir {
		t.Fatalf("Dirs.Invoices = %q, want %q", rt.Dirs.Invoices, cfg.InvoicesDir)
	}
}

func TestRuntimeCloseSucceeds(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()

	rt, err := Start(ctx, Config{
		BaseDir:     base,
		DataDir:     filepath.Join(base, "Data"),
		BackupsDir:  filepath.Join(base, "Backups"),
		InvoicesDir: filepath.Join(base, "Invoices"),
		ExportsDir:  filepath.Join(base, "Exports"),
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if err := rt.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
