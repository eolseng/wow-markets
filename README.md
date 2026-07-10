# WoW Markets clients

Public source for the WoW Markets addon and desktop companion.

- [`addon/`](addon/) captures completed Auctionator full scans in WoW Classic
  Anniversary SavedVariables.
- [`companion/`](companion/) watches those SavedVariables, keeps a local
  canonical archive, and uploads scans authorized by an installation token.

The service API and website are maintained separately and are not part of this
repository. Start with the [addon guide](addon/README.md), the
[companion guide](companion/README.md), or [CONTRIBUTING.md](CONTRIBUTING.md).

Public client contracts and golden fixtures live under [`contracts/`](contracts/).

## Verify

Install Go 1.26.4, Node.js 24, and Lua 5.1, then run:

```sh
make check
```

Development companion builds target loopback services by default. Set
`WOW_MARKETS_API_URL` and `WOW_MARKETS_INSTALLATIONS_URL` explicitly to use a
different local test environment. Never upload fixtures or synthetic scans to
production.

## Privacy and security

Read [PRIVACY.md](PRIVACY.md) for the collected data inventory. Report security
issues through the private route in [SECURITY.md](SECURITY.md), not a public
issue.

Licensed under the [Apache License 2.0](LICENSE).
