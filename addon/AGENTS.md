# WoW Markets addon

Lua 5.1 addon for the WoW Classic Anniversary client. Run `make addon-check`
from the repository root.

- Keep the release archive limited to the `WoWMarkets/` directory containing
  `WoWMarkets.toc`, `Core.lua`, `Capture.lua`, and `Icon.tga`.
- Auctionator is a dependency; do not copy or redistribute its source.
- Preserve the account-wide `WOW_MARKETS_DB` name and format 5 contract unless
  coordinated fixtures and compatibility support are ready.
- Never collect auction owner names or write outside this addon's own
  SavedVariables table.
