# WoW Markets Companion

Wails v2 desktop app for macOS and Windows that watches WoW Classic
Anniversary SavedVariables, archives new WoW Markets addon captures as
canonical gzip JSON, and uploads them to `POST /v1/scans`.

The companion does not handle account login. Users create an installation
token at <https://wowmarkets.app/account/installations>, paste it into the app,
and the full token is stored in macOS Keychain or Windows Credential Manager.

## Run and verify

```sh
make companion        # Wails dev; talks to the production API
make companion-check  # formatting, frontend tests, Go tests, and build
make companion-build  # package to companion/build/bin
```

The Go module root is `companion/`. `frontend/dist/` is hand-written vanilla
HTML, CSS, and JavaScript; there is no npm dependency or bundler step. The UI
uses Wails bindings at `window.go.main.App`, listens for lifecycle snapshots,
and retains polling as a recovery path.

## Startup and onboarding

Startup checks four independent prerequisites and focuses the first missing
one:

1. An installation token is stored.
2. a World of Warcraft installation with `_anniversary_` is detected.
3. `_anniversary_/Interface/AddOns/WowMarketScan/WowMarketScan.toc` exists.
4. a parseable
   `_anniversary_/WTF/Account/*/SavedVariables/WowMarketScan.lua` exists.

Incomplete setup is rechecked every three seconds. A correct installation with
no SavedVariables is an expected state: run an Auctionator full scan, then
`/reload` or log out. The watcher starts automatically once setup is complete.

## Upload state

The watcher polls the selected SavedVariables file every five seconds and
never writes to it. New scans are deduplicated by canonical checksum, archived,
queued, and uploaded sequentially. Retryable failures use exponential backoff
from five seconds to fifteen minutes. Server duplicates count as success.

Dashboard counts and current/recent scan details are rebuilt from durable
archive and upload records, so they survive restarts and accurately distinguish
waiting, uploading, uploaded, and failed scans. Replacing a rejected token
requeues scans that failed with HTTP 401 or 403.

## Local data and compatibility

Application data remains under the legacy `WowMarketScan` config directory to
preserve pre-1.0 archives and queues:

- macOS: `~/Library/Application Support/WowMarketScan`
- Windows: `%AppData%/WowMarketScan`

`config.json` stores non-secret paths and a token hint. `data/state.json` and
`data/scans/*.json.gz` are the canonical local archive; `data/uploads.json` is
the durable queue. The current credential-store service is
`WoW Markets Companion`; an existing token under legacy service
`Wow Market Scan` is migrated on first load.

## Background operation

Closing the window hides it while the menu-bar or notification-area icon and
watcher continue. Start at login is optional:

- macOS writes a per-user LaunchAgent that starts the current app executable
  with `--background`.
- Windows writes the per-user `Run` registry value and also uses
  `--background`.

Keep the installed app in a stable location before enabling this setting.
Manual launches are visible; login launches start hidden. A single-instance
lock prevents duplicate watchers.

## Distribution

The `Companion Build` workflow builds the macOS arm64 app and Windows x64
executable. macOS builds use Developer ID signing and notarization when secrets
are configured, otherwise ad-hoc signing. The macOS artifact is a disk image
with a branded Finder layout containing `WoW Markets Companion.app` and an
Applications link for drag-and-drop installation. The Windows artifact is an
unsigned Wails NSIS installer that installs the app in Program Files and
registers an uninstaller. Release artifacts use the `wow-markets-companion-*`
name.

## Manual smoke test

1. Start with no stored token and confirm the loading screen resolves to token
   onboarding with no email/password fields.
2. Open the installations page, paste a valid token, and confirm only its hint
   is displayed afterward.
3. Exercise missing WoW folder, missing addon, and missing SavedVariables in
   turn; confirm each has distinct guidance.
4. Using a legitimate scan in a disposable production-linked OS account, run
   `/reload` and confirm automatic detection, detected/uploading/success states,
   and row/item data. Never upload fixtures or synthetic scans: development
   builds target the production API, while automated upload coverage uses local
   test servers.
5. Close and restore the window from the tray/menu bar.
6. Enable and disable Start at login, then verify a login launch in a disposable
   OS account or VM.
7. On Windows, run the NSIS setup executable, verify the Program Files install,
   shortcuts, and Apps & features entry, then uninstall it and confirm companion
   data remains available for a later reinstall.
