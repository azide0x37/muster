#!/bin/sh
set -eu

MODE="${1:-write}"
TMP_FILE="${TMPDIR:-/tmp}/bt-audio-gateway-changelog.$$"
trap 'rm -f "$TMP_FILE"' EXIT

{
  printf '# Changelog\n\n'
  printf 'This file is generated from `RELEASE.md`. Update release notes first, then run `make changelog`.\n\n'
  awk 'BEGIN { copy = 0 } /^## [0-9]/ { copy = 1 } /^## Release Process/ { copy = 0 } copy { print }' RELEASE.md
} > "$TMP_FILE"

if [ "$MODE" = "--check" ]; then
  test -f CHANGELOG.md || { echo "CHANGELOG.md is missing; run make changelog" >&2; exit 1; }
  diff -u CHANGELOG.md "$TMP_FILE"
  exit 0
fi

if [ "$MODE" != "write" ]; then
  echo "usage: $0 [--check]" >&2
  exit 2
fi

mv "$TMP_FILE" CHANGELOG.md
trap - EXIT
