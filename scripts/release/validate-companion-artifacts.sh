#!/bin/sh
set -eu

directory=${1:?artifact directory is required}
channel=${2:?stable or beta channel is required}
case "$channel" in
  stable|beta) ;;
  *) echo "invalid update channel: $channel" >&2; exit 2 ;;
esac
for file in \
  "companion-$channel-macos-arm64.xml" \
  "companion-$channel-windows-amd64.xml" \
  wow-markets-companion-macos-arm64.dmg \
  wow-markets-companion-windows-amd64-setup.exe \
  wow-markets-companion-macos-arm64.spdx.json \
  wow-markets-companion-windows-amd64.spdx.json; do
  if [ ! -s "$directory/$file" ]; then
    echo "missing or empty release artifact: $file" >&2
    exit 1
  fi
done
