# Third-party notices

The companion uses third-party Go modules listed in `companion/go.mod` and
`companion/go.sum`. Their license texts remain with their respective source
distributions and module repositories.

Official macOS companion builds redistribute Sparkle 2.9.4, loaded from the
application's embedded `Contents/Frameworks/Sparkle.framework`. Sparkle is
licensed under the MIT License. Its license is retained at
`companion/third_party/sparkle/LICENSE` and copied into the application bundle.

`companion/third_party/systray-on-wails/` is a modified vendored fork of
`github.com/ra1phdd/systray-on-wails` at commit `79e792e24569`. It is licensed
under Apache-2.0; its complete license is retained at
`companion/third_party/systray-on-wails/LICENSE`. That fork derives from
`github.com/getlantern/systray`; source file headers and upstream attribution
are retained.

The addon integrates with, but does not redistribute, Auctionator. World of
Warcraft and related names are trademarks of Blizzard Entertainment. This
project is not affiliated with or endorsed by Blizzard Entertainment.
