# Wow Market Scan Companion

Wails v2 desktop app (macOS and Windows) that gets addon captures into WoW
Markets. It signs in to the API, enrolls the device as an uploader
installation, discovers the WoW Anniversary `WowMarketScan.lua` SavedVariables
file, and then watches it in the background: every new scan is archived
locally as canonical gzip JSON and uploaded to `POST /v1/scans`.

The API URL is hardcoded to production (`https://wow-markets.onrender.com` in
`config.go`) — there is no dev/staging switch, so `wails dev` also talks to
production.

This directory is the repository's Go module. `internal/` holds the scan-file
parsing, validation, archiving, and upload packages; `third_party/` vendors a
systray fork used by the Windows tray icon; `testdata/` holds the shared
SavedVariables fixture.

## Run

```sh
cd companion
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev    # run
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build  # package to build/bin/
go test ./...
```

Or install the CLI once (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`)
and use `wails dev` / `wails build`.

The UI in `frontend/dist` is hand-written vanilla HTML/JS/CSS committed
directly — there is no npm install or bundler step. It calls Go through the
Wails runtime (`window.go.main.App`) and polls `Snapshot()` for state.

## Local data

Everything lives under the OS user config directory
(macOS: `~/Library/Application Support/WowMarketScan`,
Windows: `%AppData%/WowMarketScan`):

- `config.json` — email, installation name, installation token prefix
  (non-secret), scan file path, WoW install path.
- `data/scans/<sha256>.json.gz` — immutable scan archives, deduplicated by
  checksum in `data/state.json`. Archives are kept after upload; the server
  stores only derived data, so these are the raw replay source.
- `data/uploads.json` — upload queue state.

Secrets are not in `config.json`: the refresh token and installation token are
stored in the macOS Keychain / Windows Credential Manager (service
`Wow Market Scan`).

The watcher polls the SavedVariables file every 5 seconds, never writes to it,
and retries failed uploads with exponential backoff (5s doubling to 15m).
Checksum duplicates from the server count as success.

## Runtime behavior

- macOS shows a native menu-bar status item (no Dock icon); Windows uses a
  notification-area tray icon. Closing the window hides it; the watcher keeps
  running. The icon menu has Show/Hide Window and Quit.
- Account sign-in and installation enrollment are separate steps. Signing out
  keeps the enrolled uploader active; removing enrollment deletes the
  installation token and stops uploads.
- Scan-file discovery checks the `WOW_MARKET_SCAN_FILE` environment variable
  first, then the configured WoW folder, then standard install paths, globbing
  `_anniversary_/WTF/Account/*/SavedVariables/WowMarketScan.lua` and selecting
  the newest file that parses.

## Distribution

The `Companion Build` GitHub Actions workflow builds macOS arm64 and Windows
x64 zips as artifacts on every PR and push to `main` (plus manual dispatch).
Automatic public distribution is on the roadmap; builds are currently shared
manually.

### macOS signing

CI ad-hoc signs macOS artifacts when Apple secrets are absent. For a
notarized, Gatekeeper-friendly build, configure repository secrets:

- `APPLE_DEVELOPER_ID_CERTIFICATE_BASE64` — base64 `.p12` export of the
  Developer ID Application certificate.
- `APPLE_DEVELOPER_ID_CERTIFICATE_PASSWORD` — password for the export.
- `APPLE_DEVELOPER_ID_APPLICATION` — optional codesign identity; defaults to
  the first Developer ID Application identity in the certificate.
- `APPLE_CODESIGN_KEYCHAIN_PASSWORD` — optional temporary CI keychain password.
- `APPLE_ID`, `APPLE_APP_SPECIFIC_PASSWORD`, `APPLE_TEAM_ID` — notarization.

With secrets present, CI signs with hardened runtime, notarizes, staples, and
verifies Gatekeeper assessment before uploading the artifact.

## Manual test checklist

1. Start with `wails dev`.
2. Confirm the status/tray icon appears immediately.
3. Sign in with a web-app email/password, then enroll the installation.
4. Confirm scan-file auto-detection reports ready when a
   `WowMarketScan.lua` exists (or select the WoW folder manually).
5. Confirm the Uploads page shows archived, queued, uploaded, and failed
   counts.
6. Close the window; confirm the process keeps running, `Show Window`
   restores it, and Quit exits.
