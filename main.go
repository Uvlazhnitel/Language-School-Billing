// main.go
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

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
