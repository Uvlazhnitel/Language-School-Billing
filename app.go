package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"langschool/internal/infra"
	"langschool/internal/paths"
)

type App struct {
	db *infra.DB
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	base := filepath.Join(userHome(), "LangSchool")
	dirs, err := paths.Ensure(base)
	if err != nil {
		log.Fatal(err)
	}

	dbFile := filepath.Join(dirs.Data, "app.sqlite")
	db, err := infra.Open(ctx, dbFile)
	if err != nil {
		log.Fatal(err)
	}
	a.db = db

	// TODO: here we will add method bindings for the UI (CRUD, invoice generation, etc.)
	_ = time.Now()
}

func userHome() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func (a *App) Ping() string { return "ok" }

func (a *App) Greet(name string) string {
	if name == "" {
		return "Привет!"
	}
	return fmt.Sprintf("Привет, %s!", name)
}
