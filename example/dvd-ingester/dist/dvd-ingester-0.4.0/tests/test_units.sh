#!/bin/sh
set -eu

for unit in \
  systemd/dvd-rip@.service \
  systemd/dvd-publish-one.service \
  systemd/dvd-publish-one.timer \
  systemd/dvd-ingester-doctor.service \
  systemd/dvd-ingester-doctor.timer \
  systemd/dvd-ingester-update.service \
  systemd/dvd-ingester-update.timer
do
  test -f "$unit"
done

grep -q '/opt/dvd-ingester/current/bin/dvd-rip-one' systemd/dvd-rip@.service
grep -q '/opt/dvd-ingester/current/bin/dvd-publish-one' systemd/dvd-publish-one.service
grep -q 'EnvironmentFile=-/etc/dvd-ingester/dvd-ingester.env' systemd/dvd-rip@.service

if command -v systemd-analyze >/dev/null 2>&1; then
  systemd-analyze verify systemd/*.service systemd/*.timer
else
  echo "systemd-analyze not installed; skipping unit verification"
fi
