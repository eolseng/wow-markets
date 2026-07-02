package watchagent

import (
	"context"
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
	Kind     string
	Message  string
	Checksum string
	Error    string
	Time     time.Time
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
		for _, result := range results {
			records = append(records, result.Record)
			if result.IsNew {
				emit(Event{
					Kind:     "archive",
					Message:  fmt.Sprintf("Archived %d rows from %s-%s", result.Record.RowCount, valueOrUnknown(result.Record.ScannerName), valueOrUnknown(result.Record.ScannerRealm)),
					Checksum: result.Record.Checksum,
					Time:     time.Now().UTC(),
				})
			}
		}
		if force {
			records, err = processor.Records()
			if err != nil {
				return err
			}
		}
		queued, err := uploader.Queue(records)
		if err != nil {
			return err
		}
		for _, result := range queued {
			if result.Queued {
				emit(Event{
					Kind:     "queue",
					Message:  "Queued scan for upload",
					Checksum: result.Checksum,
					Time:     time.Now().UTC(),
				})
			}
		}
		return nil
	}

	processUploads := func(ctx context.Context) error {
		results, err := uploader.ProcessDue(ctx)
		if err != nil {
			return err
		}
		for _, result := range results {
			record := result.Record
			if record.Status == scanupload.StatusUploaded {
				emit(Event{
					Kind:     "upload",
					Message:  fmt.Sprintf("Uploaded scan %d (%s)", record.ScanID, record.ServerStatus),
					Checksum: record.Checksum,
					Time:     time.Now().UTC(),
				})
				continue
			}
			emit(Event{
				Kind:     "upload_error",
				Message:  "Upload failed",
				Checksum: record.Checksum,
				Error:    record.LastError,
				Time:     time.Now().UTC(),
			})
		}
		return nil
	}

	emit(Event{Kind: "status", Message: "Watcher started", Time: time.Now().UTC()})
	if err := processArchive(true); err != nil {
		emitError(emit, "archive_error", err)
	}
	if err := processUploads(ctx); err != nil {
		emitError(emit, "upload_error", err)
	}

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			emit(Event{Kind: "status", Message: "Watcher stopped", Time: time.Now().UTC()})
			return nil
		case <-ticker.C:
			if err := processArchive(false); err != nil {
				emitError(emit, "archive_error", err)
			}
			if err := processUploads(ctx); err != nil {
				emitError(emit, "upload_error", err)
			}
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
