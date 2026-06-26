# Wow Market Scan Companion

This is the Wails v2 desktop companion for Wow Market Scan. It signs in to the
production API, enrolls the local installation, stores tokens in the OS
credential store, discovers Anniversary account scan files from the World of
Warcraft installation folder, and runs the uploader while the window is hidden.

## Run

Install the Wails v2 CLI once:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
```

If `wails` is not on your shell `PATH`, use `go run` instead:

```sh
cd companion
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev
```

If the CLI is on your `PATH`, the shorter command also works:

```sh
cd companion
wails dev
```

Build a packaged app:

```sh
cd companion
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build
```

## macOS Signing

The `Companion Build` GitHub Actions workflow ad-hoc signs macOS artifacts when
Apple signing secrets are not configured. To produce a Gatekeeper-friendly
notarized macOS artifact, configure these repository secrets:

- `APPLE_DEVELOPER_ID_CERTIFICATE_BASE64`: base64-encoded `.p12` export of the
  Developer ID Application certificate.
- `APPLE_DEVELOPER_ID_CERTIFICATE_PASSWORD`: password for the `.p12` export.
- `APPLE_DEVELOPER_ID_APPLICATION`: optional codesign identity, for example
  `Developer ID Application: Example Name (TEAMID)`. If omitted, CI uses the
  first Developer ID Application identity from the certificate.
- `APPLE_CODESIGN_KEYCHAIN_PASSWORD`: optional temporary CI keychain password.
- `APPLE_ID`: Apple ID used for notarization.
- `APPLE_APP_SPECIFIC_PASSWORD`: app-specific password for that Apple ID.
- `APPLE_TEAM_ID`: Apple Developer Team ID.

When those secrets are present, CI signs with hardened runtime, submits the app
to Apple's notary service, staples the notarization ticket, verifies Gatekeeper
assessment, then uploads the final zip artifact.

## Runtime Behavior

- macOS uses a native AppKit status item in the menu bar.
- Windows uses a notification-area tray icon near network and volume icons.
- Closing the window hides it and keeps the watcher running.
- `Show Window` restores the window.
- `Hide Window` hides the window while the process keeps running.
- `Quit Wow Market Scan` exits the process.
- Account sign-in and installation enrollment are separate steps.
- Signing out removes the account session but keeps an enrolled uploader active.
- Removing enrollment deletes the installation upload token and stops uploads.
- The main UI is a guided flow: sign in, enroll, then detect scan files.
- The dropdown navigates to Overview, Settings, and Uploads. Settings combines
  account, enrollment, and scan-file controls.
- Scan detection checks `WOW_MARKET_SCAN_FILE` first, then a configured WoW
  folder, then standard installation paths. It scans all Anniversary account
  SavedVariables folders and selects the newest valid `WowMarketScan.lua`.
- Config is stored under the OS user config directory in `WowMarketScan`.
- The account refresh token and installation token are stored with macOS
  Keychain or Windows Credential Manager through `github.com/zalando/go-keyring`.

## Test Checklist

1. Start with `wails dev` or the `go run ... dev` fallback.
2. Confirm the status/menu-bar icon appears immediately after startup.
3. Sign in with a web-app email/password.
4. Enroll the installation using the prefilled or edited device name.
5. Confirm scan-file auto-detection reports ready when `_anniversary_/WTF/Account/*/SavedVariables/WowMarketScan.lua` exists.
6. If auto-detection fails, choose the World of Warcraft installation folder and confirm detected accounts appear.
7. Confirm the Uploads page shows archived, queued, uploaded, and failed counts.
8. Close the window and confirm the process keeps running with the icon visible.
9. Use `Show Window` from the icon menu and confirm the window returns.
10. Use `Quit Wow Market Scan` from the icon menu and confirm the process exits.

If this fails on either target OS, capture the console output and whether the
failure happened at compile time, startup, enrollment, scan-file discovery,
close-to-background, show-window, upload, or quit.
