# License audit

Audited 2026-07-10 against `companion/go.mod` and `companion/go.sum`.

- Every resolved Go module exposes a license or notice file in its module root.
- The packages linked into the macOS build were classified by `go-licenses` as
  MIT or BSD-2-Clause, apart from this project's own Apache-2.0 packages.
- The vendored `github.com/ra1phdd/systray-on-wails` fork is Apache-2.0. Its
  license is retained beside the source.
- The customized NSIS project is derived from the Wails 2.13.0 installer
  template under the MIT License. The retained license is stored under
  `companion/third_party/wails/`.
- Auctionator is a runtime dependency of the addon and is not redistributed.
- Official macOS builds redistribute the pinned Sparkle 2.9.4 binary
  distribution under its MIT License. The release workflow verifies archive
  SHA-256
  `ce89daf967db1e1893ed3ebd67575ed82d3902563e3191ca92aaec9164fbdef9`
  before embedding it and copies the retained license into the app bundle.

## Vendored systray provenance

The repository URL is no longer publicly reachable, but the Go module proxy
retains pseudo-version
`v0.0.0-20241115230547-79e792e24569`, originating at full commit
`79e792e245699d28575194a72a83875fcc3e6a3a`. The proxy reports module checksum
`h1:gV+LVz4QUfZN4l+QyyNOUBuHh4LgNq3WGVbMVVw8SgY=`.

The vendored directory intentionally omits upstream examples, screenshots,
tests, changelog, Makefile, and README. Its `systray_darwin.m` has local ABI
compatibility changes to function names and the menu-item signature; all other
retained files matched the proxy copy during the audit. These modifications
remain covered by Apache-2.0 and are disclosed in
`THIRD_PARTY_NOTICES.md`.
