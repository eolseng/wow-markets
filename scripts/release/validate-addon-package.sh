#!/bin/sh
set -eu

archive=${1:?addon archive path is required}
actual=$(mktemp)
expected=$(mktemp)
cleanup() { rm -f "$actual" "$expected"; }
trap cleanup EXIT INT TERM

unzip -Z1 "$archive" | LC_ALL=C sort > "$actual"
printf '%s\n' \
  WoWMarkets/Capture.lua \
  WoWMarkets/Core.lua \
  WoWMarkets/WoWMarkets.toc | LC_ALL=C sort > "$expected"
if ! cmp -s "$actual" "$expected"; then
  echo "addon archive has unexpected contents:" >&2
  cat "$actual" >&2
  exit 1
fi
