package watchagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanarchive"
	"github.com/eolseng/wow-markets/companion/internal/scanfile"
	"github.com/eolseng/wow-markets/companion/internal/scanupload"
)

type Config struct {
	FilePath      string
	DataDir       string
	VariableName  string
	Interval      time.Duration
	APIURL        string
	APIToken      string
	UploadTimeout time.Duration
}

type Event struct {
	Kind            string
	Message         string
	Checksum        string
	Error           string
	Time            time.Time
	Status          scanupload.Status
	CapturedAt      int64
	ImportedAt      string
	Region          string
	Realm           string
	Market          string
	Faction         string
	AuctionHouse    string
	ScannerName     string
	ScannerRealm    string
	RowCount        int
	ItemCount       int
	Attempts        int
	Retryable       bool
	HTTPStatus      int
	QueuedAt        string
	LastAttemptAt   string
	UploadedAt      string
	ServerStatus    string
	ScanID          int64
	SubmissionID    string
	ObservationRows int64
	PriceLevels     int64
	PriceSnapshots  int64
}

func Run(ctx context.Context, config Config, emit func(Event)) error {
	if config.VariableName == "" {
		config.VariableName = scanfile.DefaultVariableName
	}
	if config.Interval <= 0 {
		config.Interval = 5 * time.Second
	}
	if config.UploadTimeout <= 0 {
		config.UploadTimeout = 15 * time.Minute
	}
	processor, err := scanarchive.New(config.DataDir)
	if err != nil {
		return err
	}
	uploader, err := scanupload.New(
		config.DataDir,
		config.APIURL,
		config.APIToken,
		config.UploadTimeout,
	)
	if err != nil {
		return err
	}

	var lastSignature fileSignature
	processArchive := func(force bool) error {
		signature, err := statSignature(config.FilePath)
		if err != nil {
			return err
		}
		if !force && signature == lastSignature {
			return nil
		}
		results, err := processor.ProcessFile(config.FilePath, config.VariableName)
		if err != nil {
			return err
		}
		lastSignature = signature

		records := make([]scanarchive.Record, 0, len(results))
		recordsByChecksum := make(map[string]scanarchive.Record, len(results))
		for _, result := range results {
			records = append(records, result.Record)
			recordsByChecksum[result.Record.Checksum] = result.Record
			if result.IsNew {
				event := archiveEvent("archive", result.Record)
				event.Message = fmt.Sprintf("Archived %d rows from %s-%s", result.Record.RowCount, valueOrUnknown(result.Record.ScannerName), valueOrUnknown(result.Record.ScannerRealm))
				emit(event)
			}
		}
		if force {
			records, err = processor.Records()
			if err != nil {
				return err
			}
			for _, record := range records {
				recordsByChecksum[record.Checksum] = record
			}
		}
		queued, err := uploader.Queue(records)
		if err != nil {
			return err
		}
		for _, result := range queued {
			if result.Queued {
				event := archiveEvent("queue", recordsByChecksum[result.Checksum])
				event.Message = "Queued scan for upload"
				event.Status = scanupload.StatusPending
				emit(event)
			}
		}
		return nil
	}

	processUploads := func(ctx context.Context) error {
		results, err := uploader.ProcessDueWithProgress(ctx, func(record scanupload.Record) {
			event := uploadEvent("uploading", record)
			event.Message = "Uploading scan"
			emit(event)
		})
		if err != nil {
			return err
		}
		for _, result := range results {
			record := result.Record
			if record.Status == scanupload.StatusUploaded {
				event := uploadEvent("upload", record)
				event.Message = fmt.Sprintf("Uploaded scan %d (%s)", record.ScanID, record.ServerStatus)
				emit(event)
				continue
			}
			event := uploadEvent("upload_error", record)
			event.Message = "Upload failed"
			event.Error = record.LastError
			emit(event)
		}
		return nil
	}

	unhealthy := false
	runCycle := func(force bool) {
		cycleFailed := false
		if err := processArchive(force); err != nil {
			if !errors.Is(err, context.Canceled) {
				emitError(emit, "archive_error", err)
				cycleFailed = true
			}
		}
		if err := processUploads(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				emitError(emit, "upload_error", err)
				cycleFailed = true
			}
		}
		if unhealthy && !cycleFailed {
			emit(Event{
				Kind:    "recovered",
				Message: "Watcher recovered",
				Time:    time.Now().UTC(),
			})
		}
		unhealthy = cycleFailed
	}

	emit(Event{Kind: "status", Message: "Watcher started", Time: time.Now().UTC()})
	runCycle(true)

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			emit(Event{Kind: "status", Message: "Watcher stopped", Time: time.Now().UTC()})
			return nil
		case <-ticker.C:
			runCycle(false)
		}
	}
}

type fileSignature struct {
	Size       int64
	ModifiedNS int64
}

func statSignature(path string) (fileSignature, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileSignature{}, fmt.Errorf("inspect %s: %w", path, err)
	}
	return fileSignature{Size: info.Size(), ModifiedNS: info.ModTime().UnixNano()}, nil
}

func emitError(emit func(Event), kind string, err error) {
	emit(Event{
		Kind:    kind,
		Message: err.Error(),
		Error:   err.Error(),
		Time:    time.Now().UTC(),
	})
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func archiveEvent(kind string, record scanarchive.Record) Event {
	return Event{
		Kind:         kind,
		Checksum:     record.Checksum,
		Time:         time.Now().UTC(),
		CapturedAt:   record.CapturedAt,
		ImportedAt:   record.ImportedAt,
		Region:       record.Region,
		Realm:        record.Realm,
		Market:       record.Market,
		Faction:      record.Faction,
		AuctionHouse: record.AuctionHouse,
		ScannerName:  record.ScannerName,
		ScannerRealm: record.ScannerRealm,
		RowCount:     record.RowCount,
		ItemCount:    record.ItemCount,
	}
}

func uploadEvent(kind string, record scanupload.Record) Event {
	itemCount := record.ArchiveItems
	if itemCount == 0 {
		itemCount = record.ItemCount
	}
	return Event{
		Kind:            kind,
		Checksum:        record.Checksum,
		Time:            time.Now().UTC(),
		Status:          record.Status,
		CapturedAt:      record.CapturedAt,
		ImportedAt:      record.ImportedAt,
		Region:          record.Region,
		Realm:           record.Realm,
		Market:          record.Market,
		Faction:         record.Faction,
		AuctionHouse:    record.AuctionHouse,
		ScannerName:     record.ScannerName,
		ScannerRealm:    record.ScannerRealm,
		RowCount:        record.ArchiveRows,
		ItemCount:       itemCount,
		Attempts:        record.Attempts,
		Retryable:       record.Retryable,
		HTTPStatus:      record.HTTPStatus,
		QueuedAt:        record.QueuedAt,
		LastAttemptAt:   record.LastAttemptAt,
		UploadedAt:      record.UploadedAt,
		ServerStatus:    record.ServerStatus,
		ScanID:          record.ScanID,
		SubmissionID:    record.SubmissionID,
		ObservationRows: record.ObservationRows,
		PriceLevels:     record.PriceLevels,
		PriceSnapshots:  record.PriceSnapshots,
	}
}
