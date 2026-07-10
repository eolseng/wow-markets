# WoW Markets clients

Public source for the WoW Markets addon and desktop companion.

- [`addon/`](addon/) captures completed Auctionator full scans in WoW Classic
  Anniversary SavedVariables.
- [`companion/`](companion/) watches those SavedVariables, keeps a local
  canonical archive, and uploads scans authorized by an installation token.

The service API and website are maintained separately and are not part of this
repository. Start with the [addon guide](addon/README.md), the
[companion guide](companion/README.md), or [CONTRIBUTING.md](CONTRIBUTING.md).

## Verify

Install Go 1.26.4, Node.js 24, and Lua 5.1, then run:

```sh
make check
```

The current companion development configuration targets production. Do not use
fixtures or synthetic scans with it. A safe upload-disabled/local development
mode is required before the repository is opened for external contributions.

## Privacy and security

Read [PRIVACY.md](PRIVACY.md) for the collected data inventory. Report security
issues through the private route in [SECURITY.md](SECURITY.md), not a public
issue.

Licensed under the [Apache License 2.0](LICENSE).
