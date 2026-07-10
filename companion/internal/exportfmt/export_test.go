package exportfmt

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/eolseng/wow-markets/companion/internal/luasv"
)

func TestDecodeCurrentFixture(t *testing.T) {
	database := loadFixture(t, "../../../contracts/saved-variables/v5/fixtures/valid.lua")
	if len(database.Scans) != 1 {
		t.Fatalf("len(Scans) = %d, want 1", len(database.Scans))
	}

	scan := database.Scans[0]
	if scan.FormatVersion != FormatVersion {
		t.Fatalf("FormatVersion = %d, want %d", scan.FormatVersion, FormatVersion)
	}
	if scan.Region != "eu" {
		t.Fatalf("Region = %q, want eu", scan.Region)
	}
	if scan.ExportDurationMS != 3750 || scan.ExportBatchSize != 250 {
		t.Fatalf("export diagnostics = %#v", scan)
	}
	if scan.ScannerName != "Examplechar" ||
		scan.ScannerRealm != "Spineshatter" ||
		scan.ScannerGUID != "Player-0000-00000001" ||
		scan.ScannerRegion != "eu" {
		t.Fatalf("scanner identity = %#v", scan)
	}
	if scan.AuctionHouse != "faction" ||
		scan.CaptureZone != "Stormwind City" ||
		scan.CaptureSubzone != "Trade District" ||
		scan.CaptureUIMapID != 1453 {
		t.Fatalf("auction house context = %#v", scan)
	}
	if len(scan.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(scan.Rows))
	}
	if scan.Rows[0].ItemID != 6149 ||
		scan.Rows[0].Name != "Greater Mana Potion" ||
		scan.Rows[1].ItemID != 24779 {
		t.Fatalf("decoded rows = %#v", scan.Rows)
	}
	if got := scan.Rows[0].UnitBuyout(); got != 8520 {
		t.Fatalf("UnitBuyout() = %d, want 8520", got)
	}

	firstChecksum, err := scan.Checksum()
	if err != nil {
		t.Fatalf("Checksum() error = %v", err)
	}
	secondChecksum, err := scan.Checksum()
	if err != nil {
		t.Fatalf("Checksum() second error = %v", err)
	}
	if firstChecksum == "" || firstChecksum != secondChecksum {
		t.Fatalf("checksums are not stable: %q, %q", firstChecksum, secondChecksum)
	}
}

func TestCanonicalJSONMatchesPublicFixture(t *testing.T) {
	scan := loadFixture(t, "../../../contracts/saved-variables/v5/fixtures/valid.lua").Scans[0]
	actual, err := scan.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	expected, err := os.ReadFile("../../../contracts/scan-archive/v1/fixtures/valid.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("canonical JSON differs from public fixture")
	}
}

func TestDecodeRejectsPreviousFormat(t *testing.T) {
	payload, err := os.ReadFile("../../../contracts/saved-variables/v5/fixtures/valid.lua")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	legacyPayload := strings.Replace(
		string(payload),
		`["formatVersion"] = 5`,
		`["formatVersion"] = 4`,
		1,
	)
	root, err := luasv.ParseVariable(
		strings.NewReader(legacyPayload),
		"WOW_MARKETS_DB",
	)
	if err != nil {
		t.Fatalf("ParseVariable() error = %v", err)
	}
	if _, err := Decode(root); err == nil {
		t.Fatal("Decode() accepted format version 4")
	}
}

func TestValidateRejectsAuctionHouseLocationMismatch(t *testing.T) {
	scan := loadFixture(t, "../../../contracts/saved-variables/v5/fixtures/valid.lua").Scans[0]
	scan.AuctionHouse = "neutral"
	if err := scan.Validate(); err == nil {
		t.Fatal("Validate() accepted a neutral AH outside a neutral zone")
	}
}

func TestValidateAcceptsNeutralAuctionHouseLocation(t *testing.T) {
	scan := loadFixture(t, "../../../contracts/saved-variables/v5/fixtures/valid.lua").Scans[0]
	scan.AuctionHouse = "neutral"
	scan.CaptureZone = "Tanaris"
	scan.CaptureSubzone = "Gadgetzan"
	scan.CaptureUIMapID = 1446
	if err := scan.Validate(); err != nil {
		t.Fatalf("Validate() rejected neutral AH location: %v", err)
	}
}

func TestValidateRejectsDuplicateSourceRows(t *testing.T) {
	scan := loadFixture(t, "../../../contracts/saved-variables/v5/fixtures/valid.lua").Scans[0]
	scan.Rows[1].SourceRow = scan.Rows[0].SourceRow
	if err := scan.Validate(); err == nil {
		t.Fatal("Validate() accepted duplicate source rows")
	}
}

func loadFixture(t *testing.T, path string) Database {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	root, err := luasv.ParseVariable(file, "WOW_MARKETS_DB")
	if err != nil {
		t.Fatalf("ParseVariable() error = %v", err)
	}
	database, err := Decode(root)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return database
}
