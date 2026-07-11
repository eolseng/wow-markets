# Release operations

Companion and addon versions are independent semantic versions. This document
is the operator runbook for companion releases; addon releases use their own
workflow and `addon/WoWMarkets/WoWMarkets.toc` version.

## Companion version sources

- Set the semantic version in `companion/wails.json` at
  `info.productVersion`.
- Increment the positive integer in `companion/build-version.txt` for every
  candidate. Never reuse or decrease it, including after a failed or withdrawn
  release.
- Update version assertions such as `companion/config_test.go` in the same
  change.
- Use tag `companion-v<version>`. Prerelease semantic versions produce beta
  appcasts; ordinary versions produce stable appcasts.

Windows fixed-file metadata requires four numeric components. Validation maps
`X.Y.Z-rc.N` to `X.Y.Z.N` and stable `X.Y.Z` to `X.Y.Z.65535`, while preserving
the semantic version in the app, installer, tag, and signed appcast. Other
prerelease labels require a reviewed ordered mapping before use.

## Prepare and build

1. Make the version change on a pull request and run `make check` plus
   `make release-check` from the repository root.
2. Merge only after CI and CodeQL pass. Release exclusively from the protected
   `main` branch; never release an unmerged branch or dirty local tree.
3. Confirm the `release` environment contains the Apple credentials and
   `UPDATE_SIGNING_PRIVATE_KEY_BASE64`, plus these variables:
   `PRODUCTION_API_URL`, `INSTALLATIONS_URL`, and
   `UPDATE_ORIGIN=https://updates.wowmarkets.app`.
4. Dispatch **Companion Release** with the exact semantic version and
   `create_draft_release=true`. With GitHub CLI:

   ```sh
   gh workflow run companion-release.yml --ref main \
     -f version=<version> -f create_draft_release=true
   ```

5. Approve the protected `release` environment for the native macOS/Windows
   jobs. After both succeed, approve it again for assembly and draft creation.
6. Require the macOS log to show a notarization submission ID, an accepted
   result, successful stapling, and Gatekeeper assessment. Do not replace a
   pending CI submission with an unrelated local submission.

Pull-request jobs have read-only permissions and receive no release secrets.
The release workflow builds on native runners, creates signed appcasts,
checksums, SPDX SBOMs, and GitHub provenance/SBOM attestations, then creates a
draft. It never publishes directly.

## Audit the draft

Download the draft assets to a clean directory and verify them before
publication:

```sh
gh release download companion-v<version> --dir <clean-directory>
cd <clean-directory>
shasum -a 256 -c SHA256SUMS
xcrun stapler validate wow-markets-companion-macos-arm64.dmg
spctl --assess --type open --context context:primary-signature --verbose=2 \
  wow-markets-companion-macos-arm64.dmg
```

Also verify:

- Exactly two platform appcasts for the selected channel are present
  (`companion-<channel>-macos-arm64.xml` and
  `companion-<channel>-windows-amd64.xml`). A single release never contains
  both stable and beta appcasts.
- Both appcasts contain the exact semantic version at item level, the exact
  increasing numeric build version on the enclosure, the final immutable tag
  URLs, and Ed25519 signatures.
- The expected DMG, Windows setup executable, selected-channel appcasts, two
  SPDX files, and `SHA256SUMS` are present.
- GitHub attestations resolve to the protected source commit. Use
  `gh attestation verify <asset> --repo eolseng/wow-markets` when performing a
  full independent audit.
- The release is still a draft and has the correct prerelease flag.

Publish only the audited draft:

```sh
# Prerelease:
gh release edit companion-v<version> --draft=false --prerelease
# Stable:
gh release edit companion-v<version> --draft=false
gh api repos/eolseng/wow-markets/releases/tags/companion-v<version> \
  --jq '{draft, prerelease, immutable, tag_name, html_url}'
```

The published release must report `immutable: true`. Never edit a signed
appcast or replace an immutable asset.

## Promote and smoke-test

Publishing does not promote an update. In the private service deployment, set
the exact immutable version and redeploy the owned update origin:

- `COMPANION_BETA_VERSION` for a prerelease version.
- `COMPANION_STABLE_VERSION` for a stable version.

Verify both platform routes under
`https://updates.wowmarkets.app/companion/<channel>/` return the release's exact
signed appcast bytes. Then test from the preceding installed version on real
macOS and Windows accounts:

1. Start the companion and confirm it discovers the update without pressing
   **Check now**.
2. Confirm Home and the tray report the available version.
3. Install from Settings and, when relevant, from the tray.
4. Confirm macOS quits and relaunches through Sparkle. Confirm Windows requests
   elevation, shows no setup wizard, and relaunches as the desktop user.
5. Confirm the updated window opens, the reported version changes, Settings
   scrolls, and token, archive history, upload state, watcher, and start-at-login
   behavior remain intact.

## Rollback and emergency response

- To stop offering a bad release, remove its channel promotion or point the
  channel variable back to the preceding safe immutable release and redeploy.
  This stops new upgrades; it does not downgrade installations already updated.
- Never modify, overwrite, or reuse a published release, tag, appcast, semantic
  version, or native build version. Fix forward with a higher semantic version
  and higher `build-version.txt` value.
- For a security or compatibility emergency, prepare a reviewed workflow
  change that generates `sparkle:criticalUpdate`; the current release workflow
  intentionally exposes no ad hoc critical-release switch. Do not hand-edit a
  signed feed.
- Preserve the affected release, workflow run, notarization ID, checksums, and
  promotion history for incident analysis. Revoke or rotate signing material
  only through the protected environment and document the recovery plan before
  publishing with a new key.
