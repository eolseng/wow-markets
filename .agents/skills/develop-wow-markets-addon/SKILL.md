---
name: develop-wow-markets-addon
description: Implement, debug, review, and verify the WoW Markets addon. Use for changes under addon involving Lua capture logic, Auctionator full-scan events, chat UX and slash commands, SavedVariables, queue rotation, Auction House classification, addon identity or packaging, and addon tests.
---

# Develop WoW Markets Addon

## Establish context

1. Read `AGENTS.md` at the repository root and `addon/README.md`.
2. Read `contracts/saved-variables/v5/specification.md` before changing the envelope or scan
   fields. Read `docs/client-architecture.md` when changing the
   Auctionator integration or capture lifecycle.
3. Treat `refs/Auctionator` and `data/` as read-only local context. Inspect the
   referenced Auctionator implementation when relying on its internal events.
   Never copy reference source into this repository.

## Preserve the addon identity

- Keep the installed addon at `addon/WoWMarkets` with `WoWMarkets.toc`.
- Keep the Lua namespace `WoWMarkets`, SavedVariables global
  `WOW_MARKETS_DB`, and generated file `WoWMarkets.lua` aligned with companion
  discovery.
- Keep `/wm` primary. `/wms` is a compatibility alias unless intentionally
  removed with documentation and migration guidance.
- Keep the user-visible name exactly `WoW Markets`.

## Implement capture changes safely

1. Subscribe through Auctionator's event bus. Its legacy full-scan completion
   payload is an internal contract, so re-check `refs/Auctionator` when the
   dependency changes.
2. Keep the synchronous completion listener small. Retain the raw payload and
   compact it in timer batches; do not transform a full scan in the event
   callback.
3. Export every row, omit owner names, and preserve item-string variants.
4. Freeze region, character, faction, and Auction House location when capture
   begins. Keep neutral classification consistent with the companion's
   independent validator.
5. Never place an in-progress scan in SavedVariables. Only append complete
   `ready` scans and preserve bounded queue rotation.

## Keep the UX small and truthful

- Use chat messages and slash commands rather than a separate addon UI unless
  the request clearly requires one.
- Distinguish waiting, preparing, ready-for-`/reload`, warnings, and errors.
- Tell users not to reload while timer-batched capture is active. After
  completion, explain that `/reload` or normal logout hands the scan to the
  companion.
- Report stored scans, not uploads: the addon cannot know companion archive or
  server upload state.
- Confirm destructive commands and surface same-session queue loss.

## Protect cross-package contracts

- Scan format 5 is shared by addon, companion, the service API, fixtures, and
  `contracts/saved-variables/v5/specification.md`. Coordinate all affected packages when its
  fields or semantics change.
- Preserve byte-stable canonical archive behavior downstream. An envelope or
  filename migration must not silently alter scan contents or checksums.
- Keep legacy `WowMarketScan.lua` handling in the companion as migration
  behavior; do not reintroduce that identity into new addon output.

## Verify

Run focused Lua tests while iterating, then from the repository root run:

```sh
make addon-check
```

Use Lua 5.1 compatibility. For Auctionator events, reload/logout behavior,
location APIs, or TOC changes, also report the remaining in-game checks. When
the SavedVariables contract changes, run the companion gate and record the
public commit required by the service consumer tests.

## Maintain this skill

When addon identity, workflows, contracts, commands, dependencies, or
verification steps change, update this skill in the same change. Invoke
`$skill-creator`, regenerate `agents/openai.yaml` if its interface metadata is
stale, and run `quick_validate.py` before finishing.
