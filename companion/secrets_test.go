package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestDeleteInstallationTokenRemovesCurrentAndLegacyCredentials(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set(keyringService, keyringInstallationToken, "current"); err != nil {
		t.Fatal(err)
	}
	if err := keyring.Set(legacyKeyringService, keyringInstallationToken, "legacy"); err != nil {
		t.Fatal(err)
	}

	if err := deleteInstallationToken(); err != nil {
		t.Fatal(err)
	}
	for _, service := range []string{keyringService, legacyKeyringService} {
		if _, err := keyring.Get(service, keyringInstallationToken); !errors.Is(err, keyring.ErrNotFound) {
			t.Fatalf("credential for %q still exists: %v", service, err)
		}
	}
}

func TestRemoveInstallationTokenPreservesLocalArchives(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	keyring.MockInit()
	if err := saveInstallationToken("wms1_example-token"); err != nil {
		t.Fatal(err)
	}
	config := companionConfig{TokenPrefix: "wms1_example-"}
	if err := saveConfig(config); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	archivePath := filepath.Join(dataDir, "data", "scans", "existing.json.gz")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, []byte("preserve me"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := &App{
		config:         config,
		configWritable: true,
		dataDir:        dataDir,
		token:          "wms1_example-token",
	}
	snapshot, err := app.RemoveInstallationToken()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TokenStored || app.token != "" {
		t.Fatal("token remained in application state")
	}
	storedConfig, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if storedConfig.TokenPrefix != "" {
		t.Fatalf("stored token hint = %q, want empty", storedConfig.TokenPrefix)
	}
	payload, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "preserve me" {
		t.Fatalf("archive contents changed to %q", payload)
	}
}
