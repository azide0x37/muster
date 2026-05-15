#!/bin/sh
set -eu

grep -q "## Muster Self-Certification" README.md

for required in \
  "systemd owns lifecycle" \
  "udev only asks systemd" \
  "timer-based update polling exists" \
  "timer-based drain/status work exists" \
  "idempotent installer exists" \
  "rollback path exists" \
  "doctor check exists" \
  "release artifacts buildable" \
  "tests runnable" \
  "MPL pattern mapping documented" \
  "T2R4.device-triggered-conveyor" \
  "T2R6.home-assistant-mqtt-bridge" \
  "R5.capability-mount" \
  "T2C1.hot-cold-nas-conveyor" \
  "Home Assistant bridge exists" \
  "Known Limits"
do
  grep -q "$required" README.md
done
