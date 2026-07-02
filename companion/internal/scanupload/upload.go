package scanupload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanarchive"
)

const (
	stateVersion       = 1
	defaultBaseBackoff = 5 * time.Second
	defaultMaxBackoff  = 15 * time.Minute
	maxResponseBytes   = 1 << 20
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusUploading Status = "uploading"
	StatusUploaded  Status = "uploaded"
	StatusFailed    Status = "failed"
)

type Record struct {
	Checksum        string `json:"checksum"`
	ArchivePath     string `json:"archive_path"`
	Status          Status `json:"status"`
	Attempts        int    `json:"attempts"`
	QueuedAt        string `json:"queued_at"`
	LastAttemptAt   string `json:"last_attempt_at,omitempty"`
	NextAttemptAt   string `json:"next_attempt_at,omitempty"`
	UploadedAt      string `json:"uploaded_at,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	Retryable       bool   `json:"retryable"`
	ServerStatus    string `json:"server_status,omitempty"`
	ScanID          int64  `json:"scan_id,omitempty"`
	SubmissionID    string `json:"submission_id,omitempty"`
	ObservationRows int64  `json:"observation_rows,omitempty"`
	ItemCount       int    `json:"item_count,omitempty"`
	PriceLevels     int64  `json:"price_levels,omitempty"`
	PriceSnapshots  int64  `json:"price_snapshots,omitempty"`
}

type QueueResult struct {
	Checksum string
	Queued   bool
	Skipped  string
}

type Result struct {
	Record Record
}

type state struct {
	Version int               `json:"version"`
	Uploads map[string]Record `json:"uploads"`
}

type uploadResponse struct {
	Checksum       string `json:"checksum"`
	ScanID         int64  `json:"scan_id"`
	SubmissionID   string `json:"submission_id"`
	Status         string `json:"status"`
	Rows           int64  `json:"rows"`
	Items          int    `json:"items"`
	PriceLevels    int64  `json:"price_levels"`
	PriceSnapshots int64  `json:"price_snapshots"`
}

type problemResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

type Agent struct {
	dataDir     string
	apiURL      string
	token       string
	client      *http.Client
	now         func() time.Time
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

func New(dataDir, apiURL, token string, timeout time.Duration) (*Agent, error) {
	if dataDir == "" {
		return nil, errors.New("data directory is required")
	}
	if strings.TrimSpace(apiURL) == "" {
		return nil, errors.New("API URL is required")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("API token is required")
	}
	if timeout <= 0 {
		return nil, errors.New("upload timeout must be positive")
	}

	absoluteDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	return &Agent{
		dataDir:     absoluteDataDir,
		apiURL:      strings.TrimRight(strings.TrimSpace(apiURL), "/"),
		token:       strings.TrimSpace(token),
		client:      &http.Client{Timeout: timeout},
		now:         time.Now,
		baseBackoff: defaultBaseBackoff,
		maxBackoff:  defaultMaxBackoff,
	}, nil
}

func (agent *Agent) Queue(records []scanarchive.Record) ([]QueueResult, error) {
	currentState, err := agent.loadState()
	if err != nil {
		return nil, err
	}

	now := agent.now().UTC().Format(time.RFC3339)
	results := make([]QueueResult, 0, len(records))
	changed := false
	for _, archive := range records {
		result := QueueResult{Checksum: archive.Checksum}
		switch {
		case archive.Truncated:
			result.Skipped = "truncated"
		case archive.Checksum == "":
			result.Skipped = "missing checksum"
		case archive.ArchivePath == "":
			result.Skipped = "missing archive path"
		default:
			if _, exists := currentState.Uploads[archive.Checksum]; !exists {
				absoluteArchivePath, err := filepath.Abs(archive.ArchivePath)
				if err != nil {
					return nil, fmt.Errorf(
						"resolve archive path for %s: %w",
						archive.Checksum,
						err,
					)
				}
				currentState.Uploads[archive.Checksum] = Record{
					Checksum:    archive.Checksum,
					ArchivePath: absoluteArchivePath,
					Status:      StatusPending,
					QueuedAt:    now,
				}
				result.Queued = true
				changed = true
			}
		}
		results = append(results, result)
	}

	if changed {
		if err := agent.writeState(currentState); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (agent *Agent) ProcessDue(ctx context.Context) ([]Result, error) {
	return agent.ProcessDueLimit(ctx, 0)
}

func (agent *Agent) ProcessDueLimit(ctx context.Context, limit int) ([]Result, error) {
	currentState, err := agent.loadState()
	if err != nil {
		return nil, err
	}

	now := agent.now().UTC()
	recovered := false
	for checksum, record := range currentState.Uploads {
		if record.Status != StatusUploading {
			continue
		}
		record.Status = StatusFailed
		record.Retryable = true
		record.NextAttemptAt = now.Format(time.RFC3339)
		record.LastError = "previous upload was interrupted"
		currentState.Uploads[checksum] = record
		recovered = true
	}
	if recovered {
		if err := agent.writeState(currentState); err != nil {
			return nil, err
		}
	}

	checksums := dueChecksums(currentState, now)
	if limit > 0 && len(checksums) > limit {
		checksums = checksums[:limit]
	}
	results := make([]Result, 0, len(checksums))
	for _, checksum := range checksums {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		attemptedAt := agent.now().UTC()
		record := currentState.Uploads[checksum]
		record.Status = StatusUploading
		record.Attempts++
		record.LastAttemptAt = attemptedAt.Format(time.RFC3339)
		record.NextAttemptAt = ""
		record.LastError = ""
		currentState.Uploads[checksum] = record
		if err := agent.writeState(currentState); err != nil {
			return results, err
		}

		response, retryable, uploadErr := agent.upload(ctx, record)
		finishedAt := agent.now().UTC()
		if uploadErr == nil {
			record.Status = StatusUploaded
			record.UploadedAt = finishedAt.Format(time.RFC3339)
			record.Retryable = false
			record.ServerStatus = response.Status
			record.ScanID = response.ScanID
			record.SubmissionID = response.SubmissionID
			record.ObservationRows = response.Rows
			record.ItemCount = response.Items
			record.PriceLevels = response.PriceLevels
			record.PriceSnapshots = response.PriceSnapshots
		} else {
			record.Status = StatusFailed
			record.LastError = uploadErr.Error()
			record.Retryable = retryable
			if retryable {
				record.NextAttemptAt = finishedAt.Add(agent.backoff(record.Attempts)).
					Format(time.RFC3339)
			}
		}
		currentState.Uploads[checksum] = record
		if err := agent.writeState(currentState); err != nil {
			return results, err
		}
		results = append(results, Result{Record: record})
	}
	return results, nil
}

func (agent *Agent) upload(ctx context.Context, record Record) (uploadResponse, bool, error) {
	file, err := os.Open(record.ArchivePath)
	if err != nil {
		return uploadResponse{}, false, fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		agent.apiURL+"/v1/scans",
		file,
	)
	if err != nil {
		return uploadResponse{}, false, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+agent.token)
	request.Header.Set("Content-Type", "application/gzip")

	response, err := agent.client.Do(request)
	if err != nil {
		return uploadResponse{}, true, fmt.Errorf("send scan: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return uploadResponse{}, true, fmt.Errorf("read response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return uploadResponse{}, false, errors.New("API response exceeds size limit")
	}

	if response.StatusCode != http.StatusOK &&
		response.StatusCode != http.StatusCreated {
		return uploadResponse{}, isRetryableStatus(response.StatusCode), apiError(
			response.StatusCode,
			payload,
		)
	}

	var result uploadResponse
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return uploadResponse{}, false, fmt.Errorf("decode success response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return uploadResponse{}, false, errors.New("decode success response: trailing JSON value")
	}
	if result.Status != "accepted" && result.Status != "duplicate" {
		return uploadResponse{}, false, fmt.Errorf(
			"API returned unsupported success status %q",
			result.Status,
		)
	}
	if result.Checksum != record.Checksum {
		return uploadResponse{}, false, fmt.Errorf(
			"API checksum %s does not match queued checksum %s",
			result.Checksum,
			record.Checksum,
		)
	}
	return result, false, nil
}

func (agent *Agent) loadState() (state, error) {
	path := filepath.Join(agent.dataDir, "uploads.json")
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return state{Version: stateVersion, Uploads: map[string]Record{}}, nil
	}
	if err != nil {
		return state{}, fmt.Errorf("open upload state: %w", err)
	}
	defer file.Close()

	var result state
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return state{}, fmt.Errorf("decode upload state: %w", err)
	}
	if result.Version != stateVersion {
		return state{}, fmt.Errorf(
			"unsupported upload state version %d; expected %d",
			result.Version,
			stateVersion,
		)
	}
	if result.Uploads == nil {
		result.Uploads = map[string]Record{}
	}
	return result, nil
}

func (agent *Agent) writeState(value state) error {
	if err := os.MkdirAll(agent.dataDir, 0o700); err != nil {
		return fmt.Errorf("create importer data directory: %w", err)
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode upload state: %w", err)
	}
	payload = append(payload, '\n')
	if err := writeAtomic(filepath.Join(agent.dataDir, "uploads.json"), payload); err != nil {
		return fmt.Errorf("write upload state: %w", err)
	}
	return nil
}

func (agent *Agent) backoff(attempts int) time.Duration {
	delay := agent.baseBackoff
	for attempt := 1; attempt < attempts && delay < agent.maxBackoff; attempt++ {
		if delay > agent.maxBackoff/2 {
			return agent.maxBackoff
		}
		delay *= 2
	}
	if delay > agent.maxBackoff {
		return agent.maxBackoff
	}
	return delay
}

func dueChecksums(currentState state, now time.Time) []string {
	checksums := make([]string, 0, len(currentState.Uploads))
	for checksum, record := range currentState.Uploads {
		switch record.Status {
		case StatusPending:
			checksums = append(checksums, checksum)
		case StatusFailed:
			if !record.Retryable {
				continue
			}
			nextAttempt, err := time.Parse(time.RFC3339, record.NextAttemptAt)
			if err == nil && !nextAttempt.After(now) {
				checksums = append(checksums, checksum)
			}
		}
	}
	sort.Strings(checksums)
	return checksums
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

func apiError(status int, payload []byte) error {
	var problem problemResponse
	if err := json.Unmarshal(payload, &problem); err == nil && problem.Error != "" {
		if problem.Detail != "" {
			return fmt.Errorf("API returned %d %s: %s", status, problem.Error, problem.Detail)
		}
		return fmt.Errorf("API returned %d %s", status, problem.Error)
	}
	return fmt.Errorf("API returned HTTP %d", status)
}

func writeAtomic(path string, payload []byte) error {
	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".uploads-*.tmp")
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
