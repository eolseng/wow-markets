---
name: develop-wow-markets-companion
description: Implement, debug, review, and verify the WoW Markets Companion desktop app. Use for changes under companion involving Go/Wails bindings, the hand-written frontend, token onboarding, WoW/addon/SavedVariables discovery, scan archiving and uploads, tray behavior, launch at login, packaging, or companion tests.
---

# Develop WoW Markets Companion

## Establish context

1. Read `AGENTS.md` at the repository root and `companion/AGENTS.md`.
2. Read only the package and contract docs touched by the request. Read
   `contracts/saved-variables/v5/specification.md` before changing scan formats. Public API
   contract documentation must remain sufficient for client changes without a
   private service checkout.
3. Treat `companion/frontend/dist/` as authored source. Do not introduce a
   bundler for an isolated UI change.

## Implement cohesively

1. Model user-visible states in Go first, then derive presentation in
   `frontend/dist/view-model.mjs`. Keep state priority explicit and table-test
   it.
2. Preserve token-only onboarding. Never add account credentials to the app or
   expose the complete installation token outside the OS credential store.
3. Inspect the WoW root, addon marker, and SavedVariables independently. Treat
   absent components as setup states; reserve errors for failed operations or
   invalid data.
4. Read upload facts from persisted archive/queue records. Carry safe metadata
   through lifecycle events before, during, and after an upload; never infer
   durable counts from in-memory event totals.
5. Keep the UI one-purpose and calm: gold for primary actions, cyan for active
   work, green for success, amber for retry/setup warnings, and red for terminal
   failures. Validate the default and minimum Wails window sizes.
6. Keep platform mutations behind build-tagged helpers. Make launch-at-login
   tests validate generated payloads or commands without changing the current
   user's login configuration.
7. Serialize setup and token mutations with watcher stop/restart. Never let a
   stale-token watcher overlap credential or durable queue changes, and ensure
   every failure path reconciles the watcher afterward.

## Protect contracts and compatibility

- Never write to WoW SavedVariables or edit `refs/`.
- Use `Interface/AddOns/WoWMarkets/WoWMarkets.toc`, `WoWMarkets.lua`, and
  `WOW_MARKETS_DB` as the primary addon discovery contract. Keep the former
  `WowMarketScan.lua`/`WOW_MARKET_SCAN_DB` pair only as an explicit migration
  input.
- Preserve canonical archive bytes and checksum behavior unless addon,
  companion, API, and format documentation change together.
- Keep `WoWMarkets` as the active config/data directory. Preserve the migration
  from the legacy `WowMarketScan` directory and `Wow Market Scan`
  credential-store service.
- Do not rewrite a platform startup registration from the background process it
  launched. Preserve show and quit requests received before Wails startup.
- Keep the user-visible name exactly `WoW Markets Companion` across Wails,
  frontend, tray/menu strings, CI paths, and release metadata.
- Local development targets loopback services by default. Use `httptest` and
  fixtures; do not send verification scans to production. Production origins
  belong only in explicit overrides or official linker-injected builds.

## Verify

Run focused tests while iterating, then finish from the repository root with:

```sh
make companion-check
```

For UI state changes, also run:

```sh
cd companion
node --test frontend/view-model.test.mjs
node --check frontend/dist/app.js
```

Build/package when platform metadata or startup behavior changes. Report any
platform behavior that still requires a disposable macOS/Windows login test.

## Maintain this skill

When companion architecture, addon discovery, data paths, contracts,
platform behavior, or verification steps change, update this skill in the
same change. Invoke `$skill-creator`, regenerate `agents/openai.yaml` if its
interface metadata is stale, and run `quick_validate.py` before finishing.
