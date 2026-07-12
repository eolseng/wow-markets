package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	developmentAPIURL           = "http://127.0.0.1:8787"
	developmentInstallationsURL = "http://127.0.0.1:3000/account/contribute"
	configDirName               = "WoWMarkets"
	legacyConfigDirName         = "WowMarketScan"
	configFileName              = "config.json"
)

//go:embed wails.json
var wailsConfigJSON []byte

// Official release workflows inject both values with -ldflags. Development
// builds default to loopback and can select another service explicitly through
// the environment.
var (
	officialAPIURL           string
	officialInstallationsURL string
	officialUpdateOrigin     string
)

type serviceEndpoints struct {
	APIURL           string
	InstallationsURL string
}

func companionVersion() string {
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsConfigJSON, &config); err != nil {
		panic(fmt.Sprintf("decode embedded wails.json: %v", err))
	}
	version := strings.TrimSpace(config.Info.ProductVersion)
	if version == "" {
		panic("wails.json info.productVersion is required")
	}
	return version
}

func configuredServiceEndpoints() serviceEndpoints {
	if strings.TrimSpace(officialAPIURL) != "" &&
		strings.TrimSpace(officialInstallationsURL) != "" {
		return serviceEndpoints{
			APIURL:           strings.TrimRight(strings.TrimSpace(officialAPIURL), "/"),
			InstallationsURL: strings.TrimSpace(officialInstallationsURL),
		}
	}
	apiURL := strings.TrimSpace(os.Getenv("WOW_MARKETS_API_URL"))
	if apiURL == "" {
		apiURL = developmentAPIURL
	}
	installationsURL := strings.TrimSpace(os.Getenv("WOW_MARKETS_INSTALLATIONS_URL"))
	if installationsURL == "" {
		installationsURL = developmentInstallationsURL
	}
	return serviceEndpoints{
		APIURL:           strings.TrimRight(apiURL, "/"),
		InstallationsURL: installationsURL,
	}
}

type companionConfig struct {
	// Email and InstallationName are retained so pre-1.0 config files continue
	// to load. Account sessions are no longer part of the companion setup.
	Email                 string `json:"email,omitempty"`
	InstallationName      string `json:"installation_name,omitempty"`
	DeferredUpdateVersion string `json:"deferred_update_version,omitempty"`
	ScanFilePath          string `json:"scan_file_path,omitempty"`
	TokenPrefix           string `json:"token_prefix,omitempty"`
	UpdateChannel         string `json:"update_channel,omitempty"`
	WowInstallPath        string `json:"wow_install_path,omitempty"`
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
	config.DeferredUpdateVersion = strings.TrimSpace(config.DeferredUpdateVersion)
	config.UpdateChannel = strings.ToLower(strings.TrimSpace(config.UpdateChannel))
	if config.UpdateChannel == "" {
		config.UpdateChannel = "stable"
	}
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
	return migrateCompanionConfigDir(root)
}

func migrateCompanionConfigDir(root string) (string, error) {
	current := filepath.Join(root, configDirName)
	if _, err := os.Stat(current); err == nil {
		return current, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect config directory: %w", err)
	}

	legacy := filepath.Join(root, legacyConfigDirName)
	if _, err := os.Stat(legacy); errors.Is(err, os.ErrNotExist) {
		return current, nil
	} else if err != nil {
		return "", fmt.Errorf("inspect legacy config directory: %w", err)
	}
	if err := os.Rename(legacy, current); err != nil {
		if _, currentErr := os.Stat(current); currentErr == nil {
			return current, nil
		}
		return "", fmt.Errorf("migrate legacy config directory: %w", err)
	}
	return current, nil
}

func companionDataDir() (string, error) {
	dir, err := companionConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data"), nil
}
