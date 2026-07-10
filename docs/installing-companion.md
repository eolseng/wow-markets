# Installing WoW Markets Companion

Public releases support Apple Silicon macOS and Windows x64 only. Download
artifacts from the repository's immutable GitHub Releases, verify `SHA256SUMS`,
and optionally verify GitHub provenance with `gh attestation verify`.

## macOS

Open `wow-markets-companion-macos-arm64.dmg`, drag WoW Markets Companion into
Applications, and launch it there. Every public macOS release must be signed
with the WoW Markets Developer ID identity, notarized by Apple, and stapled.
Release automation fails if any signing or notarization credential is absent;
there is no ad-hoc fallback.

## Windows

Run `wow-markets-companion-windows-amd64-setup.exe`. Initial Windows releases
are intentionally not Authenticode-signed. Windows may display an Unknown
Publisher prompt, and Microsoft Defender SmartScreen may require selecting
**More info** and **Run anyway**. This warning does not mean the download is
publisher-verified.

Before continuing, compare the SHA-256 checksum with `SHA256SUMS` and verify
the GitHub provenance attestation. Companion update metadata will separately
authenticate future installers with the embedded update-signing key; that does
not remove the operating-system publisher warning.

The installer places the app under Program Files and registers an uninstaller.
Uninstalling the app intentionally leaves local archives and configuration for
a later reinstall.
