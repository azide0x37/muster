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

if grep -Fq '/run/user/%U' systemd/snapclient-bt@.service; then
  echo "snapclient unit must derive the service user's UID, not the system manager UID" >&2
  exit 1
fi
grep -Fq 'uid=$$(id -u)' systemd/snapclient-bt@.service
grep -Fq 'PULSE_SERVER=unix:/run/user/$$uid/pulse/native' systemd/snapclient-bt@.service

if command -v systemd-analyze >/dev/null 2>&1; then
  systemd-analyze verify systemd/*.service systemd/*.timer
else
  echo "systemd-analyze not installed; skipping unit verification"
fi
