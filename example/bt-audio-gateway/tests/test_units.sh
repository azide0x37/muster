#!/bin/sh
set -eu

for unit in \
  systemd/bt-audio-watch.service \
  systemd/snapclient-bt@.service \
  systemd/bt-audio-doctor.service \
  systemd/bt-audio-doctor.timer \
  systemd/bt-audio-update.service \
  systemd/bt-audio-update.timer
do
  test -f "$unit"
done

if command -v systemd-analyze >/dev/null 2>&1; then
  systemd-analyze verify systemd/*.service systemd/*.timer
else
  echo "systemd-analyze not installed; skipping unit verification"
fi
