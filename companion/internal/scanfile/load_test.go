package scanfile

import "testing"

func TestLoad(t *testing.T) {
	database, err := Load("../../testdata/WowMarketScan.lua", "")
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
