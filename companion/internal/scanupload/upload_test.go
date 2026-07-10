package scanupload

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanarchive"
)

func TestProcessDueUploadsAcceptedScan(t *testing.T) {
	var authorization string
	var contentType string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		authorization = request.Header.Get("Authorization")
		contentType = request.Header.Get("Content-Type")
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		body = string(payload)
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
			"checksum":"abc123",
			"scan_id":1,
			"submission_id":"submission-1",
			"status":"accepted",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	queued, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if len(queued) != 1 || !queued[0].Queued {
		t.Fatalf("Queue() = %#v", queued)
	}

	results, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	record := results[0].Record
	if record.Status != StatusUploaded ||
		record.ServerStatus != "accepted" ||
		record.Attempts != 1 ||
		record.ScanID != 1 ||
		record.SubmissionID != "submission-1" ||
		record.ObservationRows != 200 {
		t.Fatalf("uploaded record = %#v", record)
	}
	if authorization != "Bearer test-token" {
		t.Fatalf("Authorization = %q", authorization)
	}
	if contentType != "application/gzip" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if body != "archive bytes" {
		t.Fatalf("body = %q", body)
	}

	statePayload, err := os.ReadFile(filepath.Join(agent.dataDir, "uploads.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var saved state
	if err := json.Unmarshal(statePayload, &saved); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if saved.Uploads["abc123"].Status != StatusUploaded {
		t.Fatalf("saved record = %#v", saved.Uploads["abc123"])
	}
}

func TestProcessDueTreatsDuplicateAsUploaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{
			"checksum":"abc123",
			"scan_id":1,
			"submission_id":"submission-2",
			"status":"duplicate",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	results, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if len(results) != 1 ||
		results[0].Record.Status != StatusUploaded ||
		results[0].Record.ServerStatus != "duplicate" {
		t.Fatalf("results = %#v", results)
	}
}

func TestProcessDueRetriesTransientFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		attempts++
		if attempts == 1 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			_, _ = writer.Write([]byte(`{"error":"database_unavailable"}`))
			return
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
			"checksum":"abc123",
			"scan_id":1,
			"submission_id":"submission-1",
			"status":"accepted",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	agent.now = func() time.Time { return now }
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	first, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if len(first) != 1 ||
		first[0].Record.Status != StatusFailed ||
		!first[0].Record.Retryable ||
		first[0].Record.NextAttemptAt != "2026-06-13T12:00:05Z" {
		t.Fatalf("first result = %#v", first)
	}

	notDue, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() not due error = %v", err)
	}
	if len(notDue) != 0 {
		t.Fatalf("not due results = %#v", notDue)
	}

	now = now.Add(5 * time.Second)
	second, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() retry error = %v", err)
	}
	if len(second) != 1 ||
		second[0].Record.Status != StatusUploaded ||
		second[0].Record.Attempts != 2 {
		t.Fatalf("second result = %#v", second)
	}
}

func TestProcessDueDoesNotRetryPermanentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = writer.Write([]byte(`{"error":"invalid_scan","detail":"bad row"}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	first, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if len(first) != 1 ||
		first[0].Record.Status != StatusFailed ||
		first[0].Record.Retryable ||
		first[0].Record.NextAttemptAt != "" ||
		!strings.Contains(first[0].Record.LastError, "invalid_scan: bad row") {
		t.Fatalf("first result = %#v", first)
	}

	second, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() second error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second results = %#v", second)
	}
}

func TestProcessDueLimitUploadsAtMostLimit(t *testing.T) {
	attempts := 0
	responses := []string{"abc123", "def456"}
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		if attempts >= len(responses) {
			t.Fatalf("unexpected upload attempt %d", attempts+1)
		}
		checksum := responses[attempts]
		attempts++
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
			"checksum":"` + checksum + `",
			"scan_id":1,
			"submission_id":"submission-1",
			"status":"accepted",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, firstArchivePath := newTestAgent(t, server.URL, "abc123")
	secondArchivePath := filepath.Join(agent.dataDir, "def456.json.gz")
	if err := os.WriteFile(secondArchivePath, []byte("archive bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := agent.Queue([]scanarchive.Record{
		{Checksum: "abc123", ArchivePath: firstArchivePath},
		{Checksum: "def456", ArchivePath: secondArchivePath},
	})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	first, err := agent.ProcessDueLimit(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProcessDueLimit() error = %v", err)
	}
	if len(first) != 1 || first[0].Record.Checksum != "abc123" {
		t.Fatalf("first result = %#v", first)
	}
	currentState, err := agent.loadState()
	if err != nil {
		t.Fatalf("loadState() error = %v", err)
	}
	if currentState.Uploads["abc123"].Status != StatusUploaded ||
		currentState.Uploads["def456"].Status != StatusPending {
		t.Fatalf("state after first upload = %#v", currentState.Uploads)
	}

	second, err := agent.ProcessDueLimit(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProcessDueLimit() second error = %v", err)
	}
	if len(second) != 1 || second[0].Record.Checksum != "def456" {
		t.Fatalf("second result = %#v", second)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestQueueSkipsTruncatedScan(t *testing.T) {
	agent, archivePath := newTestAgent(t, "http://127.0.0.1:1", "abc123")
	results, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
		Truncated:   true,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if len(results) != 1 || results[0].Skipped != "truncated" {
		t.Fatalf("Queue() = %#v", results)
	}
	if _, err := os.Stat(filepath.Join(agent.dataDir, "uploads.json")); !os.IsNotExist(err) {
		t.Fatalf("uploads.json error = %v, want not exist", err)
	}
}

func TestReadRecordsReturnsPersistedRecordsNewestFirst(t *testing.T) {
	agent, firstArchivePath := newTestAgent(t, "http://127.0.0.1:1", "abc123")
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	agent.now = func() time.Time { return now }
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:     "abc123",
		ArchivePath:  firstArchivePath,
		CapturedAt:   1781350243,
		ImportedAt:   "2026-06-13T12:00:00Z",
		Region:       "eu",
		Realm:        "Spineshatter",
		Market:       "eu-spineshatter-alliance",
		Faction:      "Alliance",
		AuctionHouse: "faction",
		ScannerName:  "Examplechar",
		ScannerRealm: "Spineshatter",
		RowCount:     200,
		ItemCount:    25,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	secondArchivePath := filepath.Join(agent.dataDir, "def456.json.gz")
	if err := os.WriteFile(secondArchivePath, []byte("archive bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	now = now.Add(time.Minute)
	_, err = agent.Queue([]scanarchive.Record{{
		Checksum:    "def456",
		ArchivePath: secondArchivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() second error = %v", err)
	}

	records, err := ReadRecords(agent.dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() error = %v", err)
	}
	if len(records) != 2 || records[0].Checksum != "def456" || records[1].Checksum != "abc123" {
		t.Fatalf("ReadRecords() = %#v", records)
	}
	first := records[1]
	if first.CapturedAt != 1781350243 ||
		first.ArchiveRows != 200 ||
		first.ArchiveItems != 25 ||
		first.Realm != "Spineshatter" ||
		first.ScannerName != "Examplechar" {
		t.Fatalf("persisted archive metadata = %#v", first)
	}

	agentRecords, err := agent.Records()
	if err != nil {
		t.Fatalf("Records() error = %v", err)
	}
	if len(agentRecords) != 2 || agentRecords[0].Checksum != "def456" {
		t.Fatalf("Records() = %#v", agentRecords)
	}
}

func TestReadRecordsDoesNotCreateState(t *testing.T) {
	dataDir := t.TempDir()
	records, err := ReadRecords(dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("ReadRecords() = %#v", records)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "uploads.json")); !os.IsNotExist(err) {
		t.Fatalf("uploads.json error = %v, want not exist", err)
	}
}

func TestProcessDueWithProgressPersistsUploadingBeforeRequest(t *testing.T) {
	var requestReceived atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		requestReceived.Store(true)
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
			"checksum":"abc123",
			"scan_id":1,
			"submission_id":"submission-1",
			"status":"accepted",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
		CapturedAt:  1781350243,
		RowCount:    200,
		ItemCount:   25,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}

	progressCalls := 0
	results, err := agent.ProcessDueWithProgress(
		context.Background(),
		func(record Record) {
			progressCalls++
			if requestReceived.Load() {
				t.Fatal("progress callback ran after the HTTP request")
			}
			if record.Status != StatusUploading ||
				record.Attempts != 1 ||
				record.ArchiveRows != 200 ||
				record.ArchiveItems != 25 {
				t.Fatalf("progress record = %#v", record)
			}
			persisted, readErr := agent.Records()
			if readErr != nil {
				t.Fatalf("Records() in progress callback error = %v", readErr)
			}
			if len(persisted) != 1 || persisted[0].Status != StatusUploading {
				t.Fatalf("persisted records in callback = %#v", persisted)
			}
		},
	)
	if err != nil {
		t.Fatalf("ProcessDueWithProgress() error = %v", err)
	}
	if progressCalls != 1 || !requestReceived.Load() {
		t.Fatalf("progressCalls = %d, requestReceived = %t", progressCalls, requestReceived.Load())
	}
	if len(results) != 1 || results[0].Record.Status != StatusUploaded {
		t.Fatalf("results = %#v", results)
	}
}

func TestResetFailedAuthorizationRequeuesRejectedUpload(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		attempts++
		if attempts == 1 {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(`{"error":"invalid_installation_token"}`))
			return
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
			"checksum":"abc123",
			"scan_id":1,
			"submission_id":"submission-1",
			"status":"accepted",
			"rows":200,
			"items":25,
			"price_snapshots":20
		}`))
	}))
	defer server.Close()

	agent, archivePath := newTestAgent(t, server.URL, "abc123")
	_, err := agent.Queue([]scanarchive.Record{{
		Checksum:    "abc123",
		ArchivePath: archivePath,
	}})
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	first, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() error = %v", err)
	}
	if len(first) != 1 ||
		first[0].Record.Status != StatusFailed ||
		first[0].Record.Retryable ||
		first[0].Record.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("first result = %#v", first)
	}
	failedRecords, err := ReadRecords(agent.dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() failed-state error = %v", err)
	}
	if len(failedRecords) != 1 ||
		failedRecords[0].Status != StatusFailed ||
		failedRecords[0].HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("failed records = %#v", failedRecords)
	}

	reset, err := ResetFailedAuthorization(agent.dataDir)
	if err != nil {
		t.Fatalf("ResetFailedAuthorization() error = %v", err)
	}
	if reset != 1 {
		t.Fatalf("ResetFailedAuthorization() = %d, want 1", reset)
	}
	records, err := ReadRecords(agent.dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() error = %v", err)
	}
	if len(records) != 1 ||
		records[0].Status != StatusPending ||
		records[0].HTTPStatus != 0 ||
		records[0].LastError != "" {
		t.Fatalf("reset record = %#v", records)
	}

	second, err := agent.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("ProcessDue() after reset error = %v", err)
	}
	if len(second) != 1 || second[0].Record.Status != StatusUploaded {
		t.Fatalf("second result = %#v", second)
	}
}

func TestResetFailedAuthorizationLeavesOtherFailuresUntouched(t *testing.T) {
	dataDir := t.TempDir()
	err := writeState(dataDir, state{
		Version: stateVersion,
		Uploads: map[string]Record{
			"unauthorized": {
				Checksum:   "unauthorized",
				Status:     StatusFailed,
				HTTPStatus: http.StatusUnauthorized,
				LastError:  "API returned 401 invalid_installation_token",
			},
			"legacy-forbidden": {
				Checksum:  "legacy-forbidden",
				Status:    StatusFailed,
				LastError: "API returned 403 installation_disabled",
			},
			"invalid-scan": {
				Checksum:   "invalid-scan",
				Status:     StatusFailed,
				HTTPStatus: http.StatusUnprocessableEntity,
				LastError:  "API returned 422 invalid_scan: bad row",
			},
		},
	})
	if err != nil {
		t.Fatalf("writeState() error = %v", err)
	}

	reset, err := ResetFailedAuthorization(dataDir)
	if err != nil {
		t.Fatalf("ResetFailedAuthorization() error = %v", err)
	}
	if reset != 2 {
		t.Fatalf("ResetFailedAuthorization() = %d, want 2", reset)
	}
	records, err := ReadRecords(dataDir)
	if err != nil {
		t.Fatalf("ReadRecords() error = %v", err)
	}
	byChecksum := make(map[string]Record, len(records))
	for _, record := range records {
		byChecksum[record.Checksum] = record
	}
	if byChecksum["unauthorized"].Status != StatusPending ||
		byChecksum["legacy-forbidden"].Status != StatusPending {
		t.Fatalf("authorization records = %#v", byChecksum)
	}
	invalid := byChecksum["invalid-scan"]
	if invalid.Status != StatusFailed ||
		invalid.HTTPStatus != http.StatusUnprocessableEntity ||
		invalid.LastError == "" {
		t.Fatalf("invalid scan record = %#v", invalid)
	}
}

func newTestAgent(t *testing.T, apiURL, checksum string) (*Agent, string) {
	t.Helper()
	dataDir := t.TempDir()
	archivePath := filepath.Join(dataDir, checksum+".json.gz")
	if err := os.WriteFile(archivePath, []byte("archive bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	agent, err := New(dataDir, apiURL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return agent, archivePath
}
