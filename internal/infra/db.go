// Package infra provides infrastructure components like database connections.
package infra

import (
	"context"
	"fmt"
	"log"
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
