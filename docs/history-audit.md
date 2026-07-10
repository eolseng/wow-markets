# Public history audit

Audited 2026-07-10 before the client source-of-truth cutover.

## Extraction

A fresh clone of `eolseng/wow-markets-service` was filtered with
`git-filter-repo` 2.47.0 to retain only `addon/` and `companion/`. The service
history contained 194 commits; the filtered client history contained 19
client-relevant commits before its publication commit. The tracked
`addon/.DS_Store` was removed from every rewritten commit. The original public
repository commit is retained as a merge parent.

Author email identities were normalized to the repository owner's GitHub
noreply address. Reachable paths, commit messages, URLs, fixtures, and current
files were reviewed manually. Historical `WowMarketScan` source and fixtures
were retained because they explain and test the supported migration path.

## Secret scanning

The complete rewritten Git history was scanned with:

- Gitleaks 8.30.1: 19 commits, approximately 522 KB, zero findings.
- TruffleHog 3.95.9: 310 chunks, approximately 527 KB, zero verified or
  unverified findings.

The authored public agent guidance was scanned again before its commit and had
zero Gitleaks findings.

After publication files and the CodeQL permission guard were added, the full
reachable history was scanned once more at `00c8705`: Gitleaks scanned 23
commits (approximately 532 KB) with zero findings, and TruffleHog scanned 319
chunks (approximately 537 KB) with zero verified or unverified findings.

## Validation

- `git fsck --full` passed after filtering and before push.
- `make check` passed locally for addon tests, frontend syntax/tests, all Go
  packages, and the companion build.
- GitHub CI passed for addon and companion at public commit
  `0471f388b73133514efbdbd60fe132a32077e2a6`.
- Dependency and vendored-source results are recorded in
  [license-audit.md](license-audit.md).

The audit covers source committed through `00c8705`; later commits are governed
by public pull-request checks and repository protections.
