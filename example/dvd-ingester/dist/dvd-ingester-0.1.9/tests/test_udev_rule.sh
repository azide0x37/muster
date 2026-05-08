#!/bin/sh
set -eu

RULE="udev/90-dvd-ingester.rules"
test -f "$RULE"
grep -q 'TAG+="systemd"' "$RULE"
grep -q 'ENV{SYSTEMD_READY}!="0"' "$RULE"
grep -q 'ENV{SYSTEMD_WANTS}+="dvd-rip@%k.service"' "$RULE"

if grep -Eq 'HandBrakeCLI|dvdbackup|/opt/dvd-ingester/current/bin/dvd-rip-one' "$RULE"; then
  echo "udev rule must not run long ripping work directly" >&2
  exit 1
fi
