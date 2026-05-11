package infra

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesNewSQLiteDatabase(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.sqlite")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Ent.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected database file to exist at %s: %v", dbPath, err)
	}

	count, err := db.Ent.Student.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Student.Query.Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected empty student table for new database, got %d rows", count)
	}
}

func TestOpenPreservesExistingDataOnReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.sqlite")

	first, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	created, err := first.Ent.Student.Create().
		SetFullName("Backup Safety Student").
		SetPhone("+37120000000").
		Save(ctx)
	if err != nil {
		first.Ent.Close()
		t.Fatalf("Student.Create: %v", err)
	}

	if err := first.Ent.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	second, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer second.Ent.Close()

	got, err := second.Ent.Student.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Student.Get after reopen: %v", err)
	}

	if got.FullName != created.FullName {
		t.Fatalf("expected student name %q after reopen, got %q", created.FullName, got.FullName)
	}
	if got.Phone != created.Phone {
		t.Fatalf("expected student phone %q after reopen, got %q", created.Phone, got.Phone)
	}
}
