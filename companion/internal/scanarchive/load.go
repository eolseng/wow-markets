package scanarchive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
)

func LoadArchive(path string) (exportfmt.Scan, string, error) {
	payload, err := ReadArchive(path)
	if err != nil {
		return exportfmt.Scan{}, "", fmt.Errorf("read archive: %w", err)
	}

	scan, checksum, err := DecodePayload(payload)
	if err != nil {
		return exportfmt.Scan{}, "", err
	}
	fileChecksum := strings.TrimSuffix(filepath.Base(path), ".json.gz")
	if len(fileChecksum) == 64 && fileChecksum != checksum {
		return exportfmt.Scan{}, "", fmt.Errorf(
			"archive checksum %s does not match filename %s",
			checksum,
			fileChecksum,
		)
	}
	return scan, checksum, nil
}

func DecodePayload(payload []byte) (exportfmt.Scan, string, error) {
	var scan exportfmt.Scan
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&scan); err != nil {
		return exportfmt.Scan{}, "", fmt.Errorf("decode archive: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return exportfmt.Scan{}, "", fmt.Errorf("decode archive: trailing JSON value")
		}
		return exportfmt.Scan{}, "", fmt.Errorf("decode archive: %w", err)
	}
	if err := scan.Validate(); err != nil {
		return exportfmt.Scan{}, "", fmt.Errorf("validate archive: %w", err)
	}
	canonical, err := scan.CanonicalJSON()
	if err != nil {
		return exportfmt.Scan{}, "", err
	}
	if !bytes.Equal(payload, canonical) {
		return exportfmt.Scan{}, "", fmt.Errorf(
			"archive payload is not canonical JSON",
		)
	}
	return scan, exportfmt.ChecksumBytes(canonical), nil
}
