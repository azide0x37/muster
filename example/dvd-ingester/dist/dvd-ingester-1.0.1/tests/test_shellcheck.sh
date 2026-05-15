#!/bin/sh
set -eu

if ! command -v shellcheck >/dev/null 2>&1; then
  echo "shellcheck not installed; skipping"
  exit 0
fi

shellcheck bin/*.sh tests/*.sh src/dvd-rip-one src/dvd-publish-one
