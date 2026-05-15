#!/bin/sh
set -eu

test -f muster.yaml

for required in \
  "framework: Muster" \
  "name: dvd-ingester" \
  "manager: systemd" \
  "update_polling: systemd-timer" \
  "required_tool: uv" \
  "primary: T2R4.device-triggered-conveyor" \
  "verified_head: 309eb158a8f0e3c75187e4ad42a0809780320747" \
  "C6.lifecycle-capsule" \
  "R5.capability-mount" \
  "T2C1.hot-cold-nas-conveyor"
do
  grep -q "$required" muster.yaml
done
