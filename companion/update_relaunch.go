package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const updateRelaunchStateFileName = ".update-relaunch-window"

func updateRelaunchStatePath(dataDir string) string {
	return filepath.Join(dataDir, updateRelaunchStateFileName)
}

func persistUpdateRelaunchVisibility(dataDir string, visible bool) error {
	if strings.TrimSpace(dataDir) == "" {
		return errors.New("companion data directory is unavailable")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	value := "hidden\n"
	if visible {
		value = "visible\n"
	}
	return os.WriteFile(updateRelaunchStatePath(dataDir), []byte(value), 0o600)
}

func consumeUpdateRelaunchVisibility(dataDir string) (visible, found bool, err error) {
	payload, err := os.ReadFile(updateRelaunchStatePath(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if err := os.Remove(updateRelaunchStatePath(dataDir)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, false, err
	}
	switch strings.TrimSpace(string(payload)) {
	case "visible":
		return true, true, nil
	case "hidden":
		return false, true, nil
	default:
		return false, false, nil
	}
}

func clearUpdateRelaunchVisibility(dataDir string) error {
	err := os.Remove(updateRelaunchStatePath(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
