# WowMarketScan addon

`WowMarketScan` piggybacks on Auctionator (a required dependency) so
contributors never scan twice: when Auctionator's full scan completes, the
addon compacts the payload in 250-row timer batches and appends a
SavedVariables format 5 record — the game region and each unique item identity
stored once per scan. It targets the TBC Classic Anniversary client
(interface `20505`).

The handoff to the companion app is the account-wide SavedVariables file,
written when the client reloads its UI or exits normally:

```text
World of Warcraft/_anniversary_/WTF/Account/<ACCOUNT>/SavedVariables/WowMarketScan.lua
```

The record layout is documented in
[docs/formats/saved-variables.md](../docs/formats/saved-variables.md).

## Install for development

Link the addon into the game installation:

```sh
ln -s /absolute/path/to/addon/WowMarketScan \
  "/path/to/World of Warcraft/_anniversary_/Interface/AddOns/WowMarketScan"
```

## Slash commands

- `/wms status` — capture progress and pending scan count.
- `/wms location` — current zone, subzone, map ID, and Auction House
  classification without scanning.
- `/wms clear` — empty the pending scan queue.

## Behavior notes

- The pending queue holds at most 3 scans; finishing a new capture silently
  evicts the oldest. Upload regularly (keep the companion running) to avoid
  losing scans.
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
