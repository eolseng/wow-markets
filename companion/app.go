package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/companionauth"
	"github.com/eolseng/wow-markets/companion/internal/scanfile"
	"github.com/eolseng/wow-markets/companion/internal/watchagent"
	"github.com/eolseng/wow-markets/companion/internal/wowinstall"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	mu            sync.Mutex
	ctx           context.Context
	windowShown   bool
	quitting      bool
	config        companionConfig
	token         string
	accessToken   string
	refreshToken  string
	authUser      companionauth.User
	dataDir       string
	discoveries   []wowinstall.Candidate
	watcherCancel context.CancelFunc
	watcherDone   chan struct{}
	running       bool
	lastMessage   string
	lastError     string
	lastEventAt   string
	archivedCount int
	queuedCount   int
	uploadedCount int
	failedCount   int
	lastArchiveAt string
	lastUploadAt  string
}

func NewApp() *App {
	return &App{windowShown: true}
}

func (app *App) startup(ctx context.Context) {
	app.mu.Lock()
	app.ctx = ctx
	app.windowShown = true
	app.mu.Unlock()

	app.initialize()
	startStatusItem(app)
}

func (app *App) shutdown(ctx context.Context) {
	app.mu.Lock()
	app.ctx = nil
	app.quitting = true
	app.mu.Unlock()

	app.stopWatcher()
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

	app.stopWatcher()
	stopStatusItem()
	if ctx != nil {
		runtime.Quit(ctx)
	}
}

type Snapshot struct {
	APIURL           string                 `json:"api_url"`
	ArchivedCount    int                    `json:"archived_count"`
	Configured       bool                   `json:"configured"`
	CurrentStep      string                 `json:"current_step"`
	DataDir          string                 `json:"data_dir"`
	Discoveries      []wowinstall.Candidate `json:"discoveries"`
	Email            string                 `json:"email"`
	Enrolled         bool                   `json:"enrolled"`
	FailedCount      int                    `json:"failed_count"`
	InstallationName string                 `json:"installation_name"`
	LastArchiveAt    string                 `json:"last_archive_at"`
	LastError        string                 `json:"last_error"`
	LastEventAt      string                 `json:"last_event_at"`
	LastMessage      string                 `json:"last_message"`
	LastUploadAt     string                 `json:"last_upload_at"`
	LoggedIn         bool                   `json:"logged_in"`
	QueuedCount      int                    `json:"queued_count"`
	Ready            bool                   `json:"ready"`
	Running          bool                   `json:"running"`
	ScanFileCount    int                    `json:"scan_file_count"`
	ScanFilePath     string                 `json:"scan_file_path"`
	SelectedAccount  string                 `json:"selected_account"`
	TokenPrefix      string                 `json:"token_prefix"`
	UploadedCount    int                    `json:"uploaded_count"`
	UserDisplayName  string                 `json:"user_display_name"`
	WowInstallPath   string                 `json:"wow_install_path"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type EnrollRequest struct {
	InstallationName string `json:"installation_name"`
}

func (app *App) initialize() {
	config, err := loadConfig()
	if err != nil {
		app.setError(fmt.Errorf("load config: %w", err))
	}
	token, err := loadInstallationToken()
	if err != nil {
		app.setError(fmt.Errorf("load installation token: %w", err))
	}
	refreshToken, err := loadRefreshToken()
	if err != nil {
		app.setError(fmt.Errorf("load account session: %w", err))
	}
	dataDir, err := companionDataDir()
	if err != nil {
		app.setError(err)
	}
	if config.InstallationName == "" {
		config.InstallationName = defaultInstallationName()
	}

	app.mu.Lock()
	app.config = config
	app.token = token
	app.refreshToken = refreshToken
	app.dataDir = dataDir
	if app.lastMessage == "" {
		app.lastMessage = "Ready"
	}
	app.mu.Unlock()

	_, _ = app.DiscoverScanFiles()
	_, _ = app.StartWatcher()
	if refreshToken != "" {
		go app.restoreAccountSession(refreshToken)
	}
}

func (app *App) Snapshot() Snapshot {
	app.mu.Lock()
	defer app.mu.Unlock()

	return app.snapshotLocked()
}

func (app *App) restoreAccountSession(refreshToken string) {
	if err := app.refreshAccountSessionWithToken(refreshToken); err != nil {
		app.mu.Lock()
		if app.refreshToken == refreshToken {
			app.accessToken = ""
			app.authUser = companionauth.User{}
			app.lastMessage = "Account sign-in needs attention"
			app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
		}
		app.mu.Unlock()
	}
}

func (app *App) refreshAccountSession() error {
	app.mu.Lock()
	refreshToken := app.refreshToken
	app.mu.Unlock()
	if refreshToken == "" {
		loaded, err := loadRefreshToken()
		if err != nil {
			return err
		}
		refreshToken = loaded
	}
	if refreshToken == "" {
		return errors.New("sign in before enrolling this installation")
	}
	return app.refreshAccountSessionWithToken(refreshToken)
}

func (app *App) refreshAccountSessionWithToken(refreshToken string) error {
	client, err := companionauth.New(productionAPIURL)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session, err := client.Refresh(ctx, refreshToken)
	if err != nil {
		return err
	}
	if err := saveRefreshToken(session.RefreshToken); err != nil {
		return err
	}

	app.mu.Lock()
	app.applySessionLocked(session)
	saveErr := saveConfig(app.config)
	if saveErr == nil {
		app.lastError = ""
		app.lastMessage = "Signed in"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	app.mu.Unlock()
	return saveErr
}

func (app *App) applySessionLocked(session companionauth.AuthSession) {
	app.accessToken = session.AccessToken
	app.refreshToken = session.RefreshToken
	app.authUser = session.User
	app.config.Email = session.User.Email
}

func (app *App) DiscoverScanFiles() (Snapshot, error) {
	app.mu.Lock()
	wowInstallPath := app.config.WowInstallPath
	app.mu.Unlock()

	candidates, err := wowinstall.DiscoverAnniversaryScanFiles(wowInstallPath)
	app.mu.Lock()
	defer app.mu.Unlock()
	if err != nil {
		app.lastError = err.Error()
		return app.snapshotLocked(), err
	}
	app.discoveries = candidates
	if len(candidates) > 0 {
		if app.config.ScanFilePath == "" || !candidateExists(candidates, app.config.ScanFilePath) {
			app.config.ScanFilePath = candidates[0].Path
		}
		if app.config.WowInstallPath == "" {
			app.config.WowInstallPath = candidates[0].InstallPath
		}
		if saveErr := saveConfig(app.config); saveErr != nil {
			app.lastError = saveErr.Error()
			return app.snapshotLocked(), saveErr
		}
	}
	app.lastError = ""
	app.lastMessage = fmt.Sprintf("Found %d scan file candidate(s)", len(candidates))
	app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	return app.snapshotLocked(), nil
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
		Title:            "Select World of Warcraft folder",
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
	if path == "" {
		err := errors.New("World of Warcraft folder is required")
		app.setError(err)
		return app.Snapshot(), err
	}
	if info, err := os.Stat(path); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	} else if !info.IsDir() {
		err := fmt.Errorf("%s is not a folder", path)
		app.setError(err)
		return app.Snapshot(), err
	}

	app.stopWatcher()
	candidates, err := wowinstall.DiscoverAnniversaryScanFiles(path)
	app.mu.Lock()
	app.config.WowInstallPath = path
	app.discoveries = candidates
	if err != nil {
		app.lastError = err.Error()
		snapshot := app.snapshotLocked()
		app.mu.Unlock()
		return snapshot, err
	}
	if len(candidates) == 0 {
		app.config.ScanFilePath = ""
		saveErr := saveConfig(app.config)
		app.lastError = "No WowMarketScan SavedVariables files found in this WoW installation"
		app.lastMessage = app.lastError
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
		if saveErr != nil {
			snapshot := app.snapshotLocked()
			app.mu.Unlock()
			return snapshot, saveErr
		}
		snapshot := app.snapshotLocked()
		app.mu.Unlock()
		return snapshot, errors.New(app.lastError)
	}
	app.config.ScanFilePath = candidates[0].Path
	if candidates[0].InstallPath != "" {
		app.config.WowInstallPath = candidates[0].InstallPath
	}
	saveErr := saveConfig(app.config)
	if saveErr == nil {
		app.lastError = ""
		app.lastMessage = fmt.Sprintf("Detected %d account scan file(s)", len(candidates))
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	if saveErr == nil {
		_, _ = app.StartWatcher()
	}
	return snapshot, saveErr
}

func (app *App) SelectScanFile() (Snapshot, error) {
	app.mu.Lock()
	ctx := app.ctx
	currentPath := app.config.ScanFilePath
	app.mu.Unlock()
	if ctx == nil {
		return app.Snapshot(), errors.New("application is not ready")
	}
	defaultDir := ""
	if currentPath != "" {
		defaultDir = filepath.Dir(currentPath)
	}
	path, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		DefaultDirectory: defaultDir,
		Filters: []runtime.FileFilter{{
			DisplayName: "WowMarketScan SavedVariables",
			Pattern:     "*.lua",
		}},
		Title: "Select WowMarketScan.lua",
	})
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if path == "" {
		return app.Snapshot(), nil
	}
	return app.SetScanFile(path)
}

func (app *App) SetScanFile(path string) (Snapshot, error) {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if _, err := scanfile.Load(path, scanfile.DefaultVariableName); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.stopWatcher()
	app.mu.Lock()
	app.config.ScanFilePath = path
	for _, candidate := range app.discoveries {
		if filepath.Clean(candidate.Path) == path && candidate.InstallPath != "" {
			app.config.WowInstallPath = candidate.InstallPath
			break
		}
	}
	err := saveConfig(app.config)
	if err == nil {
		app.lastError = ""
		app.lastMessage = "Scan file configured"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	if err != nil {
		app.setError(err)
		return snapshot, err
	}
	_, _ = app.StartWatcher()
	return app.Snapshot(), nil
}

func (app *App) Login(request LoginRequest) (Snapshot, error) {
	client, err := companionauth.New(productionAPIURL)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session, err := client.Login(ctx, request.Email, request.Password)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := saveRefreshToken(session.RefreshToken); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.mu.Lock()
	app.applySessionLocked(session)
	saveErr := saveConfig(app.config)
	if saveErr == nil {
		app.lastError = ""
		app.lastMessage = "Signed in"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	if saveErr != nil {
		app.setError(saveErr)
		return snapshot, saveErr
	}
	return app.Snapshot(), nil
}

func (app *App) Enroll(request EnrollRequest) (Snapshot, error) {
	installationName := request.InstallationName
	if installationName == "" {
		installationName = defaultInstallationName()
	}
	app.mu.Lock()
	accessToken := app.accessToken
	app.mu.Unlock()
	if accessToken == "" {
		if err := app.refreshAccountSession(); err != nil {
			app.setError(err)
			return app.Snapshot(), err
		}
		app.mu.Lock()
		accessToken = app.accessToken
		app.mu.Unlock()
	}
	if accessToken == "" {
		err := errors.New("sign in before enrolling this installation")
		app.setError(err)
		return app.Snapshot(), err
	}

	client, err := companionauth.New(productionAPIURL)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	created, err := client.CreateInstallation(
		ctx,
		accessToken,
		installationName,
	)
	if err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	if err := saveInstallationToken(created.Token); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}

	app.stopWatcher()
	app.mu.Lock()
	app.config.InstallationName = created.Installation.Name
	app.config.TokenPrefix = created.Installation.TokenPrefix
	app.token = created.Token
	saveErr := saveConfig(app.config)
	if saveErr == nil {
		app.lastError = ""
		app.lastMessage = "Installation enrolled"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	if saveErr != nil {
		app.setError(saveErr)
		return snapshot, saveErr
	}
	_, _ = app.StartWatcher()
	return app.Snapshot(), nil
}

func (app *App) SignOut() (Snapshot, error) {
	if err := deleteRefreshToken(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.mu.Lock()
	app.accessToken = ""
	app.refreshToken = ""
	app.authUser = companionauth.User{}
	err := saveConfig(app.config)
	if err == nil {
		app.lastError = ""
		if app.token != "" {
			app.lastMessage = "Signed out; uploader remains enrolled"
		} else {
			app.lastMessage = "Signed out"
		}
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	return snapshot, err
}

func (app *App) RemoveEnrollment() (Snapshot, error) {
	app.stopWatcher()
	if err := deleteInstallationToken(); err != nil {
		app.setError(err)
		return app.Snapshot(), err
	}
	app.mu.Lock()
	app.token = ""
	app.config.TokenPrefix = ""
	err := saveConfig(app.config)
	if err == nil {
		app.lastError = ""
		app.lastMessage = "Installation token removed"
		app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
	}
	snapshot := app.snapshotLocked()
	app.mu.Unlock()
	return snapshot, err
}

func (app *App) StartWatcher() (Snapshot, error) {
	app.mu.Lock()
	if app.running {
		snapshot := app.snapshotLocked()
		app.mu.Unlock()
		return snapshot, nil
	}
	config := app.config
	token := app.token
	dataDir := app.dataDir
	if config.ScanFilePath == "" || token == "" || dataDir == "" {
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
	app.lastMessage = "Starting watcher"
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
			VariableName:  "",
		}, app.handleWatcherEvent)
		app.mu.Lock()
		app.running = false
		if err != nil {
			app.lastError = err.Error()
			app.lastMessage = err.Error()
		}
		app.mu.Unlock()
	}()
	return app.Snapshot(), nil
}

func (app *App) StopWatcher() Snapshot {
	app.stopWatcher()
	return app.Snapshot()
}

func (app *App) stopWatcher() {
	app.mu.Lock()
	cancel := app.watcherCancel
	done := app.watcherDone
	app.watcherCancel = nil
	app.watcherDone = nil
	app.running = false
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

func (app *App) handleWatcherEvent(event watchagent.Event) {
	app.mu.Lock()
	defer app.mu.Unlock()
	switch event.Kind {
	case "archive":
		app.archivedCount++
		if !event.Time.IsZero() {
			app.lastArchiveAt = event.Time.UTC().Format(time.RFC3339)
		}
		app.lastError = ""
	case "queue":
		app.queuedCount++
		app.lastError = ""
	case "upload":
		app.uploadedCount++
		if !event.Time.IsZero() {
			app.lastUploadAt = event.Time.UTC().Format(time.RFC3339)
		}
		app.lastError = ""
	case "upload_error":
		app.failedCount++
	}
	if event.Error != "" {
		app.lastError = event.Error
	}
	if event.Message != "" {
		app.lastMessage = event.Message
	}
	if !event.Time.IsZero() {
		app.lastEventAt = event.Time.UTC().Format(time.RFC3339)
	}
}

func (app *App) setError(err error) {
	if err == nil {
		return
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.lastError = err.Error()
	app.lastMessage = err.Error()
	app.lastEventAt = time.Now().UTC().Format(time.RFC3339)
}

func (app *App) snapshotLocked() Snapshot {
	email := app.config.Email
	if app.authUser.Email != "" {
		email = app.authUser.Email
	}
	loggedIn := app.accessToken != ""
	enrolled := app.token != ""
	scanConfigured := app.config.ScanFilePath != ""
	ready := loggedIn && enrolled && scanConfigured
	return Snapshot{
		APIURL:           productionAPIURL,
		ArchivedCount:    app.archivedCount,
		Configured:       scanConfigured && enrolled,
		CurrentStep:      currentStep(loggedIn, enrolled, scanConfigured),
		DataDir:          app.dataDir,
		Discoveries:      append([]wowinstall.Candidate(nil), app.discoveries...),
		Email:            email,
		Enrolled:         enrolled,
		FailedCount:      app.failedCount,
		InstallationName: app.config.InstallationName,
		LastArchiveAt:    app.lastArchiveAt,
		LastError:        app.lastError,
		LastEventAt:      app.lastEventAt,
		LastMessage:      app.lastMessage,
		LastUploadAt:     app.lastUploadAt,
		LoggedIn:         loggedIn,
		QueuedCount:      app.queuedCount,
		Ready:            ready,
		Running:          app.running,
		ScanFileCount:    len(app.discoveries),
		ScanFilePath:     app.config.ScanFilePath,
		SelectedAccount:  selectedAccount(app.discoveries, app.config.ScanFilePath),
		TokenPrefix:      app.config.TokenPrefix,
		UploadedCount:    app.uploadedCount,
		UserDisplayName:  app.authUser.DisplayName,
		WowInstallPath:   app.config.WowInstallPath,
	}
}

func candidateExists(candidates []wowinstall.Candidate, path string) bool {
	path = filepath.Clean(path)
	for _, candidate := range candidates {
		if filepath.Clean(candidate.Path) == path {
			return true
		}
	}
	return false
}

func selectedAccount(candidates []wowinstall.Candidate, path string) string {
	path = filepath.Clean(path)
	for _, candidate := range candidates {
		if filepath.Clean(candidate.Path) == path {
			return candidate.Account
		}
	}
	return ""
}

func currentStep(loggedIn bool, enrolled bool, scanConfigured bool) string {
	switch {
	case !loggedIn:
		return "login"
	case !enrolled:
		return "enrollment"
	case !scanConfigured:
		return "scan"
	default:
		return "ready"
	}
}

func defaultInstallationName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "wow-market-scan"
	}
	return host
}
