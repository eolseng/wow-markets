#!/bin/sh
set -eu

directory=${1:?artifact directory is required}
for file in \
  wow-markets-companion-macos-arm64.dmg \
  wow-markets-companion-windows-amd64-setup.exe \
  wow-markets-companion-macos-arm64.spdx.json \
  wow-markets-companion-windows-amd64.spdx.json; do
  if [ ! -s "$directory/$file" ]; then
    echo "missing or empty release artifact: $file" >&2
    exit 1
  fi
done
