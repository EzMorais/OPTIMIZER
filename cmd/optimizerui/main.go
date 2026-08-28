// Comando optimizerui é o app desktop do Optimizer.
//
// A interface é HTML/CSS/JS embutida no .exe (Wails v2, sem CGO). Toda a lógica
// continua no Go compilado: o frontend não decide nada, só pede e mostra.
package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailswin "github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Optimizer",
		Width:            1120,
		Height:           780,
		MinWidth:         900,
		MinHeight:        620,
		WindowStartState: options.Normal,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 13, G: 16, B: 18, A: 1},
		OnStartup:        app.startup,
		Bind:             []any{app},
		Windows: &wailswin.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         wailswin.Auto,
			Theme:                wailswin.Dark,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func encerrar(ctx context.Context) {
	if ctx != nil {
		runtime.Quit(ctx)
	}
}
