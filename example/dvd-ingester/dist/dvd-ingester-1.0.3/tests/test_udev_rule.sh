#!/bin/sh
set -eu

RULE="udev/90-dvd-ingester.rules"

test -f "$RULE"
grep -q 'SUBSYSTEM=="block"' "$RULE"
grep -q 'KERNEL=="sr\[0-9\]\*"' "$RULE"
grep -q 'SYSTEMD_READY' "$RULE"
grep -q 'TAG+="systemd"' "$RULE"
grep -q 'SYSTEMD_WANTS}+="dvd-rip@%k.service"' "$RULE"

if grep -Eq 'RUN[+=]' "$RULE"; then
  echo "udev rule must not run long work directly" >&2
  exit 1
fi

if grep -Eq 'HandBrakeCLI|dvdbackup|makemkvcon|/opt/dvd-ingester/current/bin/dvd-rip-one' "$RULE"; then
  echo "udev rule must only request systemd, not rip directly" >&2
  exit 1
fi
