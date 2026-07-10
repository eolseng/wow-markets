package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

func TestFetchUpdateVerifiesFeedAndSelectsImmutableTarget(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("installer")
	feed := signedUpdateFeed(t, privateKey, "1.0.1", "macos", "arm64", len(artifact),
		base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
		"https://github.com/eolseng/wow-markets/releases/download/companion-v1.0.1/wow-markets-companion-macos-arm64.dmg")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(feed)
	}))
	defer server.Close()

	configuration := updateConfiguration{
		FeedURL:   server.URL,
		PublicKey: publicKey,
		Target:    updatefeed.TargetMacOSARM64,
		AssetName: "wow-markets-companion-macos-arm64.dmg",
		Version:   "1.0.0",
	}
	release, err := fetchUpdate(context.Background(), server.Client(), configuration)
	if err != nil {
		t.Fatalf("fetchUpdate() error = %v", err)
	}
	if release == nil || release.Version != "1.0.1" {
		t.Fatalf("fetchUpdate() = %#v, want 1.0.1", release)
	}
}

func TestFetchUpdateRejectsTamperedAndMutableMetadata(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte("installer")))
	tests := []struct {
		name     string
		assetURL string
		tamper   bool
	}{
		{name: "tampered feed", assetURL: "https://github.com/eolseng/wow-markets/releases/download/companion-v1.0.1/wow-markets-companion-macos-arm64.dmg", tamper: true},
		{name: "mutable asset", assetURL: "https://github.com/eolseng/wow-markets/releases/latest/download/wow-markets-companion-macos-arm64.dmg"},
		{name: "wrong repository", assetURL: "https://github.com/example/wow-markets/releases/download/companion-v1.0.1/wow-markets-companion-macos-arm64.dmg"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			feed := signedUpdateFeed(t, privateKey, "1.0.1", "macos", "arm64", len("installer"), signature, test.assetURL)
			if test.tamper {
				feed[20] ^= 1
			}
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				_, _ = response.Write(feed)
			}))
			defer server.Close()
			_, err := fetchUpdate(context.Background(), server.Client(), updateConfiguration{
				FeedURL:   server.URL,
				PublicKey: publicKey,
				Target:    updatefeed.TargetMacOSARM64,
				AssetName: "wow-markets-companion-macos-arm64.dmg",
				Version:   "1.0.0",
			})
			if err == nil {
				t.Fatal("fetchUpdate() accepted untrusted metadata")
			}
		})
	}
}

func TestDownloadUpdateVerifiesBeforeCommitting(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("verified Windows installer")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(artifact)
	}))
	defer server.Close()
	staging := t.TempDir()
	release := updatefeed.Release{
		URL:       server.URL,
		Length:    int64(len(artifact)),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
	}
	path, err := downloadUpdate(context.Background(), server.Client(), updateConfiguration{
		PublicKey:  publicKey,
		AssetName:  "setup.exe",
		StagingDir: staging,
	}, release)
	if err != nil {
		t.Fatalf("downloadUpdate() error = %v", err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != string(artifact) {
		t.Fatalf("staged payload = %q", payload)
	}

	release.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if _, err := downloadUpdate(context.Background(), server.Client(), updateConfiguration{
		PublicKey: publicKey, AssetName: "invalid.exe", StagingDir: staging,
	}, release); err == nil {
		t.Fatal("downloadUpdate() accepted an invalid artifact signature")
	}
	if _, err := os.Stat(filepath.Join(staging, "invalid.exe")); !os.IsNotExist(err) {
		t.Fatalf("invalid update was committed: %v", err)
	}
}

func TestDownloadUpdateRejectsInterruptedTransfer(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("partial")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(artifact)
	}))
	defer server.Close()
	_, err = downloadUpdate(context.Background(), server.Client(), updateConfiguration{
		PublicKey: publicKey, AssetName: "setup.exe", StagingDir: t.TempDir(),
	}, updatefeed.Release{
		URL:       server.URL,
		Length:    int64(len(artifact) + 10),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
	})
	if err == nil || !strings.Contains(err.Error(), "expected") {
		t.Fatalf("downloadUpdate() error = %v, want length failure", err)
	}
}

func TestUpdateOriginTrustPolicy(t *testing.T) {
	if _, err := validateUpdateOrigin(defaultUpdateOrigin, true); err != nil {
		t.Fatalf("official origin rejected: %v", err)
	}
	for _, value := range []string{
		"http://updates.wowmarkets.app",
		"https://updates.wowmarkets.app.example.com",
		"https://updates.wowmarkets.app/path",
	} {
		if _, err := validateUpdateOrigin(value, true); err == nil {
			t.Fatalf("official origin %q accepted", value)
		}
	}
	if _, err := validateUpdateOrigin("http://127.0.0.1:8788", false); err != nil {
		t.Fatalf("loopback development origin rejected: %v", err)
	}
}

func TestOfflineCheckIsRetryableAndDoesNotChangeApplicationReadiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	app := &App{
		running: true,
		updater: UpdaterSnapshot{Enabled: true, Status: updateStatusChecking},
	}
	_, err := fetchUpdate(context.Background(), server.Client(), updateConfiguration{
		FeedURL:   server.URL,
		PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize),
		Target:    updatefeed.TargetMacOSARM64,
		AssetName: "wow-markets-companion-macos-arm64.dmg",
		Version:   "1.0.0",
	})
	if err == nil || !isTransportError(err) {
		t.Fatalf("fetchUpdate() error = %v, want transport failure", err)
	}
	if !app.running {
		t.Fatal("offline update check stopped the scan watcher")
	}
}

func TestDeferralPersistsAndMandatoryUpdatesCannotBeDeferred(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := &App{
		configWritable: true,
		config:         companionConfig{UpdateChannel: "stable"},
		updater: UpdaterSnapshot{
			AvailableVersion: "1.1.0",
			Channel:          "stable",
			CurrentVersion:   "1.0.0",
			Enabled:          true,
			Status:           updateStatusReady,
		},
	}
	state, err := app.DeferUpdate()
	if err != nil {
		t.Fatalf("DeferUpdate() error = %v", err)
	}
	if state.Status != updateStatusDeferred || app.config.DeferredUpdateVersion != "1.1.0" {
		t.Fatalf("deferred state = %#v, config = %#v", state, app.config)
	}
	loaded, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeferredUpdateVersion != "1.1.0" {
		t.Fatalf("persisted deferred version = %q", loaded.DeferredUpdateVersion)
	}

	app.updater.Mandatory = true
	if _, err := app.DeferUpdate(); err == nil {
		t.Fatal("DeferUpdate() accepted a mandatory update")
	}
}

func signedUpdateFeed(t *testing.T, privateKey ed25519.PrivateKey, version, osName, arch string, length int, artifactSignature, assetURL string) []byte {
	t.Helper()
	content := []byte(fmt.Sprintf(`<?xml version="1.0"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle" xmlns:wow="https://wowmarkets.app/xml-namespaces/update"><channel><item>
<title>%s</title><sparkle:version>%s</sparkle:version>
<enclosure url="%s" length="%d" sparkle:edSignature="%s" sparkle:os="%s" wow:arch="%s" />
</item></channel></rss>
`, version, version, assetURL, length, artifactSignature, osName, arch))
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content))
	return []byte(string(content) + fmt.Sprintf("<!-- sparkle-signatures:\nedSignature: %s\nlength: %d\n-->\n", signature, len(content)))
}
