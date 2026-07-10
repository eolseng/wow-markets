#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
output=${1:-"$repository_root/dist/wow-markets-addon.zip"}
case "$output" in
  /*) ;;
  *) output="$repository_root/$output" ;;
esac
stage=$(mktemp -d)
cleanup() { rm -rf "$stage"; }
trap cleanup EXIT INT TERM

mkdir -p "$stage/WoWMarkets" "$(dirname -- "$output")"
for file in WoWMarkets.toc Core.lua Capture.lua; do
  cp "$repository_root/addon/WoWMarkets/$file" "$stage/WoWMarkets/$file"
  touch -t 198001010000 "$stage/WoWMarkets/$file"
done
rm -f "$output"
(cd "$stage" && zip -X -q "$output" \
  WoWMarkets/WoWMarkets.toc WoWMarkets/Core.lua WoWMarkets/Capture.lua)
"$repository_root/scripts/release/validate-addon-package.sh" "$output"
