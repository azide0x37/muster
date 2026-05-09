#!/bin/sh
set -eu

test -f muster.yaml

for required in \
  "framework: Muster" \
  "name: dvd-ingester" \
  "manager: systemd" \
  "update_polling: systemd-timer" \
  "primary: T2R4.device-triggered-conveyor" \
  "T2C1.hot-cold-nas-conveyor" \
  "R5.capability-mount" \
  "required_tool: uv"
do
  grep -q "$required" muster.yaml
done
