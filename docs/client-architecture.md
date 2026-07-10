# Client architecture

Auctionator performs the user-requested full scan. The WoW Markets addon
observes its completion event, compacts all listings into format 5, and queues
at most three scans in account-wide SavedVariables. World of Warcraft writes
the file during `/reload`, logout, or normal exit.

The companion discovers Anniversary installations and the account-wide
`WoWMarkets.lua` file. It parses ready scans without modifying the game file,
validates their identity and completeness, serializes canonical JSON, stores a
gzip archive and durable upload record, and uploads sequentially when enabled.
Checksums deduplicate scans across restarts and server duplicates count as
success.

Installation tokens are created on the WoW Markets website. The full token is
stored only in the operating-system credential store; configuration contains a
short hint. Local archive and queue state remain usable when the service is
offline.

The public repository owns client-produced bytes and their specifications. The
private service owns API implementation. Public CI must not require a private
checkout, and the public client must not trust an API-provided updater origin
or signing key.
