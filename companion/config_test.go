package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
