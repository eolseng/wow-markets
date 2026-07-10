package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/scanfile"
	"github.com/eolseng/wow-markets/companion/internal/scanupload"
	"github.com/eolseng/wow-markets/companion/internal/watchagent"
	"github.com/eolseng/wow-markets/companion/internal/wowinstall"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const snapshotEventName = "companion:snapshot"

type App struct {
	mu      sync.Mutex
	setupMu sync.Mutex

	ctx         context.Context
	windowShown bool
	quitting    bool

	initializing   bool
	startupPhase   string
	configWritable bool
	config         companionConfig
	token          string
	dataDir        string
	discoveries    []wowinstall.Candidate
	wowDetected    bool
	addonDetected  bool
	addonPath      string

	launchAtLogin          bool
	launchAtLoginSupported bool

	setupMonitorCancel context.CancelFunc
	setupMonitorDone   chan struct{}

	watcherCancel context.CancelFunc
	watcherDone   chan struct{}
	running       bool

	activityKind string
	lastMessage  string
	lastError    string
	lastEventAt  string

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

func NewApp() *App {
	return &App{
		initializing: true,
		startupPhase: "Starting WoW Markets Companion",
		windowShown:  !launchedInBackground(),
	}
}

func (app *App) startup(ctx context.Context) {
	showPendingWindow, quitPending := app.attachStartupContext(ctx)
	if quitPending {
		runtime.Quit(ctx)
		return
	}
	showPendingWindow = showPendingWindow && launchedInBackground()
	if showPendingWindow {
		runtime.WindowShow(ctx)
		activateVisibleWindow()
	}

	startStatusItem(app)
	go app.initialize()
}

func (app *App) attachStartupContext(ctx context.Context) (showWindow, quit bool) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.ctx = ctx
	return app.windowShown, app.quitting
}

func (app *App) shutdown(context.Context) {
	app.mu.Lock()
	app.ctx = nil
	app.quitting = true
	app.mu.Unlock()

	app.stopSetupMonitor()
	_ = app.stopWatcher()
	stopStatusItem()
}

func (app *App) beforeClose(ctx context.Context) bool {
	app.mu.Lock()
	quitting := app.quitting
	app.windowShown = false
	app.mu.Unlock()

	if quitting {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

func (app *App) ShowWindow() {
	app.mu.Lock()
	ctx := app.ctx
	app.windowShown = true
	app.mu.Unlock()

	if ctx != nil {
		runtime.WindowShow(ctx)
		activateVisibleWindow()
	}
}

func (app *App) HideWindow() {
	app.mu.Lock()
	ctx := app.ctx
	app.windowShown = false
	app.mu.Unlock()

	if ctx != nil {
		runtime.WindowHide(ctx)
	}
}

func (app *App) Quit() {
	app.mu.Lock()
	ctx := app.ctx
	app.quitting = true
	app.mu.Unlock()

	app.stopSetupMonitor()
	_ = app.stopWatcher()
	stopStatusItem()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

type Snapshot struct {
	APIURL                 string                 `json:"api_url"`
	ActivityKind           string                 `json:"activity_kind"`
	AddonDetected          bool                   `json:"addon_detected"`
	AddonPath              string                 `json:"addon_path"`
	ArchivedCount          int                    `json:"archived_count"`
	Configured             bool                   `json:"configured"`
	CurrentStep            string                 `json:"current_step"`
	CurrentUpload          *ScanSummary           `json:"current_upload,omitempty"`
	DataDir                string                 `json:"data_dir"`
	Discoveries            []wowinstall.Candidate `json:"discoveries"`
	TokenStored            bool                   `json:"token_stored"`
	FailedCount            int                    `json:"failed_count"`
	Initializing           bool                   `json:"initializing"`
	InstallationsURL       string                 `json:"installations_url"`
	LastArchiveAt          string                 `json:"last_archive_at"`
	LastDetected           *ScanSummary           `json:"last_detected,omitempty"`
	LastError              string                 `json:"last_error"`
	LastEventAt            string                 `json:"last_event_at"`
	LastMessage            string                 `json:"last_message"`
	LastUpload             *ScanSummary           `json:"last_upload,omitempty"`
	LastUploadAt           string                 `json:"last_upload_at"`
	LaunchAtLogin          bool                   `json:"launch_at_login"`
	LaunchAtLoginSupported bool                   `json:"launch_at_login_supported"`
	QueuedCount            int                    `json:"queued_count"`
	Ready                  bool                   `json:"ready"`
	RecentUploads          []ScanSummary          `json:"recent_uploads"`
	Running                bool                   `json:"running"`
	SavedVariablesDetected bool                   `json:"saved_variables_detected"`
	ScanFileCount          int                    `json:"scan_file_count"`
	ScanFilePath           string                 `json:"scan_file_path"`
	SelectedAccount        string                 `json:"selected_account"`
	StartupPhase           string                 `json:"startup_phase"`
	TokenPrefix            string                 `json:"token_prefix"`
	UploadedCount          int                    `json:"uploaded_count"`
	UploadingCount         int                    `json:"uploading_count"`
	UploadFailure          *ScanSummary           `json:"upload_failure,omitempty"`
	Version                string                 `json:"version"`
	WowDetected            bool                   `json:"wow_detected"`
	WowInstallPath         string                 `json:"wow_install_path"`
}

type InstallationTokenRequest struct {
	Token string `json:"token"`
}

func (app *App) initialize() {
	app.setStartupPhase("Loading settings")
	config, configErr := loadConfig()

	app.setStartupPhase("Checking the secure token")
	token, tokenErr := loadInstallationToken()

	app.setStartupPhase("Preparing local scan storage")
	dataDir, dataDirErr := companionDataDir()

	app.mu.Lock()
	app.config = config
	app.configWritable = configErr == nil
	app.token = token
	app.dataDir = dataDir
	if token != "" {
		app.config.TokenPrefix = installationTokenPrefix(token)
	}
	app.mu.Unlock()

	for _, startupErr := range []error{configErr, tokenErr, dataDirErr} {
		if startupErr != nil {
			app.setError(startupErr)
		}
	}

	app.setStartupPhase("Finding World of Warcraft")
	if _, err := app.refreshSetup(); err != nil {
		app.setError(err)
	}

	app.setStartupPhase("Loading upload history")
	if err := app.reloadActivity(); err != nil {
		app.setError(err)
	}

	app.setStartupPhase("Checking startup preferences")
	launchSupported := platformLaunchAtLoginSupported()
	launchEnabled := false
	if launchSupported {
		var err error
		launchEnabled, err = platformLaunchAtLoginEnabled()
		if err != nil {
			app.setError(err)
		} else if shouldRefreshLaunchAtLogin(launchEnabled, launchedInBackground()) {
			// Refresh registrations after an app move or update.
			// A background launch must not modify the startup entry that is
			// currently being processed by the operating system.
			if err := platformSetLaunchAtLogin(true); err != nil {
				app.setError(err)
			}
		}
	}

	app.mu.Lock()
	app.launchAtLoginSupported = launchSupported
	app.launchAtLogin = launchEnabled
	app.initializing = false
	app.startupPhase = ""
	if app.quitting {
		app.mu.Unlock()
		return
	}
	if app.lastMessage == "" {
		app.lastMessage = "Setup checked"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	app.mu.Unlock()

	app.emitSnapshot()
	app.startSetupMonitor()
	app.setupMu.Lock()
	_, _ = app.startWatcher()
	app.setupMu.Unlock()
}

func (app *App) Snapshot() Snapshot {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.snapshotLocked()
}

func (app *App) RefreshSetup() (Snapshot, error) {
	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	_, err := app.refreshSetupLocked()
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	_, _ = app.startWatcher()
	app.emitSnapshot()
	return app.Snapshot(), nil
}

func (app *App) AutoDetectWowFolder() (Snapshot, error) {
	root, found := wowinstall.FindInstallRoot("")
	if !found {
		err := errors.New("World of Warcraft Anniversary was not found in a standard location")
		app.setError(err)
		return app.Snapshot(), err
	}
	return app.SetWowFolder(root)
}

func (app *App) SelectWowFolder() (Snapshot, error) {
	app.mu.Lock()
	ctx := app.ctx
	currentPath := app.config.WowInstallPath
	app.mu.Unlock()
	if ctx == nil {
		return app.Snapshot(), errors.New("application is not ready")
	}
	path, err := runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		DefaultDirectory: currentPath,
		Title:            "Select the World of Warcraft folder",
	})
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if path == "" {
		return app.Snapshot(), nil
	}
	return app.SetWowFolder(path)
}

func (app *App) SetWowFolder(path string) (Snapshot, error) {
	path = wowinstall.NormalizeInstallRoot(path)
	inspection, err := wowinstall.InspectInstall(path)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if !inspection.AnniversaryPresent {
		err := fmt.Errorf("%s does not contain the _anniversary_ game folder", path)
		app.setError(err)
		return app.Snapshot(), err
	}

	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	if err := app.stopWatcher(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	defer func() { _, _ = app.startWatcher() }()
	app.mu.Lock()
	config := app.config
	config.WowInstallPath = path
	config.ScanFilePath = ""
	writable := app.configWritable
	app.mu.Unlock()
	if !writable {
		err := errors.New("settings cannot be changed because config.json could not be loaded")
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := saveConfig(config); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.mu.Lock()
	app.config = config
	app.mu.Unlock()

	snapshot, err := app.refreshSetupLocked()
	if err == nil {
		app.recordActivity("setup", "World of Warcraft folder updated", "")
		_, _ = app.startWatcher()
	}
	app.emitSnapshot()
	return snapshot, err
}

func (app *App) SetScanFile(path string) (Snapshot, error) {
	path = filepath.Clean(path)
	if _, err := scanfile.Load(path, scanfile.DefaultVariableName); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	if err := app.stopWatcher(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	defer func() { _, _ = app.startWatcher() }()
	app.mu.Lock()
	config := app.config
	config.ScanFilePath = path
	for _, candidate := range app.discoveries {
		if filepath.Clean(candidate.Path) == path && candidate.InstallPath != "" {
			config.WowInstallPath = candidate.InstallPath
			break
		}
	}
	writable := app.configWritable
	app.mu.Unlock()
	if !writable {
		err := errors.New("settings cannot be changed because config.json could not be loaded")
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := saveConfig(config); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.mu.Lock()
	app.config = config
	app.mu.Unlock()

	snapshot, err := app.refreshSetupLocked()
	if err == nil {
		app.recordActivity("setup", "Scan file selected", "")
		_, _ = app.startWatcher()
	}
	app.emitSnapshot()
	return snapshot, err
}

func (app *App) SetInstallationToken(request InstallationTokenRequest) (Snapshot, error) {
	token, err := normalizeInstallationToken(request.Token)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	if err := app.stopWatcher(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	defer func() { _, _ = app.startWatcher() }()
	if err := saveInstallationToken(token); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.mu.Lock()
	previousToken := app.token
	app.token = token
	app.config.TokenPrefix = installationTokenPrefix(token)
	config := app.config
	writable := app.configWritable
	app.mu.Unlock()
	if !writable {
		err := errors.New("settings cannot be changed because config.json could not be loaded")
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := saveConfig(config); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if previousToken != token {
		if _, err := scanupload.ResetFailedAuthorization(app.dataDirectory()); err != nil {
			app.setError(err)
			return app.Snapshot(), err
		}
	}

	app.recordActivity("token", "Installation token stored securely", "")
	if err := app.reloadActivity(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	_, _ = app.startWatcher()
	app.emitSnapshot()
	return app.Snapshot(), nil
}

func (app *App) RemoveInstallationToken() (Snapshot, error) {
	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	if err := app.stopWatcher(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	defer func() { _, _ = app.startWatcher() }()
	if err := deleteInstallationToken(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.mu.Lock()
	app.token = ""
	app.config.TokenPrefix = ""
	config := app.config
	writable := app.configWritable
	app.mu.Unlock()
	if writable {
		if err := saveConfig(config); err != nil {
			app.setError(err)
			return app.Snapshot(), err
		}
	}
	app.recordActivity("token", "Installation token forgotten on this device", "")
	app.emitSnapshot()
	return app.Snapshot(), nil
}

func (app *App) OpenInstallationsPage() error {
	app.mu.Lock()
	ctx := app.ctx
	app.mu.Unlock()
	if ctx == nil {
		return errors.New("application is not ready")
	}
	runtime.BrowserOpenURL(ctx, installationsPageURL)
	return nil
}

func (app *App) SetLaunchAtLogin(enabled bool) (Snapshot, error) {
	if !platformLaunchAtLoginSupported() {
		err := errors.New("launch at login is supported on macOS and Windows")
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := platformSetLaunchAtLogin(enabled); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	actual, err := platformLaunchAtLoginEnabled()
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.mu.Lock()
	app.launchAtLogin = actual
	app.launchAtLoginSupported = true
	app.lastError = ""
	app.lastMessage = map[bool]string{true: "Launch at login enabled", false: "Launch at login disabled"}[actual]
	app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	app.mu.Unlock()
	app.emitSnapshot()
	return app.Snapshot(), nil
}

func (app *App) startWatcher() (Snapshot, error) {
	app.mu.Lock()
	if app.running {
		snapshot := app.snapshotLocked()
		app.mu.Unlock()
		return snapshot, nil
	}
	ready := app.readyLocked()
	config := app.config
	token := app.token
	dataDir := app.dataDir
	if app.initializing || app.quitting || !ready || dataDir == "" {
		snapshot := app.snapshotLocked()
		app.mu.Unlock()
		return snapshot, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	app.watcherCancel = cancel
	app.watcherDone = done
	app.running = true
	app.lastError = ""
	app.activityKind = "status"
	app.lastMessage = "Watcher starting"
	app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	app.mu.Unlock()

	go func() {
		defer close(done)
		err := watchagent.Run(ctx, watchagent.Config{
			APIURL:        productionAPIURL,
			APIToken:      token,
			DataDir:       dataDir,
			FilePath:      config.ScanFilePath,
			Interval:      5 * time.Second,
			UploadTimeout: 15 * time.Minute,
		}, app.handleWatcherEvent)
		app.mu.Lock()
		if app.watcherDone == done {
			app.watcherCancel = nil
			app.watcherDone = nil
			app.running = false
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			app.lastError = err.Error()
			app.lastMessage = err.Error()
			app.activityKind = "watcher_error"
			app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
		}
		app.mu.Unlock()
		app.emitSnapshot()
	}()
	app.emitSnapshot()
	return app.Snapshot(), nil
}

func (app *App) stopWatcher() error {
	app.mu.Lock()
	cancel := app.watcherCancel
	done := app.watcherDone
	app.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		app.restartWatcherWhenStopped(done)
		return errors.New("the watcher is still stopping; wait a moment and try again")
	}

	app.mu.Lock()
	if app.watcherDone == done {
		app.watcherCancel = nil
		app.watcherDone = nil
		app.running = false
	}
	app.mu.Unlock()
	return nil
}

func (app *App) restartWatcherWhenStopped(done <-chan struct{}) {
	go func() {
		<-done
		app.setupMu.Lock()
		defer app.setupMu.Unlock()
		_, _ = app.startWatcher()
	}()
}

func (app *App) handleWatcherEvent(event watchagent.Event) {
	app.mu.Lock()
	app.activityKind = event.Kind
	if event.Error != "" {
		app.lastError = event.Error
	} else if event.Kind != "status" {
		app.lastError = ""
	}
	if event.Message != "" {
		app.lastMessage = event.Message
	}
	if !event.Time.IsZero() {
		app.lastEventAt = event.Time.UTC().Format(time.RFC3339)
	}
	app.mu.Unlock()

	if err := app.reloadActivity(); err != nil {
		app.setError(err)
	}
	app.emitSnapshot()
}

func (app *App) refreshSetup() (Snapshot, error) {
	app.setupMu.Lock()
	defer app.setupMu.Unlock()
	return app.refreshSetupLocked()
}

func (app *App) refreshSetupLocked() (Snapshot, error) {
	app.mu.Lock()
	configuredRoot := app.config.WowInstallPath
	configuredScan := app.config.ScanFilePath
	originalConfig := app.config
	config := originalConfig
	writable := app.configWritable
	app.mu.Unlock()

	root, found := wowinstall.FindInstallRoot(configuredRoot)
	inspection := wowinstall.InstallInspection{
		InstallPath:     configuredRoot,
		AddonMarkerPath: wowinstall.AddonMarkerPath(configuredRoot),
		ScanFiles:       []wowinstall.Candidate{},
	}
	if found {
		var err error
		inspection, err = wowinstall.InspectInstall(root)
		if err != nil {
			return app.Snapshot(), err
		}
	}

	selectedScan := ""
	if candidateExists(inspection.ScanFiles, configuredScan) {
		selectedScan = configuredScan
	} else if len(inspection.ScanFiles) > 0 {
		selectedScan = inspection.ScanFiles[0].Path
	}
	if found {
		config.WowInstallPath = inspection.InstallPath
	}
	config.ScanFilePath = selectedScan
	changed := config.WowInstallPath != originalConfig.WowInstallPath ||
		config.ScanFilePath != originalConfig.ScanFilePath
	if changed && writable {
		if err := saveConfig(config); err != nil {
			return app.Snapshot(), err
		}
	}

	app.mu.Lock()
	app.config = config
	app.discoveries = append([]wowinstall.Candidate(nil), inspection.ScanFiles...)
	app.wowDetected = inspection.AnniversaryPresent
	app.addonDetected = inspection.AddonPresent
	app.addonPath = inspection.AddonMarkerPath
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	return snapshot, nil
}

func (app *App) startSetupMonitor() {
	app.mu.Lock()
	if app.quitting || app.setupMonitorCancel != nil {
		app.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	app.setupMonitorCancel = cancel
	app.setupMonitorDone = done
	app.mu.Unlock()

	go func() {
		defer close(done)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshot := app.Snapshot()
				if snapshot.Initializing {
					continue
				}
				if snapshot.Ready {
					if !snapshot.Running {
						app.setupMu.Lock()
						_, _ = app.startWatcher()
						app.setupMu.Unlock()
					}
					continue
				}
				before := snapshot.CurrentStep
				app.setupMu.Lock()
				updated, err := app.refreshSetupLocked()
				if err != nil {
					app.setupMu.Unlock()
					app.setError(err)
					continue
				}
				if before != updated.CurrentStep {
					app.emitSnapshot()
				}
				if updated.Ready {
					_, _ = app.startWatcher()
				}
				app.setupMu.Unlock()
			}
		}
	}()
}

func (app *App) stopSetupMonitor() {
	app.mu.Lock()
	cancel := app.setupMonitorCancel
	done := app.setupMonitorDone
	app.setupMonitorCancel = nil
	app.setupMonitorDone = nil
	app.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}

func (app *App) setStartupPhase(phase string) {
	app.mu.Lock()
	app.startupPhase = phase
	app.mu.Unlock()
	app.emitSnapshot()
}

func (app *App) recordActivity(kind, message, errorMessage string) {
	app.mu.Lock()
	app.activityKind = kind
	app.lastMessage = message
	app.lastError = errorMessage
	app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	app.mu.Unlock()
}

func (app *App) setError(err error) {
	if err == nil {
		return
	}
	app.recordActivity("error", err.Error(), err.Error())
	app.emitSnapshot()
}

func (app *App) emitSnapshot() {
	app.mu.Lock()
	ctx := app.ctx
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, snapshotEventName, snapshot)
	}
}

func (app *App) snapshotLocked() Snapshot {
	ready := app.readyLocked()
	return Snapshot{
		APIURL:                 productionAPIURL,
		ActivityKind:           app.activityKind,
		AddonDetected:          app.addonDetected,
		AddonPath:              app.addonPath,
		ArchivedCount:          app.archivedCount,
		Configured:             ready,
		CurrentStep:            currentStep(app.initializing, app.token != "", app.wowDetected, app.addonDetected, app.config.ScanFilePath != ""),
		CurrentUpload:          cloneScanSummary(app.currentUpload),
		DataDir:                app.dataDir,
		Discoveries:            append([]wowinstall.Candidate(nil), app.discoveries...),
		TokenStored:            app.token != "",
		FailedCount:            app.failedCount,
		Initializing:           app.initializing,
		InstallationsURL:       installationsPageURL,
		LastArchiveAt:          app.lastArchiveAt,
		LastDetected:           cloneScanSummary(app.lastDetected),
		LastError:              app.lastError,
		LastEventAt:            app.lastEventAt,
		LastMessage:            app.lastMessage,
		LastUpload:             cloneScanSummary(app.lastUpload),
		LastUploadAt:           app.lastUploadAt,
		LaunchAtLogin:          app.launchAtLogin,
		LaunchAtLoginSupported: app.launchAtLoginSupported,
		QueuedCount:            app.queuedCount,
		Ready:                  ready,
		RecentUploads:          append([]ScanSummary(nil), app.recentUploads...),
		Running:                app.running,
		SavedVariablesDetected: app.config.ScanFilePath != "",
		ScanFileCount:          len(app.discoveries),
		ScanFilePath:           app.config.ScanFilePath,
		SelectedAccount:        selectedAccount(app.discoveries, app.config.ScanFilePath),
		StartupPhase:           app.startupPhase,
		TokenPrefix:            app.config.TokenPrefix,
		UploadedCount:          app.uploadedCount,
		UploadingCount:         app.uploadingCount,
		UploadFailure:          cloneScanSummary(app.uploadFailure),
		Version:                companionVersion,
		WowDetected:            app.wowDetected,
		WowInstallPath:         app.config.WowInstallPath,
	}
}

func (app *App) readyLocked() bool {
	return app.token != "" &&
		app.wowDetected &&
		app.addonDetected &&
		app.config.ScanFilePath != ""
}

func (app *App) dataDirectory() string {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.dataDir
}

func candidateExists(candidates []wowinstall.Candidate, path string) bool {
	if path == "" {
		return false
	}
	path = filepath.Clean(path)
	for _, candidate := range candidates {
		if filepath.Clean(candidate.Path) == path {
			return true
		}
	}
	return false
}

func selectedAccount(candidates []wowinstall.Candidate, path string) string {
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	for _, candidate := range candidates {
		if filepath.Clean(candidate.Path) == path {
			return candidate.Account
		}
	}
	return ""
}

func currentStep(initializing, tokenStored, wowDetected, addonDetected, savedVariablesDetected bool) string {
	switch {
	case initializing:
		return "loading"
	case !tokenStored:
		return "token"
	case !wowDetected:
		return "wow"
	case !addonDetected:
		return "addon"
	case !savedVariablesDetected:
		return "saved_variables"
	default:
		return "ready"
	}
}
