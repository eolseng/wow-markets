package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanarchive"
	"github.com/eolseng/wow-markets/companion/internal/scanupload"
)

func TestLoadActivityOverviewTracksPersistedUploadLifecycle(t *testing.T) {
	dataDir := t.TempDir()
	processor, err := scanarchive.New(dataDir)
	if err != nil {
		t.Fatalf("scanarchive.New() error = %v", err)
	}
	results, err := processor.ProcessFile(filepath.Join("testdata", "WoWMarkets.lua"), "")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	record := results[0].Record

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(writer, `{
			"checksum":%q,
			"scan_id":7,
			"submission_id":"submission-7",
			"status":"accepted",
			"rows":2,
			"items":2,
			"price_levels":2,
			"price_snapshots":2
		}`, record.Checksum)
	}))
	defer server.Close()

	uploader, err := scanupload.New(dataDir, server.URL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("scanupload.New() error = %v", err)
	}
	if _, err := uploader.Queue([]scanarchive.Record{record}); err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	queued, err := loadActivityOverview(dataDir)
	if err != nil {
		t.Fatalf("loadActivityOverview() queued error = %v", err)
	}
	if queued.archivedCount != 1 || queued.queuedCount != 1 || queued.uploadedCount != 0 {
		t.Fatalf("queued overview = %+v", queued)
	}
	if len(queued.recentUploads) != 1 || queued.recentUploads[0].RowCount != record.RowCount || queued.recentUploads[0].ItemCount != record.ItemCount {
		t.Fatalf("queued recent uploads = %+v", queued.recentUploads)
	}

	var during activityOverview
	var progressErr error
	if _, err := uploader.ProcessDueWithProgress(context.Background(), func(scanupload.Record) {
		during, progressErr = loadActivityOverview(dataDir)
	}); err != nil {
		t.Fatalf("ProcessDueWithProgress() error = %v", err)
	}
	if progressErr != nil {
		t.Fatalf("loadActivityOverview() progress error = %v", progressErr)
	}
	if during.uploadingCount != 1 || during.currentUpload == nil || during.currentUpload.Status != "uploading" {
		t.Fatalf("uploading overview = %+v", during)
	}

	uploaded, err := loadActivityOverview(dataDir)
	if err != nil {
		t.Fatalf("loadActivityOverview() uploaded error = %v", err)
	}
	if uploaded.queuedCount != 0 || uploaded.uploadingCount != 0 || uploaded.uploadedCount != 1 || uploaded.failedCount != 0 {
		t.Fatalf("uploaded overview = %+v", uploaded)
	}
	if uploaded.lastUpload == nil || uploaded.lastUpload.ScanID != 7 || uploaded.lastUpload.RowCount != 2 || uploaded.lastUpload.ItemCount != 2 {
		t.Fatalf("last upload = %+v", uploaded.lastUpload)
	}
	if uploaded.uploadFailure != nil {
		t.Fatalf("upload failure = %+v, want nil", uploaded.uploadFailure)
	}
}

func TestLoadActivityOverviewKeepsAuthorizationFailureActionable(t *testing.T) {
	dataDir := t.TempDir()
	processor, err := scanarchive.New(dataDir)
	if err != nil {
		t.Fatalf("scanarchive.New() error = %v", err)
	}
	results, err := processor.ProcessFile(filepath.Join("testdata", "WoWMarkets.lua"), "")
	if err != nil {
		t.Fatalf("ProcessFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":"invalid_installation_token"}`))
	}))
	defer server.Close()
	uploader, err := scanupload.New(dataDir, server.URL, "revoked-token", time.Second)
	if err != nil {
		t.Fatalf("scanupload.New() error = %v", err)
	}
	if _, err := uploader.Queue([]scanarchive.Record{results[0].Record}); err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if _, err := uploader.ProcessDue(context.Background()); err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}

	overview, err := loadActivityOverview(dataDir)
	if err != nil {
		t.Fatalf("loadActivityOverview() error = %v", err)
	}
	if overview.uploadFailure == nil || overview.uploadFailure.HTTPStatus != http.StatusUnauthorized || overview.uploadFailure.Retryable {
		t.Fatalf("upload failure = %+v", overview.uploadFailure)
	}
}
