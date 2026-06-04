package auth_test

import (
	"context"
	"path/filepath"
	"testing"

	"langschool/ent/user"
	"langschool/internal/auth"
	"langschool/internal/runtime"
)

func TestDeleteUserBlocksLastActiveAdmin(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()

	rt, err := runtime.Start(ctx, runtime.Config{
		BaseDir:       base,
		DataDir:       filepath.Join(base, "Data"),
		BackupsDir:    filepath.Join(base, "Backups"),
		InvoicesDir:   filepath.Join(base, "Invoices"),
		ExportsDir:    filepath.Join(base, "Exports"),
		AdminUsername: "admin",
		AdminPassword: "secret-password",
		SessionSecret: "session-secret",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})

	admin, err := rt.DB.Ent.User.Query().Where(user.UsernameEQ("admin")).Only(ctx)
	if err != nil {
		t.Fatalf("admin query failed: %v", err)
	}

	staff, err := rt.Auth.CreateUser(ctx, "staff", "staff-pass", auth.RoleStaff)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	err = rt.Auth.DeleteUser(ctx, staff.ID, admin.ID)
	if err != auth.ErrDeleteLastAdmin {
		t.Fatalf("DeleteUser error = %v, want %v", err, auth.ErrDeleteLastAdmin)
	}
}
