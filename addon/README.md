# Addon development

`WowMarketScan` is a companion addon and declares Auctionator as a required
dependency. It targets the TBC Classic interface version `20505`.

For local development, link the addon directory into the relevant game
installation:

```sh
ln -s /absolute/path/to/addon/WowMarketScan \
  "/path/to/World of Warcraft/_anniversary_/Interface/AddOns/WowMarketScan"
```

After an Auctionator full scan:

```text
/wms status
/wms location
```

Queue controls:

```text
/wms clear
```

The addon always exports every row from the Auctionator full-scan payload. It
compacts the payload in 250-row timer batches and reports progress through
`/wms status`. New captures use SavedVariables format 5, which stores the game
region and each
unique item identity once per scan. Earlier development formats are
intentionally unsupported.

At scan completion, the addon records the player's zone, subzone, and Classic
UI map ID. Stranglethorn Vale/Booty Bay, Tanaris/Gadgetzan, and
Winterspring/Everlook are classified as neutral Auction Houses. Other
locations are classified as the player's faction Auction House.
`/wms location` prints the current inputs and classification without scanning.

The export is written to the account-wide SavedVariables file when the client
reloads its UI or exits normally.

Do not edit the SavedVariables file while the game client is running. The
client may overwrite external changes.
