#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
toc="$repository_root/addon/WoWMarkets/WoWMarkets.toc"
requested_version=${1:-$(awk -F ': ' '/^## Version: / { print $2; exit }' "$toc")}
tag=${2:-}

if [ -n "$tag" ]; then
  "$repository_root/scripts/release/validate-version.sh" addon "$requested_version" "$tag" >/dev/null
else
  "$repository_root/scripts/release/validate-version.sh" addon "$requested_version" >/dev/null
fi

metadata() {
  key=$1
  awk -F ': ' -v key="## $key" '$1 == key { print $2; exit }' "$toc"
}

interface=$(metadata Interface)
icon_texture=$(metadata IconTexture)
curse_project_id=$(metadata X-Curse-Project-ID)
wago_project_id=$(metadata X-Wago-ID)

if [ "$interface" != 20506 ]; then
  echo "addon interface must classify as Burning Crusade Classic 2.5.6: $interface" >&2
  exit 1
fi
if [ "$icon_texture" != 'Interface\AddOns\WoWMarkets\Icon' ]; then
  echo "addon IconTexture must reference the packaged icon: $icon_texture" >&2
  exit 1
fi
if [ ! -f "$repository_root/addon/WoWMarkets/Icon.tga" ]; then
  echo "addon Icon.tga is missing" >&2
  exit 1
fi
case "$curse_project_id" in
  ''|*[!0-9]*)
    echo "addon X-Curse-Project-ID must be numeric: $curse_project_id" >&2
    exit 1
    ;;
esac
if ! printf '%s\n' "$wago_project_id" | grep -Eq '^[0-9A-Za-z]{8}$'; then
  echo "addon X-Wago-ID must be an eight-character project ID: $wago_project_id" >&2
  exit 1
fi

if [ -n "$tag" ]; then
  case "$requested_version" in
    *-*)
      case "$requested_version" in
        *-alpha*|*-beta*) ;;
        *)
          echo "BigWigs classifies prereleases only when the version contains alpha or beta: $requested_version" >&2
          exit 1
          ;;
      esac
      ;;
  esac
fi

grep -Fx 'package-as: WoWMarkets' "$repository_root/.pkgmeta" >/dev/null
grep -Fx '  WoWMarkets/addon/WoWMarkets: WoWMarkets' "$repository_root/.pkgmeta" >/dev/null

printf 'addon distribution metadata valid: %s (interface %s, CurseForge %s, Wago %s)\n' \
  "$requested_version" "$interface" "$curse_project_id" "$wago_project_id"
