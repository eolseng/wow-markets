package scanarchive

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
)

func TestLoadArchiveValidatesChecksumFilename(t *testing.T) {
	processor, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	results, err := processor.ProcessFile("../../../contracts/saved-variables/v5/fixtures/valid.lua", "")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	scan, checksum, err := LoadArchive(results[0].Record.ArchivePath)
	if err != nil {
		t.Fatalf("LoadArchive() error = %v", err)
	}
	if checksum != results[0].Record.Checksum || scan.ExportedRowCount != 2 {
		t.Fatalf("loaded checksum=%q scan=%#v", checksum, scan)
	}

	mismatchedPath := filepath.Join(
		filepath.Dir(results[0].Record.ArchivePath),
		strings.Repeat("0", 64)+".json.gz",
	)
	payload, err := os.ReadFile(results[0].Record.ArchivePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := os.WriteFile(mismatchedPath, payload, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, _, err := LoadArchive(mismatchedPath); err == nil {
		t.Fatal("LoadArchive() accepted a mismatched checksum filename")
	}
}

func TestPublicGzipFixtureMatchesCanonicalPayload(t *testing.T) {
	payload, err := ReadArchive("../../../contracts/scan-archive/v1/fixtures/valid.json.gz")
	if err != nil {
		t.Fatalf("ReadArchive() error = %v", err)
	}
	expected, err := os.ReadFile("../../../contracts/scan-archive/v1/fixtures/valid.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(payload, expected) {
		t.Fatal("gzip fixture does not contain the canonical JSON fixture")
	}
	if _, _, err := DecodePayload(payload); err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}
}

func TestLoadArchiveRejectsInvalidScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.json.gz")
	scan := exportfmt.Scan{FormatVersion: exportfmt.FormatVersion}
	payload, err := scan.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	if err := writeGzipAtomic(path, payload); err != nil {
		t.Fatalf("writeGzipAtomic() error = %v", err)
	}
	if _, _, err := LoadArchive(path); err == nil {
		t.Fatal("LoadArchive() accepted an invalid scan")
	}
}

func TestDecodePayloadRejectsNonCanonicalJSON(t *testing.T) {
	payload, err := ReadArchive(createFixtureArchive(t))
	if err != nil {
		t.Fatalf("ReadArchive() error = %v", err)
	}
	payload = append(payload, '\n')

	if _, _, err := DecodePayload(payload); err == nil {
		t.Fatal("DecodePayload() accepted non-canonical JSON")
	}
}

func TestDecodePayloadRejectsUnknownFields(t *testing.T) {
	payload, err := ReadArchive(createFixtureArchive(t))
	if err != nil {
		t.Fatalf("ReadArchive() error = %v", err)
	}
	payload = bytes.Replace(
		payload,
		[]byte(`"status":"ready"`),
		[]byte(`"status":"ready","unknown":true`),
		1,
	)

	if _, _, err := DecodePayload(payload); err == nil {
		t.Fatal("DecodePayload() accepted an unknown field")
	}
}

func createFixtureArchive(t *testing.T) string {
	t.Helper()

	processor, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	results, err := processor.ProcessFile("../../../contracts/saved-variables/v5/fixtures/valid.lua", "")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}
	return results[0].Record.ArchivePath
}
