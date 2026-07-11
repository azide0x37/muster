#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST="$REPO_ROOT/dist"
VERSION=$(cat "$REPO_ROOT/VERSION")

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

count=0
for platform in linux-amd64 linux-arm64 linux-armv7; do
  name="muster-$VERSION-$platform"
  archive="$DIST/$name.tar.gz"
  manifest="$DIST/muster-$platform.json"
  test -s "$archive"
  test -s "$manifest"
  expected=$(cat "$archive.sha256")
  actual=$(sha256_file "$archive")
  test "$actual" = "$expected"
  grep -q '"version":"'"$VERSION"'"' "$manifest"
  grep -q '"platform":"'"$platform"'"' "$manifest"
  grep -q '"sha256":"'"$actual"'"' "$manifest"
  tar -tzf "$archive" | grep -q "^$name/bin/muster$"
  tar -tzf "$archive" | grep -q "^$name/VERSION$"
  tar -tzf "$archive" | grep -q "^$name/docs/OBJECT_MODEL.md$"
  if tar -tzf "$archive" | grep -Eq '(^/|(^|/)\.\.(/|$))'; then
    echo "unsafe member in $archive" >&2
    exit 1
  fi
  count=$((count + 1))
done

test "$count" -eq 3
printf '%s\n' "ok: core release packages"
