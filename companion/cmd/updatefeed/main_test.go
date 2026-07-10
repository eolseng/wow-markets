package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

func TestGenerateCreatesVerifiablePlatformAppcasts(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	keyPath := filepath.Join(root, "private.key")
	assetsDir := filepath.Join(root, "assets")
	outputDir := filepath.Join(root, "output")
	if err := os.MkdirAll(assetsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(privateKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"wow-markets-companion-macos-arm64.dmg":         "macos artifact",
		"wow-markets-companion-windows-amd64-setup.exe": "windows artifact",
	} {
		if err := os.WriteFile(filepath.Join(assetsDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := generate([]string{
		"--private-key", keyPath,
		"--version", "1.1.0-beta.1",
		"--channel", "beta",
		"--published-at", "2026-07-11T00:00:00Z",
		"--assets-dir", assetsDir,
		"--output-dir", outputDir,
	}); err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	for name, target := range map[string]updatefeed.Target{
		"companion-beta-macos-arm64.xml":   updatefeed.TargetMacOSARM64,
		"companion-beta-windows-amd64.xml": updatefeed.TargetWindowsAMD64,
	} {
		payload, err := os.ReadFile(filepath.Join(outputDir, name))
		if err != nil {
			t.Fatal(err)
		}
		releases, err := updatefeed.ParseSigned(payload, publicKey)
		if err != nil {
			t.Fatalf("ParseSigned(%s) error = %v", name, err)
		}
		selected, err := updatefeed.Select(releases, "1.0.0", target)
		if err != nil || selected == nil || selected.Version != "1.1.0-beta.1" {
			t.Fatalf("Select(%s) = %#v, %v", name, selected, err)
		}
	}
}
