# WoW Markets addon

WoW Markets piggybacks on Auctionator (a required dependency) so
contributors never scan twice: when Auctionator's full scan completes, the
addon compacts the payload in 250-row timer batches and appends a
SavedVariables format 5 record — the game region and each unique item identity
stored once per scan. It targets the TBC Classic Anniversary client
(interface `20505`).

The handoff to the companion app is the account-wide SavedVariables file,
written when the client reloads its UI or exits normally:

```text
World of Warcraft/_anniversary_/WTF/Account/<ACCOUNT>/SavedVariables/WoWMarkets.lua
```

The record layout is documented in
[contracts/saved-variables/v5/specification.md](../contracts/saved-variables/v5/specification.md).

## Install

Install WoW Markets through
[CurseForge](https://www.curseforge.com/wow/addons/wow-markets) or
[Wago](https://addons.wago.io/addons/wow-markets). The companion detects the
installed version but never installs, updates, or modifies the addon.

## Upgrade from WowMarketScan

Remove the old `Interface/AddOns/WowMarketScan` folder before installing
`WoWMarkets`; leaving both installed loads two capture listeners. Restart WoW,
run one fresh Auctionator full scan, and type `/reload`. WoW Markets Companion
migrates its existing local archives and begins watching `WoWMarkets.lua`.

## Install for development

Link the addon into the game installation:

```sh
ln -s /absolute/path/to/addon/WoWMarkets \
  "/path/to/World of Warcraft/_anniversary_/Interface/AddOns/WoWMarkets"
```

## Slash commands

- `/wm` or `/wm status` — capture progress, stored scan count, and whether the
  latest scan needs `/reload`.
- `/wm location` — current zone, subzone, map ID, and Auction House
  classification without scanning.
- `/wm clear` — request confirmation before emptying the stored scan queue.

The former `/wms` command remains available as a compatibility alias.

## Behavior notes

- The stored queue holds at most 3 scans. Finishing a new capture rotates out
  the oldest; the addon reports the rotation and warns before another scan
  would discard one captured during the current session. Keep the companion
  running and type `/reload` after capturing so WoW writes the latest scan.
- An in-progress capture lives only in memory; only `ready` scans reach
  SavedVariables. A new Auctionator scan is ignored while a capture is active,
  and a capture is dropped entirely if the game region cannot be determined.
- At completion the addon records zone, subzone, and Classic UI map ID.
  Stranglethorn Vale/Booty Bay, Tanaris/Gadgetzan, and Winterspring/Everlook
  classify as neutral Auction Houses; everywhere else is the player's faction
  Auction House.
- Every row of the Auctionator payload is exported; listing owner names are
  not captured. Earlier development formats are intentionally unsupported and
  purged at load.
- Do not edit the SavedVariables file while the game client runs; the client
  overwrites external changes.

## Verify

With Lua 5.1 installed, run the mocked WoW runtime tests from the repository
root:

```sh
make addon-check
```

Release operators should also follow
[`docs/releasing-addon.md`](../docs/releasing-addon.md).
