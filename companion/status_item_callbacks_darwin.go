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

//export wmsStatusItemQuit
func wmsStatusItemQuit() {
	if app := currentDarwinStatusItemApp(); app != nil {
		app.Quit()
	}
}

//export wmsStatusItemInstallUpdate
func wmsStatusItemInstallUpdate() {
	if app := currentDarwinStatusItemApp(); app != nil {
		if err := app.InstallUpdate(); err != nil {
			app.setError(err)
		}
	}
}

//export wmsSparkleWillRelaunch
func wmsSparkleWillRelaunch() {
	if app := currentDarwinStatusItemApp(); app != nil {
		if err := app.prepareForUpdateRelaunch(); err != nil {
			app.setError(err)
		}
	}
}
