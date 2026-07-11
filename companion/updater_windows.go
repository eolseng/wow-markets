//go:build windows

package main

import (
	"errors"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
	"golang.org/x/sys/windows"
)

type windowsPlatformUpdater struct{}

func newPlatformUpdater() platformUpdater { return windowsPlatformUpdater{} }

func platformUpdateTarget() (updatefeed.Target, string, bool) {
	return updatefeed.TargetWindowsAMD64, "wow-markets-companion-windows-amd64-setup.exe", true
}

func (windowsPlatformUpdater) Start(string) error      { return nil }
func (windowsPlatformUpdater) SetFeedURL(string) error { return nil }
func (windowsPlatformUpdater) Check() error            { return nil }
func (windowsPlatformUpdater) Close()                  {}
func (windowsPlatformUpdater) ManagesDownloads() bool  { return false }

func (windowsPlatformUpdater) Install(path string) error {
	if path == "" {
		return errors.New("staged Windows installer path is empty")
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	installer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, installer, nil, nil, windows.SW_SHOWNORMAL)
}
