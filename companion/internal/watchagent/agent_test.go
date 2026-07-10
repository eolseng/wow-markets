package watchagent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanfile"
	"github.com/eolseng/wow-markets/companion/internal/scanupload"
)

func TestRunEmitsUploadLifecycleWithSafeScanMetadata(t *testing.T) {
	scanPath := filepath.Join("..", "..", "..", "contracts", "saved-variables", "v5", "fixtures", "valid.lua")
	database, err := scanfile.Load(scanPath, scanfile.DefaultVariableName)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(database.Scans) != 1 {
		t.Fatalf("len(scans) = %d, want 1", len(database.Scans))
	}
	scan := database.Scans[0]
	checksum, err := scan.Checksum()
	if err != nil {
		t.Fatalf("Checksum() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(writer, `{
			"checksum":%q,
			"scan_id":42,
			"submission_id":"submission-42",
			"status":"accepted",
			"rows":2,
			"items":2,
			"price_levels":2,
			"price_snapshots":2
		}`, checksum)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataDir := t.TempDir()
	var events []Event
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{
			FilePath:      scanPath,
			DataDir:       dataDir,
			Interval:      time.Hour,
			APIURL:        server.URL,
			APIToken:      "test-token",
			UploadTimeout: time.Second,
		}, func(event Event) {
			events = append(events, event)
			if event.Kind == "upload" {
				cancel()
			}
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("Run() did not finish")
	}

	indices := make(map[string]int)
	byKind := make(map[string]Event)
	for index, event := range events {
		if _, exists := indices[event.Kind]; !exists {
			indices[event.Kind] = index
			byKind[event.Kind] = event
		}
	}
	for _, kind := range []string{"archive", "queue", "uploading", "upload"} {
		if _, exists := byKind[kind]; !exists {
			t.Fatalf("missing %q event in %#v", kind, events)
		}
	}
	if !(indices["archive"] < indices["queue"] &&
		indices["queue"] < indices["uploading"] &&
		indices["uploading"] < indices["upload"]) {
		t.Fatalf("lifecycle event order = %#v", indices)
	}

	uploading := byKind["uploading"]
	if uploading.Status != scanupload.StatusUploading ||
		uploading.Checksum != checksum ||
		uploading.CapturedAt != scan.CapturedAt ||
		uploading.Realm != scan.Realm ||
		uploading.ScannerName != scan.ScannerName ||
		uploading.RowCount != scan.ExportedRowCount ||
		uploading.ItemCount != scan.ItemCount ||
		uploading.Attempts != 1 ||
		uploading.LastAttemptAt == "" ||
		uploading.HTTPStatus != 0 {
		t.Fatalf("uploading event = %#v", uploading)
	}

	uploaded := byKind["upload"]
	if uploaded.Status != scanupload.StatusUploaded ||
		uploaded.HTTPStatus != http.StatusCreated ||
		uploaded.ServerStatus != "accepted" ||
		uploaded.ScanID != 42 ||
		uploaded.ObservationRows != 2 ||
		uploaded.ItemCount != 2 ||
		uploaded.UploadedAt == "" {
		t.Fatalf("uploaded event = %#v", uploaded)
	}

	records, err := scanupload.ReadRecords(dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() error = %v", err)
	}
	if len(records) != 1 ||
		records[0].Status != scanupload.StatusUploaded ||
		records[0].ArchiveRows != scan.ExportedRowCount ||
		records[0].ArchiveItems != scan.ItemCount {
		t.Fatalf("persisted records = %#v", records)
	}
}

func TestRunEmitsRecoveryAfterSavedVariablesBecomesReadable(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "saved-variables", "v5", "fixtures", "valid.lua"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sourcePath := filepath.Join(t.TempDir(), "WoWMarkets.lua")
	if err := os.WriteFile(sourcePath, []byte("invalid SavedVariables"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = writer.Write([]byte(`{"error":"invalid_scan"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataDir := t.TempDir()
	kinds := make(chan string, 32)
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Config{
			FilePath:      sourcePath,
			DataDir:       dataDir,
			Interval:      10 * time.Millisecond,
			APIURL:        server.URL,
			APIToken:      "test-token",
			UploadTimeout: time.Second,
		}, func(event Event) {
			kinds <- event.Kind
		})
	}()

	waitForKind(t, kinds, "archive_error")
	if err := os.WriteFile(sourcePath, fixture, 0o600); err != nil {
		cancel()
		t.Fatalf("WriteFile() fixture error = %v", err)
	}
	waitForKind(t, kinds, "recovered")
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not stop")
	}
}

func waitForKind(t *testing.T, kinds <-chan string, expected string) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case kind := <-kinds:
			if kind == expected {
				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for %q", expected)
		}
	}
}
