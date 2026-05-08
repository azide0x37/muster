#!/bin/sh
set -eu

for unit in \
  systemd/dvd-rip@.service \
  systemd/dvd-ingester-doctor.service \
  systemd/dvd-ingester-doctor.timer \
  systemd/dvd-ingester-update.service \
  systemd/dvd-ingester-update.timer
do
  test -f "$unit"
done

if command -v systemd-analyze >/dev/null 2>&1; then
  systemd-analyze verify systemd/*.service systemd/*.timer
else
  echo "systemd-analyze not installed; skipping unit verification"
fi
