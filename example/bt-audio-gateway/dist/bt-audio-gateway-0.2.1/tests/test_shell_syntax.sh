#!/bin/sh
set -eu

for file in bin/*.sh tests/*.sh; do
  sh -n "$file"
done

for file in src/bt-audio-watch src/bt-audio-route; do
  if command -v bash >/dev/null 2>&1; then
    bash -n "$file"
  fi
done
