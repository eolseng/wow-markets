package main

import (
	"context"
	"encoding/base64"
	"testing"
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

func TestUpdateRelaunchRestoresPreviousWindowVisibility(t *testing.T) {
	for _, visible := range []bool{true, false} {
		dataDir := t.TempDir()
		if err := persistUpdateRelaunchVisibility(dataDir, visible); err != nil {
			t.Fatal(err)
		}
		actual, found, err := consumeUpdateRelaunchVisibility(dataDir)
		if err != nil || !found || actual != visible {
			t.Fatalf("consume visibility = %v, %v, %v; want %v, true, nil", actual, found, err, visible)
		}
		if _, foundAgain, err := consumeUpdateRelaunchVisibility(dataDir); err != nil || foundAgain {
			t.Fatalf("consumed state persisted: found=%v err=%v", foundAgain, err)
		}
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
