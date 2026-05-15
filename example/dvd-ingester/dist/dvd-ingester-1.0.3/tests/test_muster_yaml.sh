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
  "T2R6.home-assistant-mqtt-bridge" \
  "verified_head: 279573b65c7c72e6b3d4fb96e9d69edfc7f86aaf" \
  "C6.lifecycle-capsule" \
  "R5.capability-mount" \
  "T2C1.hot-cold-nas-conveyor"
do
  grep -q "$required" muster.yaml
done
