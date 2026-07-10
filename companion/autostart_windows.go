//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	windowsRunKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	windowsRunValue = "WoW Markets Companion"
)

func platformLaunchAtLoginSupported() bool {
	return true
}

func platformLaunchAtLoginEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open Windows startup settings: %w", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(windowsRunValue)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Windows startup setting: %w", err)
	}
	return value != "", nil
}

func platformSetLaunchAtLogin(enabled bool) error {
	key, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		windowsRunKey,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if err != nil {
		return fmt.Errorf("open Windows startup settings: %w", err)
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(windowsRunValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("disable launch at login: %w", err)
		}
		return nil
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate companion executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return fmt.Errorf("resolve companion executable: %w", err)
	}
	command, err := windowsStartupCommand(executable)
	if err != nil {
		return err
	}
	existing, _, err := key.GetStringValue(windowsRunValue)
	if err == nil && existing == command {
		return nil
	}
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("read Windows startup setting: %w", err)
	}
	if err := key.SetStringValue(windowsRunValue, command); err != nil {
		return fmt.Errorf("enable launch at login: %w", err)
	}
	return nil
}

func windowsStartupCommand(executable string) (string, error) {
	command := `"` + executable + `" ` + backgroundLaunchArgument
	if len(command) > 260 {
		return "", errors.New("the companion path is too long for Windows startup; move it to a shorter permanent folder")
	}
	return command, nil
}
