package wowinstall

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanfile"
)

const anniversaryFolder = "_anniversary_"

const (
	primaryScanFileName = "WoWMarkets.lua"
	legacyScanFileName  = "WowMarketScan.lua"
	primaryScanFileEnv  = "WOW_MARKETS_SCAN_FILE"
	legacyScanFileEnv   = "WOW_MARKET_SCAN_FILE"
)

type Candidate struct {
	Account     string `json:"account"`
	InstallPath string `json:"install_path"`
	Path        string `json:"path"`
	ModifiedAt  string `json:"modified_at"`
	Size        int64  `json:"size"`
}

func DiscoverAnniversaryScanFiles(extraRoots ...string) ([]Candidate, error) {
	explicitCandidates := make([]Candidate, 0, 2)
	for _, path := range environmentScanFiles() {
		if candidate, ok := validCandidate(path, ""); ok {
			explicitCandidates = append(explicitCandidates, candidate)
		}
	}

	candidates := make([]Candidate, 0)
	roots := append([]string{}, extraRoots...)
	roots = append(roots, candidateRoots()...)
	roots = dedupeStrings(roots)
	for _, root := range roots {
		root = NormalizeInstallRoot(root)
		rootCandidates, err := discoverAnniversaryScanFilesInRoot(root)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, rootCandidates...)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339, candidates[i].ModifiedAt)
		right, _ := time.Parse(time.RFC3339, candidates[j].ModifiedAt)
		return left.After(right)
	})
	return dedupeCandidates(append(explicitCandidates, candidates...)), nil
}

// DiscoverAnniversaryScanFilesInRoot returns valid account SavedVariables
// files from root only. It does not consult environment variables or standard
// install locations.
func DiscoverAnniversaryScanFilesInRoot(root string) ([]Candidate, error) {
	root = NormalizeInstallRoot(root)
	if err := validateDirectory(root, "World of Warcraft folder"); err != nil {
		return nil, err
	}
	return discoverAnniversaryScanFilesInRoot(root)
}

func discoverAnniversaryScanFilesInRoot(root string) ([]Candidate, error) {
	currentAddonInstalled, err := AddonInstalled(root)
	if err != nil {
		return nil, err
	}
	matches := make([]string, 0)
	for _, fileName := range []string{primaryScanFileName, legacyScanFileName} {
		if currentAddonInstalled && fileName == legacyScanFileName {
			continue
		}
		fileMatches, err := filepath.Glob(filepath.Join(
			root,
			anniversaryFolder,
			"WTF",
			"Account",
			"*",
			"SavedVariables",
			fileName,
		))
		if err != nil {
			return nil, err
		}
		matches = append(matches, fileMatches...)
	}

	candidates := make([]Candidate, 0, len(matches))
	for _, match := range matches {
		candidate, ok := validCandidate(match, root)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		leftCurrent := filepath.Base(candidates[i].Path) == primaryScanFileName
		rightCurrent := filepath.Base(candidates[j].Path) == primaryScanFileName
		if leftCurrent != rightCurrent {
			return leftCurrent
		}
		left, _ := time.Parse(time.RFC3339, candidates[i].ModifiedAt)
		right, _ := time.Parse(time.RFC3339, candidates[j].ModifiedAt)
		return left.After(right)
	})
	return dedupeAccountCandidates(candidates), nil
}

func NormalizeInstallRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(os.ExpandEnv(path))
	if filepath.Base(path) == anniversaryFolder {
		return filepath.Dir(path)
	}
	return path
}

func candidateRoots() []string {
	roots := []string{}
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		roots = append(roots, filepath.Clean(os.ExpandEnv(path)))
	}

	switch runtime.GOOS {
	case "darwin":
		add("/Applications/World of Warcraft")
		if home, err := os.UserHomeDir(); err == nil {
			add(filepath.Join(home, "Applications", "World of Warcraft"))
		}
	case "windows":
		add(filepath.Join(os.Getenv("ProgramFiles(x86)"), "World of Warcraft"))
		add(filepath.Join(os.Getenv("ProgramFiles"), "World of Warcraft"))
	}
	return dedupeStrings(roots)
}

func environmentScanFiles() []string {
	return dedupeStrings([]string{
		os.Getenv(primaryScanFileEnv),
		os.Getenv(legacyScanFileEnv),
	})
}

func validCandidate(path string, installRoot string) (Candidate, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Candidate{}, false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return Candidate{}, false
	}
	if _, err := scanfile.Load(path, ""); err != nil {
		return Candidate{}, false
	}
	installRoot = NormalizeInstallRoot(installRoot)
	if installRoot == "" {
		installRoot = inferInstallRoot(path)
	}
	return Candidate{
		Account:     inferAccount(path),
		InstallPath: installRoot,
		Path:        path,
		ModifiedAt:  info.ModTime().UTC().Format(time.RFC3339),
		Size:        info.Size(),
	}, true
}

func inferAccount(path string) string {
	clean := filepath.Clean(path)
	savedVariables := filepath.Dir(clean)
	account := filepath.Dir(savedVariables)
	if filepath.Base(savedVariables) != "SavedVariables" {
		return ""
	}
	return filepath.Base(account)
}

func inferInstallRoot(path string) string {
	clean := filepath.Clean(path)
	separator := string(filepath.Separator)
	marker := separator + anniversaryFolder + separator
	if index := strings.Index(clean, marker); index >= 0 {
		return filepath.Clean(clean[:index])
	}
	if strings.HasSuffix(clean, separator+anniversaryFolder) {
		return filepath.Dir(clean)
	}
	return ""
}

func dedupeCandidates(candidates []Candidate) []Candidate {
	seen := map[string]bool{}
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := filepath.Clean(candidate.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}

func dedupeAccountCandidates(candidates []Candidate) []Candidate {
	seen := map[string]bool{}
	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(candidate.Account)
		if key == "" {
			key = strings.ToLower(filepath.Clean(candidate.Path))
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(filepath.Clean(value))
		if key == "." || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func BestAnniversaryScanFile() (Candidate, error) {
	candidates, err := DiscoverAnniversaryScanFiles()
	if err != nil {
		return Candidate{}, err
	}
	if len(candidates) == 0 {
		return Candidate{}, errors.New("no WoW Markets SavedVariables file found")
	}
	return candidates[0], nil
}
