// main.go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// assets embeds the frontend/dist directory into the binary at compile time.
// This allows the application to serve the frontend assets without requiring
// a separate web server or file system access.
//
//go:embed all:frontend/dist
var assets embed.FS

// main is the entry point of the application.
// It initializes the App instance and configures the Wails application
// with window settings, lifecycle hooks, and the embedded frontend assets.
func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "LangSchool",
		Width:            1200,
		Height:           800,
		MinWidth:         1024,
		MinHeight:        640,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		OnStartup:  app.startup,
		OnDomReady: app.domReady,
		OnShutdown: app.shutdown,

		Bind: []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
