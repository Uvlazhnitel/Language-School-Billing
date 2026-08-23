package infra

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/ent/payment"

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
	if !item.IsActive {
		t.Fatal("legacy enrollment is_active = false, want migration default true")
	}
}

func TestOpenMigratesPerLessonDiscountToLessonPriceOverride(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-per-lesson.sqlite")

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
			note TEXT DEFAULT '',
			course_id INTEGER,
			student_id INTEGER
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Legacy Student', 1)`,
		`INSERT INTO courses (id, name, teacher_name, type, lesson_price_cents, subscription_price_cents, is_active) VALUES (1, 'Legacy Course', '', 'group', 2000, 0, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, note, course_id, student_id) VALUES (1, 1, 'per_lesson', 1, 25, '', 1, 1)`,
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
	if item.LessonPriceOverrideCents != 1500 {
		t.Fatalf("lesson price override cents = %d, want 1500", item.LessonPriceOverrideCents)
	}
}

func TestOpenBackfillsStudentCreditPaymentAllocationSources(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-payment-allocation.sqlite")

	legacyDB, err := sql.Open("sqlite3", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer legacyDB.Close()

	for _, stmt := range []string{
		`CREATE TABLE students (id INTEGER PRIMARY KEY, full_name TEXT, is_active BOOLEAN)`,
		`CREATE TABLE invoices (id INTEGER PRIMARY KEY, version INTEGER DEFAULT 1, student_id INTEGER, period_year INTEGER, period_month INTEGER, total_amount_cents INTEGER DEFAULT 0, status TEXT DEFAULT 'issued')`,
		`CREATE TABLE payments (
			id INTEGER PRIMARY KEY,
			student_id INTEGER,
			invoice_id INTEGER,
			paid_at datetime,
			amount REAL DEFAULT 0,
			amount_cents INTEGER DEFAULT 0,
			method TEXT,
			note TEXT DEFAULT '',
			created_at datetime
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Credit Student', 1)`,
		`INSERT INTO invoices (id, student_id, period_year, period_month, total_amount_cents, status) VALUES (1, 1, 2026, 7, 10000, 'issued')`,
		`INSERT INTO payments (id, student_id, invoice_id, paid_at, amount_cents, method, note, created_at) VALUES (1, 1, 1, CURRENT_TIMESTAMP, 2500, 'cash', 'Applied from student credit', CURRENT_TIMESTAMP)`,
		`INSERT INTO payments (id, student_id, invoice_id, paid_at, amount_cents, method, note, created_at) VALUES (2, 1, 1, CURRENT_TIMESTAMP, 3000, 'cash', 'course payment (applied from credit)', CURRENT_TIMESTAMP)`,
		`INSERT INTO payments (id, student_id, invoice_id, paid_at, amount_cents, method, note, created_at) VALUES (3, 1, 1, CURRENT_TIMESTAMP, 1000, 'cash', 'ordinary payment', CURRENT_TIMESTAMP)`,
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

	items, err := db.Ent.Payment.Query().Order(payment.ByID()).All(ctx)
	if err != nil {
		t.Fatalf("Payment.Query: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("payment count = %d, want 3", len(items))
	}
	if items[0].AllocationSource != payment.AllocationSourceStudentCredit || items[0].CreditOriginNote != "" {
		t.Fatalf("first migrated payment = %+v", items[0])
	}
	if items[1].AllocationSource != payment.AllocationSourceStudentCredit || items[1].CreditOriginNote != "course payment" {
		t.Fatalf("second migrated payment = %+v", items[1])
	}
	if items[2].AllocationSource != payment.AllocationSourceDirect || items[2].CreditOriginNote != "" {
		t.Fatalf("ordinary payment changed = %+v", items[2])
	}
}

func TestOpenBackfillsSharedSubscriptionLessonsToAttendanceMonths(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-subscription-months.sqlite")

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
			note TEXT DEFAULT '',
			course_id INTEGER,
			student_id INTEGER
		)`,
		`CREATE TABLE course_month_stats (
			id INTEGER PRIMARY KEY,
			course_id INTEGER,
			year INTEGER,
			month INTEGER,
			subscription_lessons_held REAL DEFAULT 0
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Student One', 1)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (2, 'Student Two', 1)`,
		`INSERT INTO courses (id, name, teacher_name, type, lesson_price_cents, subscription_price_cents, is_active) VALUES (1, 'Legacy Course', '', 'group', 1500, 0, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, note, course_id, student_id) VALUES (1, 1, 'subscription', 1, 0, '', 1, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, note, course_id, student_id) VALUES (2, 1, 'subscription', 1, 0, '', 1, 2)`,
		`INSERT INTO course_month_stats (id, course_id, year, month, subscription_lessons_held) VALUES (1, 1, 2026, 6, 4)`,
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

	rows, err := db.Ent.AttendanceMonth.Query().All(ctx)
	if err != nil {
		t.Fatalf("AttendanceMonth.Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("attendance month count = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.Hours != 4 {
			t.Fatalf("attendance hours for student %d = %v, want 4", row.StudentID, row.Hours)
		}
	}
}

func TestOpenBackfillDoesNotOverwriteExistingAttendanceMonths(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-subscription-no-overwrite.sqlite")

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
			note TEXT DEFAULT '',
			course_id INTEGER,
			student_id INTEGER
		)`,
		`CREATE TABLE course_month_stats (
			id INTEGER PRIMARY KEY,
			course_id INTEGER,
			year INTEGER,
			month INTEGER,
			subscription_lessons_held REAL DEFAULT 0
		)`,
		`CREATE TABLE attendance_months (
			id INTEGER PRIMARY KEY,
			student_id INTEGER,
			course_id INTEGER,
			year INTEGER,
			month INTEGER,
			lessons_count REAL DEFAULT 0
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Student One', 1)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (2, 'Student Two', 1)`,
		`INSERT INTO courses (id, name, teacher_name, type, lesson_price_cents, subscription_price_cents, is_active) VALUES (1, 'Legacy Course', '', 'group', 1500, 0, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, note, course_id, student_id) VALUES (1, 1, 'subscription', 1, 0, '', 1, 1)`,
		`INSERT INTO enrollments (id, version, billing_mode, charge_materials, discount_pct, note, course_id, student_id) VALUES (2, 1, 'subscription', 1, 0, '', 1, 2)`,
		`INSERT INTO course_month_stats (id, course_id, year, month, subscription_lessons_held) VALUES (1, 1, 2026, 6, 4)`,
		`INSERT INTO attendance_months (id, student_id, course_id, year, month, lessons_count) VALUES (1, 1, 1, 2026, 6, 2)`,
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

	rows, err := db.Ent.AttendanceMonth.Query().Order(attendancemonth.ByStudentID()).All(ctx)
	if err != nil {
		t.Fatalf("AttendanceMonth.Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("attendance month count = %d, want 2", len(rows))
	}
	if rows[0].StudentID != 1 || rows[0].Hours != 2 {
		t.Fatalf("first attendance month = %+v, want student 1 with hours 2", rows[0])
	}
	if rows[1].StudentID != 2 || rows[1].Hours != 4 {
		t.Fatalf("second attendance month = %+v, want student 2 with hours 4", rows[1])
	}
}

func TestOpenBackfillsStudentCreatedAtForLegacyRows(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-student-created-at.sqlite")

	legacyDB, err := sql.Open("sqlite3", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer legacyDB.Close()

	for _, stmt := range []string{
		`CREATE TABLE students (
			id INTEGER PRIMARY KEY,
			version INTEGER DEFAULT 1,
			full_name TEXT,
			personal_code TEXT DEFAULT '',
			phone TEXT DEFAULT '',
			email TEXT DEFAULT '',
			note TEXT DEFAULT '',
			is_minor BOOLEAN DEFAULT 0,
			payer_name TEXT DEFAULT '',
			payer_role TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT 1
		)`,
		`INSERT INTO students (id, full_name, is_active) VALUES (1, 'Legacy Student', 1)`,
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

	item, err := db.Ent.Student.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Student.Get: %v", err)
	}
	if item.CreatedAt == nil || item.CreatedAt.IsZero() {
		t.Fatalf("createdAt = %v, want non-empty", item.CreatedAt)
	}
}

func TestOpenCreatesUniqueIndexForNonEmptyStudentPersonalCode(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "student-personal-code-index.sqlite")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Ent.Close()

	if _, err := db.Ent.Student.Create().
		SetFullName("First Student").
		SetPersonalCode("010101-12345").
		Save(ctx); err != nil {
		t.Fatalf("Student.Create first: %v", err)
	}

	if _, err := db.Ent.Student.Create().
		SetFullName("Duplicate Student").
		SetPersonalCode("010101-12345").
		Save(ctx); err == nil {
		t.Fatal("expected duplicate personal_code insert to fail")
	} else if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected unique constraint error, got %v", err)
	}
}

func TestOpenAllowsMultipleEmptyStudentPersonalCodes(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "student-personal-code-empty.sqlite")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Ent.Close()

	for _, name := range []string{"First Student", "Second Student"} {
		if _, err := db.Ent.Student.Create().
			SetFullName(name).
			SetPersonalCode("").
			Save(ctx); err != nil {
			t.Fatalf("Student.Create %q: %v", name, err)
		}
	}
}
