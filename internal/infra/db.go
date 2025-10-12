package infra

import (
	"context"
	"fmt"

	"langschool/ent"

	"entgo.io/ent/dialect"
	_ "github.com/ncruces/go-sqlite3" // регистрирует драйвер "sqlite3"
)

type DB struct {
	Ent *ent.Client
}

func Open(ctx context.Context, dbPath string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?_fk=1&cache=shared&mode=rwc", dbPath)
	client, err := ent.Open(dialect.SQLite, dsn)
	if err != nil {
		return nil, err
	}

	if err := client.Schema.Create(ctx); err != nil {
		return nil, err
	}
	return &DB{Ent: client}, nil
}
