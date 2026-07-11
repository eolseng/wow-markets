---
name: develop-wow-markets-addon
description: Implement, debug, review, release, and verify the WoW Markets addon. Use for changes under addon involving Lua capture logic, Auctionator full-scan events, chat UX and slash commands, SavedVariables, queue rotation, Auction House classification, addon identity, CurseForge or Wago distribution, packaging, release metadata, and addon tests.
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

## Release and distribute

1. Read `docs/releasing-addon.md` completely before changing a version,
   distribution ID, `.pkgmeta`, or `.github/workflows/addon-release.yml`.
2. Keep CurseForge project `1605493`, Wago project `qGZOdXNd`, interface
   `20505`, and the Auctionator dependency synchronized across the TOC,
   `.pkgmeta`, dashboards, tests, and companion links.
3. Use `addon-v<version>` tags. Use semantic `alpha` or `beta` labels for
   prereleases because the pinned BigWigs packager treats other tagged labels,
   including `rc`, as stable.
4. Never expose release secrets to pull requests. Keep `CF_API_KEY` and
   `WAGO_API_TOKEN` in the protected `addon-release` environment. Generate the
   Wago publishing token from the Wago Addons developer API-key page; updater
   API keys cannot publish.
5. Validate the BigWigs preflight archive before tagging. Publish only from an
   exact reviewed `main` commit, approve the protected environment, verify all
   three channels, then perform the distributed-archive in-game smoke test.
6. Never replace a published archive or tag. Fix forward with a higher addon
   version.
7. Treat uploads as sequential external mutations. If one channel succeeds
   and a later channel fails, verify the channel states and retry only the
   missing destinations; never blindly rerun against the successful channel.
   Revoke any credential exposed in output before continuing.

## Maintain this skill

When addon identity, workflows, contracts, commands, dependencies, or
verification steps change, update this skill in the same change. Invoke
`$skill-creator`, regenerate `agents/openai.yaml` if its interface metadata is
stale, and run `quick_validate.py` before finishing.
