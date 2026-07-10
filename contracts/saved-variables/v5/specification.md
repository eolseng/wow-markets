# SavedVariables format

The account-wide SavedVariables name is `WOW_MARKETS_DB`.
The only supported per-scan format is version 5. Importers reject every other
format version.

## Envelope

```lua
WOW_MARKETS_DB = {
  schemaVersion = 1,
  config = {
    maxPendingScans = 3,
  },
  pendingScans = {
    {
      formatVersion = 5,
      status = "ready",
      capturedAt = 1781344800,
      exportStartedAt = 1781344800,
      exportFinishedAt = 1781344804,
      exportDurationMs = 3750,
      exportBatchSize = 250,
      region = "eu",
      realm = "Example Realm",
      faction = "Alliance",
      auctionHouse = "faction",
      captureZone = "Stormwind City",
      captureSubzone = "Trade District",
      captureUiMapID = 1453,
      scannerCharacterName = "Examplechar",
      scannerCharacterRealm = "Example Realm",
      scannerCharacterGUID = "Player-0000-00000001",
      scannerRegion = "eu",
      source = "Auctionator",
      sourceEvent = "get_all_scan_complete",
      sourceVersion = "unknown",
      addonVersion = "0.5.0-dev",
      sourceRowCount = 1,
      exportedRowCount = 1,
      itemCount = 1,
      truncated = false,
      itemFields = {
        "itemId",
        "itemString",
        "name",
        "quality",
        "requiredLevel",
      },
      items = {
        {
          21877,
          "item:21877::::::::70:::::::",
          "Netherweave Cloth",
          1,
          0,
        },
      },
      rowFields = {
        "sourceRow",
        "itemRef",
        "stackCount",
        "minBid",
        "minIncrement",
        "buyout",
        "bidAmount",
        "saleStatus",
        "hasAllInfo",
      },
      rows = {
        {
          1,
          1,
          20,
          10000,
          500,
          12000,
          0,
          0,
          1,
        },
      },
    },
  },
}
```

## Row encoding

Items and rows are positional to avoid repeating field names hundreds of
thousands of times. `itemFields` and `rowFields` are included in every scan so
an importer can reject an unexpected layout.

Each unique item identity is stored once in `items`. A listing's `itemRef` is
the one-based index into that table. This avoids repeating long item strings
and names across listings while preserving suffix and item-string variants.

Money values are integer copper amounts. `buyout` is the stack buyout.
Importers calculate unit buyout as ceiling division:

```text
(buyout + stackCount - 1) / stackCount
```

Auction owner names are not captured.

`hasAllInfo` is encoded as `1` or `0` to keep rows dense and avoid ambiguity
from omitted Lua values.

## Auction House identity

`region` identifies the game region containing the scanned realm: `us`, `eu`,
`kr`, `tw`, or `cn`. It is market identity and is separate from character
provenance, even though both values normally match for an in-game scan.

`auctionHouse` is separate from player `faction` and is either `faction` or
`neutral`. The addon freezes the location when Auctionator emits the completed
scan:

- `captureZone`: value returned by `GetZoneText()`.
- `captureSubzone`: value returned by `GetSubZoneText()`, which may be empty.
- `captureUiMapID`: value returned by `C_Map.GetBestMapForUnit("player")`.

The Classic UI map IDs for Stranglethorn Vale, Tanaris, and Winterspring, the
matching zone names, and the town subzones Booty Bay, Gadgetzan, and Everlook
each independently classify the location as neutral. All other locations are
classified as the player's faction Auction House. The companion app's export
validator independently recomputes the classification and rejects scans whose
stored value disagrees.
Ingestion converts these source fields into one analytical market:
`Alliance` or `Horde` for faction Auction Houses and `Neutral` for neutral
Auction Houses.

## Scanner identity

Scanner fields are scan-level provenance:

- `scannerCharacterName`: current player character name.
- `scannerCharacterRealm`: character realm reported by the game client.
- `scannerCharacterGUID`: stable in-game character GUID.
- `scannerRegion`: `us`, `eu`, `kr`, `tw`, or `cn`. The addon aborts the
  capture entirely when the game region cannot be determined, so `unknown`
  never reaches the file.

Name, realm, and region can later support a WoW profile lookup. The GUID is
useful for distinguishing characters but is not a replacement for an
authenticated uploader identity. A future API credential should identify the
person or installation submitting the scan.

## Completeness

`sourceRowCount` is the number of entries emitted by Auctionator.
`exportedRowCount` is the number stored by the addon, which always exports
every row, so `truncated` is always false. The companion app validates that
the flag is consistent with the counts and never queues truncated scans; the
API rejects truncated uploads outright.

## Compatibility policy

Public distribution requires an explicit support window. A breaking successor
format is introduced in this order:

1. Deploy service support for both the current and successor formats.
2. Release a companion that accepts both formats.
3. Release an addon that emits the successor format.
4. Observe adoption and failed uploads.
5. Retire the old format only after a documented support window.

Canonical bytes, checksums, SavedVariables identity, and discovery paths must
not change without coordinated public fixtures and service consumer tests.

WoW Markets Companion accepts the former `WowMarketScan.lua` file containing
`WOW_MARKET_SCAN_DB` as a migration input. New addon captures use only
`WoWMarkets.lua` and `WOW_MARKETS_DB`.
