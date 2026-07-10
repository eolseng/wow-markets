# WoW Markets Companion

Go/Wails v2 desktop app for macOS and Windows. The Go module root is this
directory. `frontend/dist/` is hand-written source, not generated output; keep
it vanilla HTML/CSS/ES modules with no bundler.

## Verify

- From the repository root: `make companion-check`.
- Focused Go: `cd companion && go test ./...`.
- Frontend: `node --check frontend/dist/app.js` and
  `node --test frontend/view-model.test.mjs`.
- Package: `make companion-build` (the app always targets the production API).

## Product invariants

- The companion never accepts account credentials. Users create a `wms1_…`
  installation token at `https://wowmarkets.app/account/installations` and
  paste it into the app. Store the full token only in the OS credential store;
  expose only its 13-character hint.
- Setup order is token → Anniversary WoW folder → addon marker → parseable
  SavedVariables. Missing addon or scan data is an onboarding state, not a
  runtime error. Recheck incomplete setup automatically.
- Treat WoW files and `refs/` as read-only. Never modify SavedVariables.
- Canonical scan JSON/checksums are cross-package contracts. Do not change
  archive serialization without coordinating addon, companion, API, and
  `docs/formats/saved-variables.md`.
- Derive upload counts and current/recent scan details from persisted archive
  and queue state, not process-local counters. Emit lifecycle progress before
  network uploads so the UI can show active work.
- Serialize token/setup changes with watcher stop/restart. A watcher using an
  old token must never overlap credential or upload-state mutation, and failure
  paths must restore a ready watcher.
- Preserve `WowMarketScan` data paths and the legacy `Wow Market Scan` keyring
  service as migration inputs. The user-visible product name is exactly
  `WoW Markets Companion`.

## Platform notes

- Closing the window hides it; tray/menu-bar operation continues.
- Launch-at-login code is split by build tags. Never mutate the real login
  configuration in tests; test payload/command construction instead.
- Keep CI package paths aligned with `wails.json` when renaming build output.
