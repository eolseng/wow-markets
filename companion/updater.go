package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

const (
	updatePublicKeyBase64 = "92xOZ5+HUUc84qIBQB1DhrGsUfx+f5/rWrMOEJ4xA18="
	defaultUpdateOrigin   = "https://updates.wowmarkets.app"
	trustedReleaseOrigin  = "https://github.com/eolseng/wow-markets/releases/download/"
	updateCheckInterval   = 6 * time.Hour
	maximumAppcastBytes   = 2 << 20
	maximumArtifactBytes  = 256 << 20

	updateStatusDisabled    = "disabled"
	updateStatusCurrent     = "current"
	updateStatusChecking    = "checking"
	updateStatusDownloading = "downloading"
	updateStatusAvailable   = "available"
	updateStatusReady       = "ready"
	updateStatusDeferred    = "deferred"
	updateStatusOffline     = "offline"
	updateStatusError       = "error"
)

const defaultUpdateChannel = updatefeed.ChannelStable

const windowsSilentInstallArguments = "/S"

var errNoPromotedUpdate = errors.New("no release has been promoted to this update channel")

type UpdaterSnapshot struct {
	AvailableVersion string `json:"available_version"`
	Channel          string `json:"channel"`
	CurrentVersion   string `json:"current_version"`
	Enabled          bool   `json:"enabled"`
	LastCheckedAt    string `json:"last_checked_at"`
	Mandatory        bool   `json:"mandatory"`
	Message          string `json:"message"`
	Progress         int    `json:"progress"`
	ReadyToInstall   bool   `json:"ready_to_install"`
	Status           string `json:"status"`
}

type updateConfiguration struct {
	Origin     *url.URL
	FeedURL    string
	PublicKey  ed25519.PublicKey
	Target     updatefeed.Target
	AssetName  string
	Channel    updatefeed.Channel
	Version    string
	StagingDir string
}

type platformUpdater interface {
	Start(feedURL string) error
	SetFeedURL(feedURL string) error
	Check() error
	Install(path string) error
	Close()
	ManagesDownloads() bool
}

func (app *App) startUpdater() {
	configuration, err := app.updateConfiguration()
	if err != nil {
		app.setUpdaterError(updateStatusError, err)
		return
	}
	if configuration == nil {
		return
	}

	native := newPlatformUpdater()
	if err := native.Start(configuration.FeedURL); err != nil {
		app.setUpdaterError(updateStatusError, err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	app.mu.Lock()
	if app.quitting {
		app.mu.Unlock()
		cancel()
		native.Close()
		return
	}
	app.nativeUpdater = native
	app.updateCancel = cancel
	app.updateDone = done
	app.updater.Enabled = true
	app.updater.Channel = string(configuration.Channel)
	app.updater.Status = updateStatusCurrent
	app.updater.Message = "The companion is up to date"
	app.mu.Unlock()
	app.emitSnapshot()

	go func() {
		defer close(done)
		app.runUpdateCheck(ctx, false)
		ticker := time.NewTicker(updateCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				app.runUpdateCheck(ctx, false)
			}
		}
	}()
}

func (app *App) stopUpdater() {
	app.mu.Lock()
	cancel := app.updateCancel
	done := app.updateDone
	native := app.nativeUpdater
	app.updateCancel = nil
	app.updateDone = nil
	app.nativeUpdater = nil
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
	if native != nil {
		native.Close()
	}
}

func (app *App) CheckForUpdates() (UpdaterSnapshot, error) {
	configuration, err := app.updateConfiguration()
	if err != nil {
		return app.updaterSnapshot(), err
	}
	if configuration == nil {
		return app.updaterSnapshot(), errors.New("updates are available only in official builds")
	}
	app.runUpdateCheck(context.Background(), true)
	return app.updaterSnapshot(), nil
}

func (app *App) SetUpdateChannel(value string) (UpdaterSnapshot, error) {
	channel, err := updatefeed.ParseChannel(value)
	if err != nil {
		return app.updaterSnapshot(), err
	}
	app.setupMu.Lock()
	app.mu.Lock()
	config := app.config
	config.UpdateChannel = string(channel)
	config.DeferredUpdateVersion = ""
	writable := app.configWritable
	app.mu.Unlock()
	if !writable {
		app.setupMu.Unlock()
		return app.updaterSnapshot(), errors.New("settings cannot be changed because config.json could not be loaded")
	}
	if err := saveConfig(config); err != nil {
		app.setupMu.Unlock()
		return app.updaterSnapshot(), err
	}
	app.mu.Lock()
	app.config = config
	app.stagedUpdatePath = ""
	app.updater.Channel = string(channel)
	app.updater.Status = updateStatusChecking
	app.updater.Message = "Switching update channel"
	app.updater.ReadyToInstall = false
	native := app.nativeUpdater
	app.mu.Unlock()
	app.setupMu.Unlock()

	configuration, err := app.updateConfiguration()
	if err != nil {
		return app.updaterSnapshot(), err
	}
	if configuration != nil && native != nil {
		if err := native.SetFeedURL(configuration.FeedURL); err != nil {
			return app.updaterSnapshot(), err
		}
	}
	app.emitSnapshot()
	go app.runUpdateCheck(context.Background(), false)
	return app.updaterSnapshot(), nil
}

func (app *App) DeferUpdate() (UpdaterSnapshot, error) {
	app.setupMu.Lock()
	app.mu.Lock()
	version := app.updater.AvailableVersion
	if version == "" {
		app.mu.Unlock()
		app.setupMu.Unlock()
		return app.updaterSnapshot(), errors.New("there is no update to defer")
	}
	if app.updater.Mandatory {
		app.mu.Unlock()
		app.setupMu.Unlock()
		return app.updaterSnapshot(), errors.New("this compatibility or security update cannot be deferred")
	}
	config := app.config
	config.DeferredUpdateVersion = version
	writable := app.configWritable
	app.mu.Unlock()
	if !writable {
		app.setupMu.Unlock()
		return app.updaterSnapshot(), errors.New("settings cannot be changed because config.json could not be loaded")
	}
	if err := saveConfig(config); err != nil {
		app.setupMu.Unlock()
		return app.updaterSnapshot(), err
	}
	app.mu.Lock()
	app.config = config
	app.updater.Status = updateStatusDeferred
	app.updater.Message = "Update deferred; scans and uploads will continue"
	app.mu.Unlock()
	app.setupMu.Unlock()
	app.emitSnapshot()
	return app.updaterSnapshot(), nil
}

func (app *App) InstallUpdate() error {
	app.mu.Lock()
	native := app.nativeUpdater
	path := app.stagedUpdatePath
	status := app.updater.Status
	app.mu.Unlock()
	if native == nil {
		return errors.New("the platform updater is unavailable")
	}
	if native.ManagesDownloads() {
		return native.Check()
	}
	if (status != updateStatusReady && status != updateStatusDeferred) || path == "" {
		return errors.New("the update has not finished downloading")
	}
	if err := app.prepareForUpdateRelaunch(); err != nil {
		app.setUpdaterError(updateStatusError, err)
		return err
	}
	if err := native.Install(path); err != nil {
		app.cancelUpdateRelaunch()
		app.setUpdaterError(updateStatusError, err)
		return err
	}
	app.Quit()
	return nil
}

func (app *App) runUpdateCheck(ctx context.Context, manual bool) {
	app.updateMu.Lock()
	defer app.updateMu.Unlock()
	configuration, err := app.updateConfiguration()
	if err != nil {
		app.setUpdaterError(updateStatusError, err)
		return
	}
	if configuration == nil {
		return
	}
	app.setUpdaterState(func(state *UpdaterSnapshot) {
		state.Enabled = true
		state.Channel = string(configuration.Channel)
		state.Status = updateStatusChecking
		state.Message = "Checking for updates"
		state.Progress = 0
		state.ReadyToInstall = false
	})

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	release, err := fetchUpdate(checkCtx, http.DefaultClient, *configuration)
	if err != nil {
		if errors.Is(err, errNoPromotedUpdate) {
			app.setUpdaterState(func(state *UpdaterSnapshot) {
				state.Status = updateStatusCurrent
				state.Message = fmt.Sprintf("No release has been promoted to the %s channel yet", configuration.Channel)
				state.AvailableVersion = ""
				state.Mandatory = false
				state.Progress = 0
				state.ReadyToInstall = false
				state.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
			})
			return
		}
		status := updateStatusError
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || isTransportError(err) {
			status = updateStatusOffline
		}
		app.setUpdaterError(status, err)
		return
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	if release == nil {
		app.setUpdaterState(func(state *UpdaterSnapshot) {
			state.Status = updateStatusCurrent
			state.Message = "The companion is up to date"
			state.AvailableVersion = ""
			state.Mandatory = false
			state.Progress = 0
			state.ReadyToInstall = false
			state.LastCheckedAt = checkedAt
		})
		return
	}

	app.mu.Lock()
	deferred := app.config.DeferredUpdateVersion == release.Version
	native := app.nativeUpdater
	app.mu.Unlock()
	if deferred && !release.Mandatory && !manual {
		app.setUpdaterState(func(state *UpdaterSnapshot) {
			state.Status = updateStatusDeferred
			state.Message = "Update deferred; scans and uploads will continue"
			state.AvailableVersion = release.Version
			state.Mandatory = false
			state.LastCheckedAt = checkedAt
		})
		return
	}
	if native != nil && native.ManagesDownloads() {
		app.setUpdaterState(func(state *UpdaterSnapshot) {
			state.Status = updateStatusAvailable
			state.Message = "An update is available and Sparkle is preparing it"
			state.AvailableVersion = release.Version
			state.Mandatory = release.Mandatory
			state.ReadyToInstall = false
			state.LastCheckedAt = checkedAt
		})
		if manual {
			if err := native.Check(); err != nil {
				app.setUpdaterError(updateStatusError, err)
			}
		}
		return
	}

	app.setUpdaterState(func(state *UpdaterSnapshot) {
		state.Status = updateStatusDownloading
		state.Message = "Downloading the verified update in the background"
		state.AvailableVersion = release.Version
		state.Mandatory = release.Mandatory
		state.ReadyToInstall = false
		state.LastCheckedAt = checkedAt
	})
	downloadCtx, downloadCancel := context.WithTimeout(ctx, 15*time.Minute)
	defer downloadCancel()
	path, err := downloadUpdate(downloadCtx, http.DefaultClient, *configuration, *release)
	if err != nil {
		app.setUpdaterError(updateStatusError, err)
		return
	}
	app.mu.Lock()
	app.stagedUpdatePath = path
	app.updater.Status = updateStatusReady
	app.updater.Message = "Update verified and ready to install"
	app.updater.Progress = 100
	app.updater.ReadyToInstall = true
	app.mu.Unlock()
	app.emitSnapshot()
}

func (app *App) updateConfiguration() (*updateConfiguration, error) {
	app.mu.Lock()
	channelValue := app.config.UpdateChannel
	stagingRoot := app.dataDir
	app.mu.Unlock()
	if channelValue == "" {
		channelValue = string(defaultUpdateChannel)
	}
	channel, err := updatefeed.ParseChannel(channelValue)
	if err != nil {
		return nil, err
	}
	originValue := strings.TrimSpace(officialUpdateOrigin)
	if originValue == "" && os.Getenv("WOW_MARKETS_ENABLE_UPDATES") == "1" {
		originValue = strings.TrimSpace(os.Getenv("WOW_MARKETS_UPDATE_ORIGIN"))
	}
	if originValue == "" {
		return nil, nil
	}
	origin, err := validateUpdateOrigin(originValue, strings.TrimSpace(officialUpdateOrigin) != "")
	if err != nil {
		return nil, err
	}
	publicKey, err := base64.StdEncoding.DecodeString(updatePublicKeyBase64)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("compiled update public key is invalid")
	}
	target, assetName, supported := platformUpdateTarget()
	if !supported {
		return nil, nil
	}
	if stagingRoot == "" {
		stagingRoot, err = companionDataDir()
		if err != nil {
			return nil, err
		}
	}
	feedURL := fmt.Sprintf("%s/companion/%s/%s-%s.xml", strings.TrimRight(origin.String(), "/"), channel, target.OS, target.Arch)
	return &updateConfiguration{
		Origin:     origin,
		FeedURL:    feedURL,
		PublicKey:  ed25519.PublicKey(publicKey),
		Target:     target,
		AssetName:  assetName,
		Channel:    channel,
		Version:    companionVersion(),
		StagingDir: filepath.Join(stagingRoot, "updates"),
	}, nil
}

func fetchUpdate(ctx context.Context, client *http.Client, configuration updateConfiguration) (*updatefeed.Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, configuration.FeedURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check update feed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: server returned %s", errNoPromotedUpdate, response.Status)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check update feed: server returned %s", response.Status)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumAppcastBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read update feed: %w", err)
	}
	if len(payload) > maximumAppcastBytes {
		return nil, errors.New("update feed exceeds the size limit")
	}
	releases, err := updatefeed.ParseSigned(payload, configuration.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("verify update feed: %w", err)
	}
	release, err := updatefeed.Select(releases, configuration.Version, configuration.Target)
	if err != nil || release == nil {
		return release, err
	}
	if err := validateRelease(*release, configuration); err != nil {
		return nil, err
	}
	return release, nil
}

func downloadUpdate(ctx context.Context, client *http.Client, configuration updateConfiguration, release updatefeed.Release) (string, error) {
	if release.Length <= 0 || release.Length > maximumArtifactBytes {
		return "", errors.New("update artifact size is outside the allowed range")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download update: server returned %s", response.Status)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, release.Length+1))
	if err != nil {
		return "", fmt.Errorf("read update: %w", err)
	}
	if int64(len(payload)) != release.Length {
		return "", fmt.Errorf("downloaded update has %d bytes, expected %d", len(payload), release.Length)
	}
	if err := updatefeed.VerifyArtifact(payload, release.Signature, configuration.PublicKey); err != nil {
		return "", err
	}
	if err := os.MkdirAll(configuration.StagingDir, 0o700); err != nil {
		return "", fmt.Errorf("create update staging directory: %w", err)
	}
	path := filepath.Join(configuration.StagingDir, configuration.AssetName)
	temporary := path + ".download"
	if err := os.WriteFile(temporary, payload, 0o700); err != nil {
		return "", fmt.Errorf("stage update: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return "", fmt.Errorf("commit staged update: %w", err)
	}
	return path, nil
}

func validateUpdateOrigin(value string, official bool) (*url.URL, error) {
	origin, err := url.Parse(strings.TrimRight(value, "/"))
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("update origin is invalid")
	}
	if official {
		if origin.Scheme != "https" || origin.Host != "updates.wowmarkets.app" || origin.Path != "" {
			return nil, errors.New("official update origin must be https://updates.wowmarkets.app")
		}
	} else if !(origin.Scheme == "http" && (origin.Hostname() == "127.0.0.1" || origin.Hostname() == "localhost")) && origin.Scheme != "https" {
		return nil, errors.New("development update origin must use HTTPS or loopback HTTP")
	}
	return origin, nil
}

func validateRelease(release updatefeed.Release, configuration updateConfiguration) error {
	parsed, err := url.Parse(release.URL)
	if err != nil {
		return err
	}
	expectedPath := fmt.Sprintf("/eolseng/wow-markets/releases/download/companion-v%s/%s", release.Version, configuration.AssetName)
	if parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.Path != expectedPath || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("update %s uses an untrusted or non-immutable asset URL", release.Version)
	}
	if !strings.HasPrefix(release.URL, trustedReleaseOrigin) {
		return fmt.Errorf("update %s uses an untrusted release origin", release.Version)
	}
	return nil
}

func (app *App) updaterSnapshot() UpdaterSnapshot {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.updater
}

func (app *App) setUpdaterState(change func(*UpdaterSnapshot)) {
	app.mu.Lock()
	change(&app.updater)
	app.mu.Unlock()
	app.emitSnapshot()
}

func (app *App) setUpdaterError(status string, err error) {
	app.setUpdaterState(func(state *UpdaterSnapshot) {
		state.Status = status
		state.Progress = 0
		if status == updateStatusOffline {
			state.Message = "Updates could not be checked; scans and uploads will continue"
		} else {
			state.Message = "Update verification failed: " + err.Error()
		}
		state.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	})
}

func isTransportError(err error) bool {
	var urlError *url.Error
	return errors.As(err, &urlError)
}
