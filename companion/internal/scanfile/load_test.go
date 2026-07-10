package scanfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	database, err := Load("../../testdata/WoWMarkets.lua", "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(database.Scans) != 1 {
		t.Fatalf("len(Scans) = %d, want 1", len(database.Scans))
	}
	if database.Scans[0].ScannerName != "Examplechar" {
		t.Fatalf("ScannerName = %q", database.Scans[0].ScannerName)
	}
}

func TestLoadFallsBackToLegacyVariable(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/WoWMarkets.lua")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	legacyPayload := strings.Replace(
		string(payload),
		DefaultVariableName,
		LegacyVariableName,
		1,
	)
	path := filepath.Join(t.TempDir(), "WowMarketScan.lua")
	if err := os.WriteFile(path, []byte(legacyPayload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	database, err := Load(path, "")
	if err != nil {
		t.Fatalf("Load() legacy error = %v", err)
	}
	if len(database.Scans) != 1 {
		t.Fatalf("len(Scans) = %d, want 1", len(database.Scans))
	}
}
