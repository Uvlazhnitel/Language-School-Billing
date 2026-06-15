package infra

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"langschool/ent/enrollment"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestOpenBackfillsSubscriptionLessonPriceFromLegacyDiscount(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.sqlite")

	legacyDB, err := sql.Open("sqlite3", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer legacyDB.Close()

	for _, stmt := range []string{
		`CREATE TABLE students (id INTEGER PRIMARY KEY, full_name TEXT, is_active BOOLEAN)`,
		`CREATE TABLE courses (
			id INTEGER PRIMARY KEY,
			version INTEGER DEFAULT 1,
			name TEXT,
			teacher_name TEXT DEFAULT '',
			type TEXT,
			lesson_price_cents INTEGER DEFAULT 0,
			subscription_price_cents INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE enrollments (
			id INTEGER PRIMARY KEY,
			version INTEGER DEFAULT 1,
			billing_mode TEXT,
			charge_materials BOOLEAN DEFAULT 1,
			discount_pct REAL DEFAULT 0,
			subscription_discount_pct REAL DEFAULT 20,
			note TEXT DEFAULT '',
			course_id INTEGER,
			student_id INTEGER
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Legacy Student', 1)`,
		`INSERT INTO courses (id, name, teacher_name, type, lesson_price_cents, subscription_price_cents, is_active) VALUES (1, 'Legacy Course', '', 'group', 1500, 0, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, subscription_discount_pct, note, course_id, student_id) VALUES (1, 1, 'subscription', 1, 10, 20, '', 1, 1)`,
	} {
		if _, err := legacyDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("legacy schema setup failed for %q: %v", stmt, err)
		}
	}

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Ent.Close()

	item, err := db.Ent.Enrollment.Query().Where(enrollment.IDEQ(1)).Only(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Query: %v", err)
	}
	if item.SubscriptionLessonPriceCents != 1050 {
		t.Fatalf("subscription lesson price cents = %d, want 1050", item.SubscriptionLessonPriceCents)
	}
}
