# Privacy

WoW Markets collects Auction House listing observations so players can inspect
market history. The addon records realm, region, faction, Auction House type,
capture location, character name, character realm, character GUID, addon and
Auctionator versions, item identity and metadata, stack size, bid and buyout
amounts, sale state, and capture timing. Auction owner names are not collected.

The addon keeps up to three scans in account-wide SavedVariables. World of
Warcraft writes that file on `/reload`, logout, or normal exit. The companion
reads but never modifies it. The companion stores selected paths and a token
hint in configuration, keeps canonical scan archives and durable upload state
locally, and stores the full installation token only in macOS Keychain or
Windows Credential Manager.

When upload is enabled, the companion sends the canonical scan archive and its
installation token to the WoW Markets API over HTTPS. The service can associate
an upload with the installation that supplied the token. Development builds
must be upload-disabled or point to a local mock; fixtures must never be sent
to production.

Removing the addon stops new captures. Removing the companion stops uploads;
local application data and credential-store entries may need to be removed
separately using the operating system's tools.
