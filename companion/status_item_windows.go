//go:build windows

package main

import (
	_ "embed"
	"log"

	"github.com/ra1phdd/systray-on-wails"
)

// White rendition of the bar-chart icon drawn natively for macOS in
// status_item_darwin.go; regenerate with `go run assets/generate_tray_icon.go`.
//
//go:embed assets/tray.ico
var trayIcon []byte

func registerStatusItem(app *App) {
	systray.Register(func() {
		onWindowsStatusItemReady(app)
	}, func() {
		log.Print("companion tray stopped")
	})
}

func startStatusItem(app *App) {}

func stopStatusItem() {
	systray.Quit()
}

func activateVisibleWindow() {}

func onWindowsStatusItemReady(app *App) {
	systray.SetIcon(trayIcon)
	systray.SetTitle("WMS")
	systray.SetTooltip("WoW Markets Companion is running")

	status := systray.AddMenuItem("Status: Running", "WoW Markets Companion is running")
	status.Disable()
	systray.AddSeparator()

	show := systray.AddMenuItem("Show Window", "Open WoW Markets Companion")
	hide := systray.AddMenuItem("Hide Window", "Keep WoW Markets Companion running in the background")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit WoW Markets Companion", "Stop WoW Markets Companion")

	go func() {
		for {
			select {
			case <-show.ClickedCh:
				app.ShowWindow()
			case <-hide.ClickedCh:
				app.HideWindow()
			case <-quit.ClickedCh:
				app.Quit()
				return
			}
		}
	}()
}
