package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/eolseng/wow-markets/companion/internal/wowinstall"
)

func TestPendingShowSurvivesStartupContextAttachment(t *testing.T) {
	app := &App{}
	app.ShowWindow()

	ctx := context.Background()
	showWindow, quit := app.attachStartupContext(ctx)
	if !showWindow {
		t.Fatal("ShowWindow() request was lost before the Wails context was attached")
	}
	if quit {
		t.Fatal("fresh app unexpectedly requested quit")
	}
	if app.ctx != ctx {
		t.Fatal("startup context was not attached")
	}
}

func TestPendingQuitSurvivesStartupContextAttachment(t *testing.T) {
	app := &App{quitting: true}

	_, quit := app.attachStartupContext(context.Background())
	if !quit {
		t.Fatal("Quit() request was lost before the Wails context was attached")
	}
}

func TestSparkleRelaunchAllowsExternalQuit(t *testing.T) {
	app := &App{dataDir: t.TempDir(), windowShown: true}
	if err := app.prepareForUpdateRelaunch(); err != nil {
		t.Fatal(err)
	}
	if app.beforeClose(context.Background()) {
		t.Fatal("Sparkle install-and-relaunch request was treated as a window hide")
	}
}

func TestUpdateRelaunchAlwaysShowsWindow(t *testing.T) {
	app := &App{dataDir: t.TempDir(), windowShown: false}
	if err := app.prepareForUpdateRelaunch(); err != nil {
		t.Fatal(err)
	}
	visible, found, err := consumeUpdateRelaunchVisibility(app.dataDir)
	if err != nil || !found || !visible {
		t.Fatalf("consume visibility = %v, %v, %v; want true, true, nil", visible, found, err)
	}
	if _, foundAgain, err := consumeUpdateRelaunchVisibility(app.dataDir); err != nil || foundAgain {
		t.Fatalf("consumed state persisted: found=%v err=%v", foundAgain, err)
	}
}

func TestBackgroundLaunchDoesNotRefreshStartupRegistration(t *testing.T) {
	if shouldRefreshLaunchAtLogin(true, true) {
		t.Fatal("background launch would rewrite the startup registration")
	}
	if !shouldRefreshLaunchAtLogin(true, false) {
		t.Fatal("manual launch should repair an enabled startup registration")
	}
	if shouldRefreshLaunchAtLogin(false, false) {
		t.Fatal("disabled startup registration should not be created during initialization")
	}
}

func TestCurrentStepUsesTokenOnlySetupOrder(t *testing.T) {
	tests := []struct {
		name     string
		inputs   [5]bool
		expected string
	}{
		{name: "loading", inputs: [5]bool{true, false, false, false, false}, expected: "loading"},
		{name: "token", inputs: [5]bool{false, false, true, true, true}, expected: "token"},
		{name: "wow", inputs: [5]bool{false, true, false, true, true}, expected: "wow"},
		{name: "addon", inputs: [5]bool{false, true, true, false, true}, expected: "addon"},
		{name: "saved variables", inputs: [5]bool{false, true, true, true, false}, expected: "saved_variables"},
		{name: "ready", inputs: [5]bool{false, true, true, true, true}, expected: "ready"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := currentStep(test.inputs[0], test.inputs[1], test.inputs[2], test.inputs[3], test.inputs[4])
			if actual != test.expected {
				t.Fatalf("currentStep%v = %q, want %q", test.inputs, actual, test.expected)
			}
		})
	}
}

func TestSnapshotIncludesAddonDistributionDetails(t *testing.T) {
	app := &App{addonDetected: true, addonPath: "/WoWMarkets.toc", addonVersion: "0.5.0-beta.1"}
	snapshot := app.Snapshot()
	if snapshot.AddonVersion != "0.5.0-beta.1" || snapshot.AddonPath != "/WoWMarkets.toc" {
		t.Fatalf("addon snapshot = %+v", snapshot)
	}
	if snapshot.AddonCurseForgeURL != addonCurseForgeURL || snapshot.AddonWagoURL != addonWagoURL {
		t.Fatalf("distribution URLs = %q, %q", snapshot.AddonCurseForgeURL, snapshot.AddonWagoURL)
	}
}

func TestMissingAddonRefreshDoesNotRediscoverSavedVariables(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "_anniversary_"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := wowinstall.Candidate{Account: "ACCOUNT", Path: "/already/discovered.lua"}
	app := &App{
		config:      companionConfig{WowInstallPath: root, ScanFilePath: existing.Path},
		discoveries: []wowinstall.Candidate{existing},
		wowDetected: true,
	}

	snapshot, err := app.refreshMissingAddonLocked()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.AddonDetected {
		t.Fatal("missing addon was detected")
	}
	if snapshot.AddonPath != wowinstall.AddonMarkerPath(root) {
		t.Fatalf("addon path = %q, want %q", snapshot.AddonPath, wowinstall.AddonMarkerPath(root))
	}
	if snapshot.ScanFilePath != existing.Path || len(snapshot.Discoveries) != 1 {
		t.Fatalf("cheap addon refresh changed SavedVariables state: %+v", snapshot)
	}
}

func TestUnchangedSavedVariablesRefreshDoesNotReparseFiles(t *testing.T) {
	root := t.TempDir()
	savedVariables := filepath.Join(root, "_anniversary_", "WTF", "Account", "ACCOUNT", "SavedVariables")
	if err := os.MkdirAll(savedVariables, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(wowinstall.AddonMarkerPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wowinstall.AddonMarkerPath(root), []byte("## Version: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(savedVariables, "WoWMarkets.lua"), []byte("not parseable"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := &App{config: companionConfig{WowInstallPath: root}, wowDetected: true, addonDetected: true}
	if _, err := app.refreshMissingSavedVariablesLocked(); err != nil {
		t.Fatal(err)
	}

	sentinel := wowinstall.Candidate{Account: "sentinel", Path: "/not/from/disk.lua"}
	app.mu.Lock()
	app.discoveries = []wowinstall.Candidate{sentinel}
	app.mu.Unlock()
	snapshot, err := app.refreshMissingSavedVariablesLocked()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Discoveries) != 1 || snapshot.Discoveries[0].Path != sentinel.Path {
		t.Fatalf("unchanged metadata triggered SavedVariables rediscovery: %+v", snapshot.Discoveries)
	}
}

func TestNormalizeInstallationToken(t *testing.T) {
	secret := make([]byte, installationTokenSecretBytes)
	for index := range secret {
		secret[index] = byte(index)
	}
	valid := installationTokenVersionPrefix + base64.RawURLEncoding.EncodeToString(secret)
	actual, err := normalizeInstallationToken("  " + valid + "\n")
	if err != nil {
		t.Fatalf("normalizeInstallationToken() error = %v", err)
	}
	if actual != valid {
		t.Fatalf("normalizeInstallationToken() = %q, want %q", actual, valid)
	}
	if prefix := installationTokenPrefix(actual); prefix != actual[:installationTokenHintLength] {
		t.Fatalf("installationTokenPrefix() = %q", prefix)
	}

	for _, invalid := range []string{"", "abc", "wms1_short", valid + "="} {
		if _, err := normalizeInstallationToken(invalid); err == nil {
			t.Fatalf("normalizeInstallationToken(%q) succeeded", invalid)
		}
	}
}
