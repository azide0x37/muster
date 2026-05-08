#!/bin/sh
set -eu

test -f muster.yaml

for required in \
  "framework: Muster" \
  "name: dvd-ingester" \
  "manager: systemd" \
  "update_polling: systemd-timer" \
  "required_tool: uv"
do
  grep -q "$required" muster.yaml
done
