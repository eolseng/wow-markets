package main

/*
 */
import "C"

import "sync"

var darwinStatusItem struct {
	sync.Mutex
	app *App
}

func setDarwinStatusItemApp(app *App) {
	darwinStatusItem.Lock()
	defer darwinStatusItem.Unlock()

	darwinStatusItem.app = app
}

func currentDarwinStatusItemApp() *App {
	darwinStatusItem.Lock()
	defer darwinStatusItem.Unlock()

	return darwinStatusItem.app
}

//export wmsStatusItemShowWindow
func wmsStatusItemShowWindow() {
	if app := currentDarwinStatusItemApp(); app != nil {
		app.ShowWindow()
	}
}

//export wmsStatusItemHideWindow
func wmsStatusItemHideWindow() {
	if app := currentDarwinStatusItemApp(); app != nil {
		app.HideWindow()
	}
}

//export wmsStatusItemQuit
func wmsStatusItemQuit() {
	if app := currentDarwinStatusItemApp(); app != nil {
		app.Quit()
	}
}
