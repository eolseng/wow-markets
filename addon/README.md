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
```

Queue controls:

```text
/wms clear
```

The addon always exports every row from the Auctionator full-scan payload. It
compacts the payload in 250-row timer batches and reports progress through
`/wms status`. New captures use SavedVariables format 3, which stores each
unique item identity once per scan. Earlier development formats are
intentionally unsupported.

The export is written to the account-wide SavedVariables file when the client
reloads its UI or exits normally.

Do not edit the SavedVariables file while the game client is running. The
client may overwrite external changes.
