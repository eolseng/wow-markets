package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	startHidden := launchedInBackground()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	registerStatusItem(app)
	go func() {
		<-signals
		app.Quit()
	}()

	err := wails.Run(&options.App{
		Title:             "WoW Markets Companion",
		Width:             720,
		Height:            560,
		MinWidth:          620,
		MinHeight:         480,
		StartHidden:       startHidden,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: options.NewRGB(8, 13, 23),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.wowmarkets.companion",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				if !hasBackgroundLaunchArgument(data.Args) {
					app.ShowWindow()
				}
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
