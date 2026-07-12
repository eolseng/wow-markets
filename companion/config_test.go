package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredServiceEndpointsDefaultToLoopback(t *testing.T) {
	t.Setenv("WOW_MARKETS_API_URL", "")
	t.Setenv("WOW_MARKETS_INSTALLATIONS_URL", "")
	previousAPIURL := officialAPIURL
	previousInstallationsURL := officialInstallationsURL
	officialAPIURL = ""
	officialInstallationsURL = ""
	t.Cleanup(func() {
		officialAPIURL = previousAPIURL
		officialInstallationsURL = previousInstallationsURL
	})

	actual := configuredServiceEndpoints()
	if actual.APIURL != developmentAPIURL || actual.InstallationsURL != developmentInstallationsURL {
		t.Fatalf("configuredServiceEndpoints() = %#v", actual)
	}
}

func TestCompanionVersionComesFromWailsConfig(t *testing.T) {
	if actual := companionVersion(); actual != "1.0.1" {
		t.Fatalf("companionVersion() = %q, want %q", actual, "1.0.1")
	}
}

func TestUpdatePreferencesPersistWithExistingConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	config := companionConfig{
		DeferredUpdateVersion: "1.1.0",
		ScanFilePath:          "/tmp/scan.lua",
		TokenPrefix:           "wms1_example",
		UpdateChannel:         "beta",
		WowInstallPath:        "/tmp/wow",
	}
	if err := saveConfig(config); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}
	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if loaded != config {
		t.Fatalf("loadConfig() = %#v, want %#v", loaded, config)
	}
}

func TestConfiguredServiceEndpointsRequireExplicitDevelopmentOverride(t *testing.T) {
	t.Setenv("WOW_MARKETS_API_URL", "https://example.invalid/")
	t.Setenv("WOW_MARKETS_INSTALLATIONS_URL", "https://example.invalid/installations")
	previousAPIURL := officialAPIURL
	previousInstallationsURL := officialInstallationsURL
	officialAPIURL = ""
	officialInstallationsURL = ""
	t.Cleanup(func() {
		officialAPIURL = previousAPIURL
		officialInstallationsURL = previousInstallationsURL
	})

	actual := configuredServiceEndpoints()
	if actual.APIURL != "https://example.invalid" ||
		actual.InstallationsURL != "https://example.invalid/installations" {
		t.Fatalf("configuredServiceEndpoints() = %#v", actual)
	}
}

func TestConfiguredServiceEndpointsPreferOfficialBuildValues(t *testing.T) {
	t.Setenv("WOW_MARKETS_API_URL", "https://ignored.invalid")
	t.Setenv("WOW_MARKETS_INSTALLATIONS_URL", "https://ignored.invalid/installations")
	previousAPIURL := officialAPIURL
	previousInstallationsURL := officialInstallationsURL
	officialAPIURL = "https://api.wowmarkets.app/"
	officialInstallationsURL = "https://wowmarkets.app/account/contribute"
	t.Cleanup(func() {
		officialAPIURL = previousAPIURL
		officialInstallationsURL = previousInstallationsURL
	})

	actual := configuredServiceEndpoints()
	if actual.APIURL != "https://api.wowmarkets.app" ||
		actual.InstallationsURL != "https://wowmarkets.app/account/contribute" {
		t.Fatalf("configuredServiceEndpoints() = %#v", actual)
	}
}

func TestMigrateCompanionConfigDir(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, legacyConfigDirName)
	if err := os.MkdirAll(filepath.Join(legacy, "data", "scans"), 0o700); err != nil {
		t.Fatalf("create legacy data directory: %v", err)
	}
	marker := filepath.Join(legacy, "data", "scans", "archive.json.gz")
	if err := os.WriteFile(marker, []byte("archive"), 0o600); err != nil {
		t.Fatalf("write legacy archive marker: %v", err)
	}

	current, err := migrateCompanionConfigDir(root)
	if err != nil {
		t.Fatalf("migrateCompanionConfigDir() error = %v", err)
	}
	if current != filepath.Join(root, configDirName) {
		t.Fatalf("current directory = %q", current)
	}
	if _, err := os.Stat(filepath.Join(current, "data", "scans", "archive.json.gz")); err != nil {
		t.Fatalf("migrated archive missing: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy directory still exists: %v", err)
	}
}

func TestMigrateCompanionConfigDirPrefersExistingCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, configDirName)
	legacy := filepath.Join(root, legacyConfigDirName)
	if err := os.MkdirAll(current, 0o700); err != nil {
		t.Fatalf("create current directory: %v", err)
	}
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatalf("create legacy directory: %v", err)
	}

	got, err := migrateCompanionConfigDir(root)
	if err != nil {
		t.Fatalf("migrateCompanionConfigDir() error = %v", err)
	}
	if got != current {
		t.Fatalf("current directory = %q, want %q", got, current)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy directory should remain when current exists: %v", err)
	}
}
