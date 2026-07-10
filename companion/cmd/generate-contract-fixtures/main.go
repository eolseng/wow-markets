package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eolseng/wow-markets/companion/internal/scanfile"
)

const (
	savedVariablesFixture = "../contracts/saved-variables/v5/fixtures/valid.lua"
	archiveFixturesDir    = "../contracts/scan-archive/v1/fixtures"
	checksumsPath         = "../contracts/scan-archive/v1/checksums.txt"
)

func main() {
	database, err := scanfile.Load(savedVariablesFixture, "")
	must(err)
	if len(database.Scans) != 1 {
		panic(fmt.Sprintf("fixture contains %d scans; expected 1", len(database.Scans)))
	}
	payload, err := database.Scans[0].CanonicalJSON()
	must(err)
	must(database.Scans[0].Validate())

	must(os.MkdirAll(archiveFixturesDir, 0o755))
	jsonPath := filepath.Join(archiveFixturesDir, "valid.json")
	gzipPath := filepath.Join(archiveFixturesDir, "valid.json.gz")
	must(os.WriteFile(jsonPath, payload, 0o644))

	file, err := os.Create(gzipPath)
	must(err)
	writer, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	must(err)
	_, err = writer.Write(payload)
	must(err)
	must(writer.Close())
	must(file.Close())

	gzipPayload, err := os.ReadFile(gzipPath)
	must(err)
	checksums := fmt.Sprintf(
		"%s  fixtures/valid.json\n%s  fixtures/valid.json.gz\n",
		checksum(payload),
		checksum(gzipPayload),
	)
	must(os.WriteFile(checksumsPath, []byte(checksums), 0o644))
}

func checksum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
