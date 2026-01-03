// Package infra provides infrastructure components like database connections.
package infra

import (
	"context"
	"fmt"
	"log"

	"langschool/ent"
	"langschool/ent/migrate"

	_ "github.com/mattn/go-sqlite3"
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
// - mode=rwc: Read-write-create mode
//
// Migrations are configured to drop columns and indexes that have been
// removed from the schema, ensuring the database structure matches the code.
func Open(ctx context.Context, dbPath string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_fk=1&_busy_timeout=5000&cache=shared&mode=rwc", dbPath)

	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Use migration options to enable dropping columns and indexes that were removed from schema
	if err := client.Schema.Create(ctx, migrate.WithDropColumn(true), migrate.WithDropIndex(true)); err != nil {
		_ = client.Close()
		return nil, err
	}

	log.Println("DB ready at", dbPath)
	return &DB{Ent: client}, nil
}
