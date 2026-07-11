#!/bin/sh
set -eu

for file in bin/*.sh tests/*.sh; do
  sh -n "$file"
done

grep -Fq 'timeout --signal=TERM 5 bluetoothctl' src/bt-audio-watch
grep -Fq 'systemctl is-active --quiet bluetooth.service' src/bt-audio-watch
grep -Fq 'timeout --signal=TERM 5 bluetoothctl info' bin/doctor.sh
grep -Fq 'nc -w 3 -z' bin/doctor.sh

for file in src/bt-audio-watch src/bt-audio-route; do
  if command -v bash >/dev/null 2>&1; then
    bash -n "$file"
  fi
done
