# Companion updater architecture

Status: accepted for implementation, 2026-07-11.

## Decision

The companion uses one signed appcast format and one Ed25519 update key, with
platform adapters selected by Go build tags:

- macOS uses Sparkle 2.9.4. The framework owns background download, atomic app
  replacement, relaunch, authentication, and its standard update UI.
- Windows uses the shared Go appcast verifier and background downloader, then
  launches the existing NSIS installer after the user chooses to restart. The
  installer continues to provide elevation and the expected unsigned-publisher
  prompt.

Stable and beta use separate signed feeds for each supported platform. This
keeps WinSparkle-compatible appcast metadata simple and prevents a client from
selecting an asset for another platform or architecture. Official builds trust
only `https://updates.wowmarkets.app/` for feeds and immutable release assets
under `https://github.com/eolseng/wow-markets/releases/download/`.

Appcasts and their enclosures are Ed25519-signed. The public key is compiled
into the application and the private key exists only in the approval-protected
GitHub `release` environment. The API cannot provide a feed URL or key.

## Spike results

Sparkle 2.9.4 supports the required Developer ID application update path,
signed feeds, Ed25519 enclosure signatures, automatic background downloads,
deferred installation, critical updates, and stable/beta channels. It can
update the ordinary Wails `.app` bundle without changing the DMG distribution
format. The framework is copied into `Contents/Frameworks` before the outer app
is signed and notarized.

WinSparkle 0.9.3 supports Ed25519 enclosure verification, NSIS launch, and
thread-safe shutdown callbacks. It does not expose a supported API for silently
staging an ordinary update and reporting a restart-ready state to the host UI.
Its appcast parser also does not implement Sparkle's per-item beta channel
filter. Using it would therefore either add a second user-visible update flow
or require a fork. The production Windows adapter instead reuses the small,
tested Go verifier/downloader and launches the same NSIS artifact WinSparkle
would launch.

## Rejected alternatives

- Replacing the macOS app directly from Go was rejected because permission,
  quarantine, rollback, code-signing validation, and relaunch are security-
  sensitive behavior already handled by Sparkle.
- Shipping WinSparkle and accepting its separate UI was rejected because it
  cannot provide the product's calm background-download and restart-ready
  state without a fork.
- Mutable `latest` asset URLs and API-supplied feed URLs were rejected. Update
  assets always use immutable version tags and clients enforce trusted origins.
- A single unsigned appcast was rejected because an attacker could otherwise
  change critical-update policy or platform selection even though they could
  not forge an enclosure signature.

## Lifecycle and persistence

Update checks must never stop scan archiving or uploads. Before Windows starts
an installer, the companion stops its watcher, releases tray resources, and
quits normally so the single-instance lock is released. Configuration, tokens,
archives, queue state, and launch-at-login registrations live outside the app
installation and are not replaced by either platform adapter.

The final release-candidate gate installs an exact first candidate on clean
macOS and Windows accounts and updates it to a second candidate. The test must
cover deferral, offline recovery, invalid signatures, interrupted downloads,
cancelled installs, downgrade rejection, preserved local state, tray shutdown,
single-instance release, and launch-at-login path stability.

