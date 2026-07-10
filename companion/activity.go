package main

import (
	"sort"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanarchive"
	"github.com/eolseng/wow-markets/companion/internal/scanupload"
)

const recentUploadLimit = 6

type ScanSummary struct {
	Attempts        int    `json:"attempts"`
	AuctionHouse    string `json:"auction_house"`
	CapturedAt      string `json:"captured_at"`
	Checksum        string `json:"checksum"`
	Error           string `json:"error"`
	HTTPStatus      int    `json:"http_status"`
	ImportedAt      string `json:"imported_at"`
	ItemCount       int    `json:"item_count"`
	LastAttemptAt   string `json:"last_attempt_at"`
	Market          string `json:"market"`
	NextAttemptAt   string `json:"next_attempt_at"`
	ObservationRows int64  `json:"observation_rows"`
	PriceLevels     int64  `json:"price_levels"`
	PriceSnapshots  int64  `json:"price_snapshots"`
	QueuedAt        string `json:"queued_at"`
	Realm           string `json:"realm"`
	Region          string `json:"region"`
	Retryable       bool   `json:"retryable"`
	RowCount        int    `json:"row_count"`
	ScanID          int64  `json:"scan_id"`
	ScannerName     string `json:"scanner_name"`
	ScannerRealm    string `json:"scanner_realm"`
	ServerStatus    string `json:"server_status"`
	Status          string `json:"status"`
	SubmissionID    string `json:"submission_id"`
	UploadedAt      string `json:"uploaded_at"`
}

type activityOverview struct {
	archivedCount  int
	queuedCount    int
	uploadingCount int
	uploadedCount  int
	failedCount    int
	lastArchiveAt  string
	lastUploadAt   string
	currentUpload  *ScanSummary
	lastDetected   *ScanSummary
	lastUpload     *ScanSummary
	uploadFailure  *ScanSummary
	recentUploads  []ScanSummary
}

func (app *App) reloadActivity() error {
	app.mu.Lock()
	dataDir := app.dataDir
	app.mu.Unlock()
	if dataDir == "" {
		return nil
	}

	overview, err := loadActivityOverview(dataDir)
	if err != nil {
		return err
	}
	app.mu.Lock()
	app.archivedCount = overview.archivedCount
	app.queuedCount = overview.queuedCount
	app.uploadingCount = overview.uploadingCount
	app.uploadedCount = overview.uploadedCount
	app.failedCount = overview.failedCount
	app.lastArchiveAt = overview.lastArchiveAt
	app.lastUploadAt = overview.lastUploadAt
	app.currentUpload = overview.currentUpload
	app.lastDetected = overview.lastDetected
	app.lastUpload = overview.lastUpload
	app.uploadFailure = overview.uploadFailure
	app.recentUploads = overview.recentUploads
	app.mu.Unlock()
	return nil
}

func loadActivityOverview(dataDir string) (activityOverview, error) {
	processor, err := scanarchive.New(dataDir)
	if err != nil {
		return activityOverview{}, err
	}
	archives, err := processor.Records()
	if err != nil {
		return activityOverview{}, err
	}
	uploads, err := scanupload.ReadRecords(dataDir)
	if err != nil {
		return activityOverview{}, err
	}

	archiveByChecksum := make(map[string]scanarchive.Record, len(archives))
	for _, archive := range archives {
		archiveByChecksum[archive.Checksum] = archive
	}
	sort.SliceStable(archives, func(i, j int) bool {
		return parseTime(archives[i].ImportedAt).After(parseTime(archives[j].ImportedAt))
	})

	overview := activityOverview{archivedCount: len(archives)}
	if len(archives) > 0 {
		summary := summaryFromArchive(archives[0])
		overview.lastDetected = &summary
		overview.lastArchiveAt = archives[0].ImportedAt
	}

	sort.SliceStable(uploads, func(i, j int) bool {
		return uploadActivityTime(uploads[i]).After(uploadActivityTime(uploads[j]))
	})
	for _, upload := range uploads {
		switch upload.Status {
		case scanupload.StatusPending:
			overview.queuedCount++
		case scanupload.StatusUploading:
			overview.uploadingCount++
		case scanupload.StatusUploaded:
			overview.uploadedCount++
		case scanupload.StatusFailed:
			overview.failedCount++
		}

		archive, found := archiveByChecksum[upload.Checksum]
		summary := summaryFromUpload(upload, archive, found)
		if upload.Status == scanupload.StatusUploading && overview.currentUpload == nil {
			current := summary
			overview.currentUpload = &current
		}
		if upload.Status == scanupload.StatusUploaded && overview.lastUpload == nil {
			last := summary
			overview.lastUpload = &last
			overview.lastUploadAt = summary.UploadedAt
		}
		if upload.Status == scanupload.StatusFailed && overview.uploadFailure == nil {
			failure := summary
			overview.uploadFailure = &failure
		}
		if len(overview.recentUploads) < recentUploadLimit {
			overview.recentUploads = append(overview.recentUploads, summary)
		}
	}
	return overview, nil
}

func summaryFromArchive(record scanarchive.Record) ScanSummary {
	return ScanSummary{
		AuctionHouse: record.AuctionHouse,
		CapturedAt:   unixTime(record.CapturedAt),
		Checksum:     record.Checksum,
		ImportedAt:   record.ImportedAt,
		ItemCount:    record.ItemCount,
		Market:       record.Market,
		Realm:        record.Realm,
		Region:       record.Region,
		RowCount:     record.RowCount,
		ScannerName:  record.ScannerName,
		ScannerRealm: record.ScannerRealm,
		Status:       "detected",
	}
}

func summaryFromUpload(upload scanupload.Record, archive scanarchive.Record, archiveFound bool) ScanSummary {
	rowCount := upload.ArchiveRows
	itemCount := upload.ArchiveItems
	capturedAt := upload.CapturedAt
	importedAt := upload.ImportedAt
	region := upload.Region
	realm := upload.Realm
	market := upload.Market
	auctionHouse := upload.AuctionHouse
	scannerName := upload.ScannerName
	scannerRealm := upload.ScannerRealm
	if archiveFound {
		rowCount = archive.RowCount
		itemCount = archive.ItemCount
		capturedAt = archive.CapturedAt
		importedAt = archive.ImportedAt
		region = archive.Region
		realm = archive.Realm
		market = archive.Market
		auctionHouse = archive.AuctionHouse
		scannerName = archive.ScannerName
		scannerRealm = archive.ScannerRealm
	}
	if rowCount == 0 && upload.ObservationRows > 0 {
		rowCount = int(upload.ObservationRows)
	}
	if itemCount == 0 && upload.ItemCount > 0 {
		itemCount = upload.ItemCount
	}
	return ScanSummary{
		Attempts:        upload.Attempts,
		AuctionHouse:    auctionHouse,
		CapturedAt:      unixTime(capturedAt),
		Checksum:        upload.Checksum,
		Error:           upload.LastError,
		HTTPStatus:      upload.HTTPStatus,
		ImportedAt:      importedAt,
		ItemCount:       itemCount,
		LastAttemptAt:   upload.LastAttemptAt,
		Market:          market,
		NextAttemptAt:   upload.NextAttemptAt,
		ObservationRows: upload.ObservationRows,
		PriceLevels:     upload.PriceLevels,
		PriceSnapshots:  upload.PriceSnapshots,
		QueuedAt:        upload.QueuedAt,
		Realm:           realm,
		Region:          region,
		Retryable:       upload.Retryable,
		RowCount:        rowCount,
		ScanID:          upload.ScanID,
		ScannerName:     scannerName,
		ScannerRealm:    scannerRealm,
		ServerStatus:    upload.ServerStatus,
		Status:          string(upload.Status),
		SubmissionID:    upload.SubmissionID,
		UploadedAt:      upload.UploadedAt,
	}
}

func uploadActivityTime(record scanupload.Record) time.Time {
	for _, value := range []string{record.UploadedAt, record.LastAttemptAt, record.QueuedAt} {
		if parsed := parseTime(value); !parsed.IsZero() {
			return parsed
		}
	}
	return time.Time{}
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func unixTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func cloneScanSummary(value *ScanSummary) *ScanSummary {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
