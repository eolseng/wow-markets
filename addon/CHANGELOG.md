# WoW Markets addon changelog

## 0.5.0-beta.2

- Support the Burning Crusade Classic Anniversary 2.5.6 client (interface
  `20506`).
- Finish pending scan compaction during logout so closing WoW immediately after
  a full scan still writes a companion-ready record.
- Show the ready message on every login and report listener startup failures.
- Make `/wms` the canonical command because the game client can claim `/wm`;
  add `/wowmarkets` as a long-form alias.

## 0.5.0-beta.1

- Capture Auctionator full-scan results without starting a second scan.
- Export compact, account-wide SavedVariables records in format 5.
- Record market, character, location, source, row, and item metadata needed by
  WoW Markets Companion.
- Retain up to three completed captures until the companion archives them.
- Support the Burning Crusade Classic Anniversary client (interface `20505`).
