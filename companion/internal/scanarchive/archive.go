package scanarchive

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/exportfmt"
	"github.com/eolseng/wow-markets/companion/internal/scanfile"
)

const stateVersion = 1

type Record struct {
	Checksum       string `json:"checksum"`
	ArchivePath    string `json:"archive_path"`
	SourcePath     string `json:"source_path"`
	ImportedAt     string `json:"imported_at"`
	CapturedAt     int64  `json:"captured_at"`
	Region         string `json:"region"`
	Realm          string `json:"realm"`
	Market         string `json:"market"`
	Faction        string `json:"faction"`
	AuctionHouse   string `json:"auction_house"`
	CaptureZone    string `json:"capture_zone,omitempty"`
	CaptureSubzone string `json:"capture_subzone,omitempty"`
	CaptureUIMapID int    `json:"capture_ui_map_id,omitempty"`
	ScannerName    string `json:"scanner_character_name,omitempty"`
	ScannerRealm   string `json:"scanner_character_realm,omitempty"`
	ScannerGUID    string `json:"scanner_character_guid,omitempty"`
	ScannerRegion  string `json:"scanner_region,omitempty"`
	RowCount       int    `json:"row_count"`
	ItemCount      int    `json:"item_count,omitempty"`
	ExportMS       int    `json:"export_duration_ms,omitempty"`
	Truncated      bool   `json:"truncated"`
}

type Result struct {
	Record Record
	IsNew  bool
}

type state struct {
	Version int               `json:"version"`
	Scans   map[string]Record `json:"scans"`
}

type Processor struct {
	dataDir string
	now     func() time.Time
}

func New(dataDir string) (*Processor, error) {
	if dataDir == "" {
		return nil, errors.New("data directory is required")
	}
	absoluteDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	return &Processor{
		dataDir: absoluteDataDir,
		now:     time.Now,
	}, nil
}

func (processor *Processor) ProcessFile(sourcePath, variableName string) ([]Result, error) {
	absoluteSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve source path: %w", err)
	}
	database, err := scanfile.Load(absoluteSourcePath, variableName)
	if err != nil {
		return nil, err
	}

	currentState, err := processor.loadState()
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(database.Scans))
	stateChanged := false
	for _, scan := range database.Scans {
		payload, err := scan.CanonicalJSON()
		if err != nil {
			return nil, err
		}
		checksum := exportfmt.ChecksumBytes(payload)
		if record, exists := currentState.Scans[checksum]; exists {
			results = append(results, Result{Record: record})
			continue
		}

		archivePath, err := processor.archiveScan(checksum, payload)
		if err != nil {
			return nil, err
		}
		record := Record{
			Checksum:       checksum,
			ArchivePath:    archivePath,
			SourcePath:     absoluteSourcePath,
			ImportedAt:     processor.now().UTC().Format(time.RFC3339),
			CapturedAt:     scan.CapturedAt,
			Region:         scan.Region,
			Realm:          scan.Realm,
			Market:         scan.Market(),
			Faction:        scan.Faction,
			AuctionHouse:   scan.AuctionHouse,
			CaptureZone:    scan.CaptureZone,
			CaptureSubzone: scan.CaptureSubzone,
			CaptureUIMapID: scan.CaptureUIMapID,
			ScannerName:    scan.ScannerName,
			ScannerRealm:   scan.ScannerRealm,
			ScannerGUID:    scan.ScannerGUID,
			ScannerRegion:  scan.ScannerRegion,
			RowCount:       scan.ExportedRowCount,
			ItemCount:      scan.ItemCount,
			ExportMS:       scan.ExportDurationMS,
			Truncated:      scan.Truncated,
		}
		currentState.Scans[checksum] = record
		results = append(results, Result{Record: record, IsNew: true})
		stateChanged = true
	}

	if stateChanged {
		if err := processor.writeState(currentState); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (processor *Processor) Records() ([]Record, error) {
	currentState, err := processor.loadState()
	if err != nil {
		return nil, err
	}
	checksums := make([]string, 0, len(currentState.Scans))
	for checksum := range currentState.Scans {
		checksums = append(checksums, checksum)
	}
	sort.Strings(checksums)

	records := make([]Record, 0, len(checksums))
	for _, checksum := range checksums {
		records = append(records, currentState.Scans[checksum])
	}
	return records, nil
}

func (processor *Processor) archiveScan(checksum string, payload []byte) (string, error) {
	scansDir := filepath.Join(processor.dataDir, "scans")
	if err := os.MkdirAll(scansDir, 0o700); err != nil {
		return "", fmt.Errorf("create scan archive directory: %w", err)
	}

	archivePath := filepath.Join(scansDir, checksum+".json.gz")
	if _, err := os.Stat(archivePath); err == nil {
		return archivePath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect archive %s: %w", archivePath, err)
	}

	if err := writeGzipAtomic(archivePath, payload); err != nil {
		return "", err
	}
	return archivePath, nil
}

func (processor *Processor) loadState() (state, error) {
	path := filepath.Join(processor.dataDir, "state.json")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return state{Version: stateVersion, Scans: map[string]Record{}}, nil
	}
	if err != nil {
		return state{}, fmt.Errorf("open importer state: %w", err)
	}
	defer file.Close()

	var result state
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return state{}, fmt.Errorf("decode importer state: %w", err)
	}
	if result.Version != stateVersion {
		return state{}, fmt.Errorf(
			"unsupported importer state version %d; expected %d",
			result.Version,
			stateVersion,
		)
	}
	if result.Scans == nil {
		result.Scans = map[string]Record{}
	}
	return result, nil
}

func (processor *Processor) writeState(value state) error {
	if err := os.MkdirAll(processor.dataDir, 0o700); err != nil {
		return fmt.Errorf("create importer data directory: %w", err)
	}

	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode importer state: %w", err)
	}
	payload = append(payload, '\n')
	if err := writeAtomic(filepath.Join(processor.dataDir, "state.json"), payload); err != nil {
		return fmt.Errorf("write importer state: %w", err)
	}
	return nil
}

func writeGzipAtomic(path string, payload []byte) error {
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".scan-*.tmp")
	if err != nil {
		return fmt.Errorf("create archive temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("set archive permissions: %w", err)
	}

	writer := gzip.NewWriter(temp)
	writer.Header.ModTime = time.Unix(0, 0)
	if _, err := writer.Write(payload); err != nil {
		writer.Close()
		temp.Close()
		return fmt.Errorf("compress scan archive: %w", err)
	}
	if err := writer.Close(); err != nil {
		temp.Close()
		return fmt.Errorf("finish scan archive: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync scan archive: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close scan archive: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish scan archive: %w", err)
	}
	return nil
}

func writeAtomic(path string, payload []byte) error {
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func ReadArchive(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
