#!/bin/sh
set -eu

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "usage: $0 <companion|addon> <version> [tag]" >&2
  exit 2
fi

product=$1
requested_version=$2
tag=${3:-}
repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

case "$product" in
  companion)
    source_version=$(jq -er '.info.productVersion' "$repository_root/companion/wails.json")
    prefix=companion-v
    ;;
  addon)
    source_version=$(awk -F ': ' '/^## Version: / { print $2; exit }' \
      "$repository_root/addon/WoWMarkets/WoWMarkets.toc")
    prefix=addon-v
    ;;
  *)
    echo "unsupported product: $product" >&2
    exit 2
    ;;
esac

if ! printf '%s\n' "$source_version" | grep -Eq \
  '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'; then
  echo "$product source version is not semantic: $source_version" >&2
  exit 1
fi
if [ "$requested_version" != "$source_version" ]; then
  echo "$product version mismatch: requested $requested_version, source has $source_version" >&2
  exit 1
fi
if [ -n "$tag" ] && [ "$tag" != "$prefix$source_version" ]; then
  echo "$product tag mismatch: got $tag, expected $prefix$source_version" >&2
  exit 1
fi

printf '%s\n' "$source_version"
