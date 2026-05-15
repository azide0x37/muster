#!/bin/sh
set -eu

for file in bin/*.sh tests/*.sh; do
  sh -n "$file"
done

for file in src/dvd-rip-one src/dvd-publish-one; do
  sh -n "$file"
done
