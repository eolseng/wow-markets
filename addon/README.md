# Addon development

`WowMarketScan` is a companion addon and declares Auctionator as a required
dependency. It targets the TBC Classic interface version `20505`.

For local development, link the addon directory into the relevant game
installation:

```sh
ln -s /absolute/path/to/addon/WowMarketScan \
  "/path/to/World of Warcraft/_classic_/Interface/AddOns/WowMarketScan"
```

After an Auctionator full scan:

```text
/wms status
```

Capture controls:

```text
/wms rows 100
/wms rows all
/wms clear
```

`rows all` enables a complete export. The addon compacts the payload in
250-row timer batches and reports progress through `/wms status`. New captures
use a dictionary format that stores each unique item identity once per scan.

The export is written to the account-wide SavedVariables file when the client
reloads its UI or exits normally.

Do not edit the SavedVariables file while the game client is running. The
client may overwrite external changes.
