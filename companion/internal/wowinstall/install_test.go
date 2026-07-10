package wowinstall

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindInstallRootPrefersConfiguredThenEnvironment(t *testing.T) {
	configuredRoot := t.TempDir()
	environmentRoot := t.TempDir()
	mkdirAll(t, AnniversaryPath(configuredRoot))
	mkdirAll(t, AnniversaryPath(environmentRoot))

	environmentScan := scanPath(environmentRoot, "ENVIRONMENT")
	t.Setenv("WOW_MARKET_SCAN_FILE", environmentScan)

	got, ok := FindInstallRoot(filepath.Join(configuredRoot, anniversaryFolder))
	if !ok {
		t.Fatal("FindInstallRoot() did not find configured root")
	}
	if got != configuredRoot {
		t.Fatalf("FindInstallRoot() = %q, want configured root %q", got, configuredRoot)
	}

	missingRoot := filepath.Join(t.TempDir(), "missing")
	got, ok = FindInstallRoot(missingRoot)
	if !ok {
		t.Fatal("FindInstallRoot() did not find environment root")
	}
	if got != environmentRoot {
		t.Fatalf("FindInstallRoot() = %q, want environment root %q", got, environmentRoot)
	}
}

func TestInspectInstallRepresentsMissingComponentsWithoutErrors(t *testing.T) {
	root := t.TempDir()

	inspection, err := InspectInstall(root)
	if err != nil {
		t.Fatalf("InspectInstall() with no Anniversary client error = %v", err)
	}
	if inspection.AnniversaryPresent || inspection.AddonPresent || len(inspection.ScanFiles) != 0 {
		t.Fatalf("InspectInstall() with no components = %+v", inspection)
	}
	if inspection.AnniversaryPath != filepath.Join(root, anniversaryFolder) {
		t.Fatalf("AnniversaryPath = %q", inspection.AnniversaryPath)
	}
	if inspection.AddonMarkerPath != filepath.Join(root, anniversaryFolder, "Interface", "AddOns", "WowMarketScan", "WowMarketScan.toc") {
		t.Fatalf("AddonMarkerPath = %q", inspection.AddonMarkerPath)
	}

	mkdirAll(t, AnniversaryPath(root))
	inspection, err = InspectInstall(root)
	if err != nil {
		t.Fatalf("InspectInstall() with no addon or SavedVariables error = %v", err)
	}
	if !inspection.AnniversaryPresent || inspection.AddonPresent || len(inspection.ScanFiles) != 0 {
		t.Fatalf("InspectInstall() with only Anniversary client = %+v", inspection)
	}
}

func TestInspectInstallDetectsAddonAndValidScanFiles(t *testing.T) {
	root := t.TempDir()
	marker := AddonMarkerPath(root)
	mkdirAll(t, filepath.Dir(marker))
	if err := os.WriteFile(marker, []byte("## Interface: 20504\n"), 0o644); err != nil {
		t.Fatalf("write addon marker: %v", err)
	}
	validScan := writeValidScan(t, root, "ACCOUNT-ONE")
	invalidScan := scanPath(root, "ACCOUNT-TWO")
	mkdirAll(t, filepath.Dir(invalidScan))
	if err := os.WriteFile(invalidScan, []byte("not SavedVariables"), 0o644); err != nil {
		t.Fatalf("write invalid scan: %v", err)
	}

	inspection, err := InspectInstall(root)
	if err != nil {
		t.Fatalf("InspectInstall() error = %v", err)
	}
	if !inspection.AnniversaryPresent || !inspection.AddonPresent {
		t.Fatalf("InspectInstall() = %+v", inspection)
	}
	if len(inspection.ScanFiles) != 1 || inspection.ScanFiles[0].Path != validScan {
		t.Fatalf("ScanFiles = %+v, want only %q", inspection.ScanFiles, validScan)
	}
	if inspection.ScanFiles[0].Account != "ACCOUNT-ONE" {
		t.Fatalf("Account = %q, want ACCOUNT-ONE", inspection.ScanFiles[0].Account)
	}
}

func TestDiscoverAnniversaryScanFilesInRootDoesNotSearchElsewhere(t *testing.T) {
	selectedRoot := t.TempDir()
	otherRoot := t.TempDir()
	selectedScan := writeValidScan(t, selectedRoot, "SELECTED")
	otherScan := writeValidScan(t, otherRoot, "OTHER")
	t.Setenv("WOW_MARKET_SCAN_FILE", otherScan)

	candidates, err := DiscoverAnniversaryScanFilesInRoot(selectedRoot)
	if err != nil {
		t.Fatalf("DiscoverAnniversaryScanFilesInRoot() error = %v", err)
	}
	if len(candidates) != 1 || candidates[0].Path != selectedScan {
		t.Fatalf("candidates = %+v, want only %q", candidates, selectedScan)
	}
}

func TestDiscoverAnniversaryScanFilesKeepsExplicitEnvironmentCandidateFirst(t *testing.T) {
	root := t.TempDir()
	environmentRoot := t.TempDir()
	rootScan := writeValidScan(t, root, "ROOT")
	environmentScan := writeValidScan(t, environmentRoot, "ENVIRONMENT")
	now := time.Now()
	if err := os.Chtimes(rootScan, now, now); err != nil {
		t.Fatalf("set root scan time: %v", err)
	}
	if err := os.Chtimes(environmentScan, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatalf("set environment scan time: %v", err)
	}
	t.Setenv("WOW_MARKET_SCAN_FILE", environmentScan)

	candidates, err := DiscoverAnniversaryScanFiles(root)
	if err != nil {
		t.Fatalf("DiscoverAnniversaryScanFiles() error = %v", err)
	}
	if len(candidates) < 2 {
		t.Fatalf("len(candidates) = %d, want at least 2", len(candidates))
	}
	if candidates[0].Path != environmentScan {
		t.Fatalf("first candidate = %q, want explicit environment file %q", candidates[0].Path, environmentScan)
	}
	if !candidatePathPresent(candidates, rootScan) {
		t.Fatalf("candidates = %+v, missing root scan %q", candidates, rootScan)
	}
}

func TestInspectInstallRejectsInvalidRoot(t *testing.T) {
	if _, err := InspectInstall(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("InspectInstall() error = nil for missing root")
	}
}

func writeValidScan(t *testing.T, root, account string) string {
	t.Helper()
	fixture, err := os.ReadFile("../../testdata/WowMarketScan.lua")
	if err != nil {
		t.Fatalf("read scan fixture: %v", err)
	}
	path := scanPath(root, account)
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, fixture, 0o644); err != nil {
		t.Fatalf("write scan fixture: %v", err)
	}
	return path
}

func scanPath(root, account string) string {
	return filepath.Join(root, anniversaryFolder, "WTF", "Account", account, "SavedVariables", "WowMarketScan.lua")
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create directory %s: %v", path, err)
	}
}

func candidatePathPresent(candidates []Candidate, path string) bool {
	for _, candidate := range candidates {
		if candidate.Path == path {
			return true
		}
	}
	return false
}
