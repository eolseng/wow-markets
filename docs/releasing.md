# Release operations

Companion and addon versions are independent semantic versions. The companion
source is `companion/wails.json` at `info.productVersion`; the addon source is
the `Version` field in `addon/WoWMarkets/WoWMarkets.toc`. Tags are
`companion-v<version>` and `addon-v<version>`.

Pull-request CI has read-only permissions and receives no release credentials.
The companion release workflow runs only by manual dispatch from `main`, uses
the approval-protected `release` environment, builds on native runners, and
never publishes a release directly. It can optionally create a draft release
for inspection after producing final-form artifacts, checksums, SPDX SBOMs, and
GitHub provenance/SBOM attestations. It also signs the platform appcasts with
the protected `UPDATE_SIGNING_PRIVATE_KEY_BASE64` secret. Prerelease semantic
versions generate beta feeds; ordinary versions generate stable feeds.

Before a dry run:

1. Confirm all release secrets exist at environment scope, not repository
   scope.
2. Confirm `PRODUCTION_API_URL`, `INSTALLATIONS_URL`, and `UPDATE_ORIGIN`
   environment variables. `UPDATE_ORIGIN` must be exactly
   `https://updates.wowmarkets.app`.
3. Dispatch **Companion Release** with the exact version from `wails.json`.
4. Approve the `release` environment deployment.
5. Inspect signing, notarization, package validation, signed appcasts,
   checksums, SBOMs, and attestations before requesting a draft release.

The release contains `companion-<channel>-macos-arm64.xml` and
`companion-<channel>-windows-amd64.xml`. Promotion copies those exact signed
bytes to `/companion/<channel>/macos-arm64.xml` and
`/companion/<channel>/windows-amd64.xml` on the owned update origin. Never edit
an appcast after signing it.

Publishing remains disabled until the Phase 4 updater is present and proven.
Draft assets must never be described as a public companion release.
