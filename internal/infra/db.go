package infra

import (
	"context"
	"fmt"
	"log"

	"langschool/ent"
	"langschool/ent/migrate"

	_ "github.com/ncruces/go-sqlite3"
)

type DB struct {
	Ent *ent.Client
}

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
