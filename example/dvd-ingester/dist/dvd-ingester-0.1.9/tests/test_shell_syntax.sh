#!/bin/sh
set -eu

for file in bin/*.sh tests/*.sh; do
  sh -n "$file"
done

if command -v bash >/dev/null 2>&1; then
  bash -n src/dvd-rip-one
fi
