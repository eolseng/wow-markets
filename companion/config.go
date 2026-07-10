package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	productionAPIURL     = "https://api.wowmarkets.app"
	installationsPageURL = "https://wowmarkets.app/account/installations"
	configDirName        = "WowMarketScan"
	configFileName       = "config.json"
	companionVersion     = "1.0.0"
)

type companionConfig struct {
	// Email and InstallationName are retained so pre-1.0 config files continue
	// to load. Account sessions are no longer part of the companion setup.
	Email            string `json:"email,omitempty"`
	InstallationName string `json:"installation_name,omitempty"`
	ScanFilePath     string `json:"scan_file_path,omitempty"`
	TokenPrefix      string `json:"token_prefix,omitempty"`
	WowInstallPath   string `json:"wow_install_path,omitempty"`
}

func loadConfig() (companionConfig, error) {
	path, err := configPath()
	if err != nil {
		return companionConfig{}, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return companionConfig{}, nil
	}
	if err != nil {
		return companionConfig{}, err
	}
	defer file.Close()

	var config companionConfig
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return companionConfig{}, err
	}
	config.Email = strings.TrimSpace(config.Email)
	config.InstallationName = strings.TrimSpace(config.InstallationName)
	config.ScanFilePath = strings.TrimSpace(config.ScanFilePath)
	config.TokenPrefix = strings.TrimSpace(config.TokenPrefix)
	config.WowInstallPath = strings.TrimSpace(config.WowInstallPath)
	return config, nil
}

func saveConfig(config companionConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o600)
}

func configPath() (string, error) {
	dir, err := companionConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func companionConfigDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(root, configDirName), nil
}

func companionDataDir() (string, error) {
	dir, err := companionConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data"), nil
}
