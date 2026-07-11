#!/bin/sh
set -eu

script=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/windows-file-version.sh
repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
installer="$repository_root/companion/build/windows/installer/project.nsi"

assert_version() {
  expected=$1
  semantic=$2
  actual=$($script "$semantic")
  if [ "$actual" != "$expected" ]; then
    echo "$semantic mapped to $actual; expected $expected" >&2
    exit 1
  fi
}

assert_rejected() {
  if $script "$1" >/dev/null 2>&1; then
    echo "unexpectedly accepted Windows version $1" >&2
    exit 1
  fi
}

assert_version 1.0.0.1 1.0.0-rc.1
assert_version 1.0.0.2 1.0.0-rc.2
assert_version 1.0.0.65535 1.0.0
assert_version 2.3.4.65535 2.3.4+build.7
assert_rejected 1.0.0-beta.1
assert_rejected 1.0.0-rc.0
assert_rejected 1.0.0-rc.65535
assert_rejected 65536.0.0

grep -F 'VIProductVersion "$%WOW_WINDOWS_FILE_VERSION%"' "$installer" >/dev/null
grep -F 'VIFileVersion    "$%WOW_WINDOWS_FILE_VERSION%"' "$installer" >/dev/null
grep -F 'VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"' "$installer" >/dev/null

echo "Windows file-version tests passed"
