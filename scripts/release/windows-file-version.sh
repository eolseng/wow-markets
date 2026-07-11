#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <companion-semantic-version>" >&2
  exit 2
fi

requested=$1
version=${requested%%+*}
core=${version%%-*}
prerelease=
if [ "$version" != "$core" ]; then
  prerelease=${version#"$core"-}
fi

if ! printf '%s\n' "$core" | grep -Eq \
  '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  echo "companion version has an invalid numeric core: $requested" >&2
  exit 1
fi

old_ifs=$IFS
IFS=.
set -- $core
IFS=$old_ifs
major=$1
minor=$2
patch=$3

for component in "$major" "$minor" "$patch"; do
  if [ "$component" -gt 65535 ]; then
    echo "Windows version component exceeds 65535: $component" >&2
    exit 1
  fi
done

case "$prerelease" in
  "")
    revision=65535
    ;;
  rc.*)
    revision=${prerelease#rc.}
    if ! printf '%s\n' "$revision" | grep -Eq '^[1-9][0-9]*$' ||
      [ "$revision" -ge 65535 ]; then
      echo "Windows release candidates require rc.1 through rc.65534: $requested" >&2
      exit 1
    fi
    ;;
  *)
    echo "Windows packaging supports stable versions or rc.N prereleases: $requested" >&2
    exit 1
    ;;
esac

printf '%s.%s.%s.%s\n' "$major" "$minor" "$patch" "$revision"
