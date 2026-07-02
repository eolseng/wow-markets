package scanarchive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
)

func TestProcessFileArchivesOnce(t *testing.T) {
	dataDir := t.TempDir()
	processor, err := New(dataDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	processor.now = func() time.Time {
		return time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	}

	first, err := processor.ProcessFile("../../testdata/WowMarketScan.lua", "")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}
	if len(first) != 1 || !first[0].IsNew {
		t.Fatalf("first results = %#v", first)
	}

	second, err := processor.ProcessFile("../../testdata/WowMarketScan.lua", "")
	if err != nil {
		t.Fatalf("ProcessFile() second error = %v", err)
	}
	if len(second) != 1 || second[0].IsNew {
		t.Fatalf("second results = %#v", second)
	}

	payload, err := ReadArchive(first[0].Record.ArchivePath)
	if err != nil {
		t.Fatalf("ReadArchive() error = %v", err)
	}
	var scan exportfmt.Scan
	if err := json.Unmarshal(payload, &scan); err != nil {
		t.Fatalf("decode archived scan: %v", err)
	}
	if scan.ScannerName != "Examplechar" ||
		scan.Region != "eu" ||
		scan.CaptureZone != "Stormwind City" ||
		len(scan.Rows) != 2 ||
		scan.ItemCount != 2 ||
		scan.ExportDurationMS != 3750 {
		t.Fatalf("archived scan = %#v", scan)
	}

	statePayload, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var savedState state
	if err := json.Unmarshal(statePayload, &savedState); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(savedState.Scans) != 1 {
		t.Fatalf("len(state.Scans) = %d, want 1", len(savedState.Scans))
	}

	records, err := processor.Records()
	if err != nil {
		t.Fatalf("Records() error = %v", err)
	}
	if len(records) != 1 || records[0].Checksum != first[0].Record.Checksum {
		t.Fatalf("Records() = %#v", records)
	}
	if records[0].CaptureZone != "Stormwind City" ||
		records[0].CaptureUIMapID != 1453 ||
		records[0].Region != "eu" ||
		records[0].Market != "Alliance" {
		t.Fatalf("record capture location = %#v", records[0])
	}
}
