package main

import (
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "LangSchool",
		Width:     1200,
		Height:    800,
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
