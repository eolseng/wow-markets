//go:build windows

package main

import (
	_ "embed"
	"log"
	"sync"

	"github.com/ra1phdd/systray-on-wails"
)

// White rendition of the bar-chart icon drawn natively for macOS in
// status_item_darwin.go; regenerate with `go run assets/generate_tray_icon.go`.
//
//go:embed assets/tray.ico
var trayIcon []byte

var windowsStatusItem struct {
	sync.Mutex
	status *systray.MenuItem
	update *systray.MenuItem
}

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
	update := systray.AddMenuItem("Install update", "Install the available WoW Markets Companion update")
	update.Hide()
	windowsStatusItem.Lock()
	windowsStatusItem.status = status
	windowsStatusItem.update = update
	windowsStatusItem.Unlock()
	updateStatusItem(app.updaterSnapshot())
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit WoW Markets Companion", "Stop WoW Markets Companion")

	go func() {
		for {
			select {
			case <-show.ClickedCh:
				app.ShowWindow()
			case <-update.ClickedCh:
				if err := app.InstallUpdate(); err != nil {
					app.setError(err)
				}
			case <-quit.ClickedCh:
				app.Quit()
				return
			}
		}
	}()
}

func updateStatusItem(snapshot UpdaterSnapshot) {
	windowsStatusItem.Lock()
	status := windowsStatusItem.status
	item := windowsStatusItem.update
	windowsStatusItem.Unlock()
	if item == nil || status == nil {
		return
	}
	available := snapshot.AvailableVersion != "" && (snapshot.Status == updateStatusAvailable || snapshot.Status == updateStatusDownloading || snapshot.Status == updateStatusReady || snapshot.Status == updateStatusDeferred)
	if !available {
		systray.SetTooltip("WoW Markets Companion is running")
		status.SetTitle("Status: Running")
		item.Hide()
		return
	}
	systray.SetTooltip("WoW Markets Companion update " + snapshot.AvailableVersion + " is available")
	status.SetTitle("Status: Update " + snapshot.AvailableVersion + " available")
	if !snapshot.ReadyToInstall {
		item.Hide()
		return
	}
	item.SetTitle("Install update " + snapshot.AvailableVersion)
	item.Show()
}
