//go:build !darwin && !windows

package main

func registerStatusItem(app *App) {}

func startStatusItem(app *App) {}

func stopStatusItem() {}

func activateVisibleWindow() {}

func updateStatusItem(UpdaterSnapshot) {}
