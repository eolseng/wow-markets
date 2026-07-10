//go:build !darwin && !windows

package main

import "errors"

func platformLaunchAtLoginSupported() bool {
	return false
}

func platformLaunchAtLoginEnabled() (bool, error) {
	return false, nil
}

func platformSetLaunchAtLogin(bool) error {
	return errors.New("launch at login is supported on macOS and Windows")
}
