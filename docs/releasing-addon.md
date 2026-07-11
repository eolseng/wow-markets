# Addon release operations

The addon is distributed from the public repository to the same users through
GitHub, CurseForge project `1605493`, and Wago project `qGZOdXNd`. The release
workflow packages one `WoWMarkets/` archive with the pinned BigWigs packager,
uploads it to CurseForge and Wago, validates its contents, and publishes that
same archive in an immutable GitHub release.

## Version and release metadata

- Set the semantic version in `addon/WoWMarkets/WoWMarkets.toc`.
- Add the matching release notes to the top of `addon/CHANGELOG.md`.
- Keep `X-Curse-Project-ID`, `X-Wago-ID`, interface `20506`, and the
  Auctionator dependency aligned with `.pkgmeta`.
- Tag releases as `addon-v<version>`.
- Include `alpha` or `beta` in prerelease versions. BigWigs treats an untagged
  commit as alpha, a tag containing `alpha` as alpha, a tag containing `beta`
  as beta, and every other tag—including `rc`—as stable. Use `beta.N` for an
  addon release candidate rather than `rc.N`.

The `addon-release` GitHub environment holds `CF_API_KEY` and
`WAGO_API_TOKEN`. Pull requests and ordinary branch builds do not receive
these secrets. BigWigs packager `v2.5.1` is pinned by full commit SHA in both
CI and the release workflow.

Create `WAGO_API_TOKEN` under **Wago Addons → Account Settings → API Keys**.
Wago keys intended for addon updaters are a different credential type and
return HTTP 401 when used for publishing. Keep the publishing token only in
the protected GitHub environment unless a separate, reviewed local release
workflow requires it.

## Prepare and publish

1. Run `make addon-check` and `make release-check` from the repository root.
2. Confirm the CI addon job packages a preflight archive and validates that it
   contains exactly `WoWMarkets.toc`, `Core.lua`, and `Capture.lua` beneath one
   `WoWMarkets/` directory.
3. Merge the reviewed version and changelog change to `main`. Release only an
   exact, clean `main` commit.
4. Create and push the tag:

   ```sh
   version=<semantic-version>
   ./scripts/release/validate-addon-distribution.sh "$version" "addon-v$version"
   git tag -s "addon-v$version" -m "WoW Markets addon $version"
   git push origin "addon-v$version"
   ```

5. Approve the protected `addon-release` environment. Require the workflow log
   to show game version `2.5.6`, the intended alpha/beta/release classification,
   and successful CurseForge and Wago uploads.
6. Confirm the GitHub release is published, has the correct prerelease flag,
   and reports `immutable: true`.

The first CurseForge upload sends the new project and file through moderation.
The release is not complete until the file is approved and publicly visible.
Wago publishes the project immediately but may still take time to index a new
version.

## Audit and smoke-test

Download each channel's archive without re-uploading or replacing it. Confirm
all three report the same semantic version, interface `20506`, and package
contents. Record SHA-256 values when comparing downloaded files; metadata or
platform processing may prevent byte-for-byte identity, but extracted addon
files must match the GitHub source release.

Install a downloaded distribution archive into a clean
`_anniversary_/Interface/AddOns` folder, then:

1. Start the game and confirm WoW reports the intended addon version and no
   Lua errors.
2. Run an Auctionator full scan. In one pass, wait for the WoW Markets
   completion message and type `/reload`; in another, exit WoW immediately
   after Auctionator completes to exercise the logout flush.
3. Confirm the companion reports the installed addon version and detects both
   new SavedVariables captures. Inspect the newest persisted scan if needed and
   verify its `addonVersion` matches the release.
4. Confirm the companion archives and uploads the scans and the production web
   experience reflects the accepted data.

Never replace a published archive, tag, or immutable GitHub release. Correct a
bad release by incrementing the semantic version and publishing a new tag. A
bad CurseForge or Wago version may be archived only after the replacement is
available and the incident is recorded.

BigWigs uploads CurseForge before Wago. If a later channel fails after an
earlier one reports success, do not rerun the unchanged job with every
credential present: that may create a duplicate file on the successful
channel. First verify each channel independently, suppress the already
successful channel for the single recovery run, retry only the missing
destinations from the immutable tag, then restore and verify the protected
environment. Revoke and replace any credential that appears in logs or tool
output before retrying.
