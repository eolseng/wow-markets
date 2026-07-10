package wowinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const addonMarkerRelativePath = "Interface/AddOns/WowMarketScan/WowMarketScan.toc"

// InstallInspection describes the local pieces needed by the companion. A
// missing Anniversary client, addon, or SavedVariables file is represented by
// fields in this value rather than an error.
type InstallInspection struct {
	InstallPath        string      `json:"install_path"`
	AnniversaryPath    string      `json:"anniversary_path"`
	AnniversaryPresent bool        `json:"anniversary_present"`
	AddonMarkerPath    string      `json:"addon_marker_path"`
	AddonPresent       bool        `json:"addon_present"`
	ScanFiles          []Candidate `json:"scan_files"`
}

// FindInstallRoot returns the first WoW root containing an Anniversary client.
// A valid configured root wins over a root inferred from WOW_MARKET_SCAN_FILE
// and the platform's standard install locations.
func FindInstallRoot(configuredRoot string) (string, bool) {
	roots := make([]string, 0, 2)
	roots = append(roots, configuredRoot)
	if environmentRoot := inferInstallRoot(os.Getenv("WOW_MARKET_SCAN_FILE")); environmentRoot != "" {
		roots = append(roots, environmentRoot)
	}
	roots = append(roots, candidateRoots()...)

	for _, root := range dedupeStrings(roots) {
		root = NormalizeInstallRoot(root)
		present, err := AnniversaryInstalled(root)
		if err == nil && present {
			return root, true
		}
	}
	return "", false
}

// InspectInstall inspects one selected WoW root without searching elsewhere.
// Only an invalid or unreadable root (or another filesystem failure) is an
// error; missing install components are valid inspection results.
func InspectInstall(root string) (InstallInspection, error) {
	root = NormalizeInstallRoot(root)
	inspection := InstallInspection{
		InstallPath:     root,
		AnniversaryPath: AnniversaryPath(root),
		AddonMarkerPath: AddonMarkerPath(root),
		ScanFiles:       []Candidate{},
	}
	if err := validateDirectory(root, "World of Warcraft folder"); err != nil {
		return inspection, err
	}

	anniversaryPresent, err := AnniversaryInstalled(root)
	if err != nil {
		return inspection, err
	}
	inspection.AnniversaryPresent = anniversaryPresent

	addonPresent, err := AddonInstalled(root)
	if err != nil {
		return inspection, err
	}
	inspection.AddonPresent = addonPresent

	scanFiles, err := discoverAnniversaryScanFilesInRoot(root)
	if err != nil {
		return inspection, err
	}
	inspection.ScanFiles = scanFiles
	return inspection, nil
}

// AnniversaryPath returns the Anniversary client directory for a WoW root.
func AnniversaryPath(root string) string {
	root = NormalizeInstallRoot(root)
	if root == "" {
		return ""
	}
	return filepath.Join(root, anniversaryFolder)
}

// AnniversaryInstalled reports whether root contains an Anniversary client
// directory. A missing directory is not an error.
func AnniversaryInstalled(root string) (bool, error) {
	path := AnniversaryPath(root)
	if path == "" {
		return false, nil
	}
	return directoryExists(path)
}

// AddonMarkerPath returns the expected WowMarketScan TOC path for a WoW root.
func AddonMarkerPath(root string) string {
	anniversaryPath := AnniversaryPath(root)
	if anniversaryPath == "" {
		return ""
	}
	return filepath.Join(anniversaryPath, filepath.FromSlash(addonMarkerRelativePath))
}

// AddonInstalled reports whether the WowMarketScan TOC marker exists as a
// file. A missing marker is not an error.
func AddonInstalled(root string) (bool, error) {
	path := AddonMarkerPath(root)
	if path == "" {
		return false, nil
	}
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect addon marker %s: %w", path, err)
}

func directoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect folder %s: %w", path, err)
}

func validateDirectory(path, label string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%s is required", label)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect %s %s: %w", label, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %s is not a folder", label, path)
	}
	return nil
}
