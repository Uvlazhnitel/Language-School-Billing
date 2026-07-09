// Package infra provides infrastructure components like database connections.
package infra

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/url"

	"langschool/ent"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// DB wraps the Ent ORM client and provides database access.
// The Ent client is used for all database operations throughout the application.
type DB struct {
	Ent *ent.Client
}

// Open creates a new database connection and initializes the schema.
// It opens a SQLite database at the specified path and runs migrations
// to ensure the schema is up to date. The connection string includes:
// - _fk=1: Enable foreign key constraints
// - _busy_timeout=5000: Wait up to 5 seconds if database is locked
// - cache=shared: Use shared cache mode for better concurrency
// - _journal_mode=WAL: Use write-ahead logging for safer crash recovery
// - _synchronous=FULL: Favor durability of business data over write speed
// - mode=rwc: Read-write-create mode
//
// Migrations are configured to safely apply additive schema updates without
// automatically dropping existing columns or indexes from user databases.
func Open(ctx context.Context, dbPath string) (*DB, error) {
	dsn := buildDSN(dbPath)
	legacySubscriptionPrices, err := loadLegacySubscriptionLessonPrices(ctx, dsn)
	if err != nil {
		return nil, err
	}

	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Apply non-destructive automatic migrations only. Schema removals must be
	// handled through explicit, manual migrations in future updates.
	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	if err := applyLegacySubscriptionLessonPrices(ctx, dsn, legacySubscriptionPrices); err != nil {
		_ = client.Close()
		return nil, err
	}
	if err := migratePerLessonDiscountsToLessonPriceOverrides(ctx, dsn); err != nil {
		_ = client.Close()
		return nil, err
	}
	if err := backfillSubscriptionAttendanceMonths(ctx, dsn); err != nil {
		_ = client.Close()
		return nil, err
	}
	if err := ensureStudentPersonalCodeUniqueIndex(ctx, dsn); err != nil {
		_ = client.Close()
		return nil, err
	}

	log.Println("DB ready at", dbPath)
	return &DB{Ent: client}, nil
}

func buildDSN(dbPath string) string {
	params := url.Values{}
	params.Set("_fk", "1")
	params.Set("_busy_timeout", "5000")
	params.Set("cache", "shared")
	params.Set("mode", "rwc")
	params.Set("_journal_mode", "WAL")
	params.Set("_synchronous", "FULL")
	return fmt.Sprintf("file:%s?%s", dbPath, params.Encode())
}

func loadLegacySubscriptionLessonPrices(ctx context.Context, dsn string) (map[int]int64, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	hasOldColumn, err := hasColumn(ctx, db, "enrollments", "subscription_discount_pct")
	if err != nil {
		return nil, err
	}
	hasNewColumn, err := hasColumn(ctx, db, "enrollments", "subscription_lesson_price_cents")
	if err != nil {
		return nil, err
	}
	if !hasOldColumn || hasNewColumn {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `
SELECT e.id, e.billing_mode, e.discount_pct, e.subscription_discount_pct, COALESCE(c.lesson_price_cents, 0)
FROM enrollments e
LEFT JOIN courses c ON c.id = e.course_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int]int64{}
	for rows.Next() {
		var (
			id                      int
			billingMode             string
			discountPct             float64
			subscriptionDiscountPct float64
			lessonPriceCents        int64
		)
		if err := rows.Scan(&id, &billingMode, &discountPct, &subscriptionDiscountPct, &lessonPriceCents); err != nil {
			return nil, err
		}
		if billingMode != "subscription" {
			out[id] = 0
			continue
		}
		totalDiscountPct := discountPct + subscriptionDiscountPct
		if totalDiscountPct < 0 {
			totalDiscountPct = 0
		}
		if totalDiscountPct > 100 {
			totalDiscountPct = 100
		}
		out[id] = int64(math.Round(float64(lessonPriceCents) * (1 - totalDiscountPct/100.0)))
	}
	return out, rows.Err()
}

func applyLegacySubscriptionLessonPrices(ctx context.Context, dsn string, prices map[int]int64) error {
	if len(prices) == 0 {
		return nil
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	for enrollmentID, priceCents := range prices {
		if _, err := db.ExecContext(
			ctx,
			`UPDATE enrollments SET subscription_lesson_price_cents = ? WHERE id = ? AND subscription_lesson_price_cents < 0`,
			priceCents,
			enrollmentID,
		); err != nil {
			return err
		}
	}
	return nil
}

func migratePerLessonDiscountsToLessonPriceOverrides(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	hasOverrideColumn, err := hasColumn(ctx, db, "enrollments", "lesson_price_override_cents")
	if err != nil {
		return err
	}
	if !hasOverrideColumn {
		return nil
	}

	_, err = db.ExecContext(ctx, `
UPDATE enrollments
SET lesson_price_override_cents = CAST(ROUND(COALESCE((
	SELECT c.lesson_price_cents
	FROM courses c
	WHERE c.id = enrollments.course_id
), 0) * (1 - discount_pct / 100.0)) AS INTEGER)
WHERE billing_mode = 'per_lesson'
  AND discount_pct <> 0
  AND lesson_price_override_cents < 0
`)
	return err
}

func backfillSubscriptionAttendanceMonths(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	hasAttendanceTable, err := hasTable(ctx, db, "attendance_months")
	if err != nil {
		return err
	}
	hasCourseMonthStatsTable, err := hasTable(ctx, db, "course_month_stats")
	if err != nil {
		return err
	}
	if !hasAttendanceTable || !hasCourseMonthStatsTable {
		return nil
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO attendance_months (student_id, course_id, year, month, lessons_count)
SELECT e.student_id, cms.course_id, cms.year, cms.month, cms.subscription_lessons_held
FROM course_month_stats cms
JOIN enrollments e ON e.course_id = cms.course_id
WHERE e.billing_mode = 'subscription'
  AND NOT EXISTS (
    SELECT 1
    FROM attendance_months am
    WHERE am.student_id = e.student_id
      AND am.course_id = cms.course_id
      AND am.year = cms.year
      AND am.month = cms.month
  )
`)
	return err
}

func ensureStudentPersonalCodeUniqueIndex(ctx context.Context, dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
CREATE UNIQUE INDEX IF NOT EXISTS idx_students_personal_code_nonempty_unique
ON students (personal_code COLLATE NOCASE)
WHERE personal_code <> ''
`)
	return err
}

func hasColumn(ctx context.Context, db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+tableName+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}

func hasTable(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	row := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1", tableName)
	var value int
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return value == 1, nil
}
