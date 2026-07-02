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
