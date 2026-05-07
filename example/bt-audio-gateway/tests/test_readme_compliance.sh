#!/bin/sh
set -eu

grep -q "## Muster Self-Certification" README.md

for required in \
  "systemd owns lifecycle" \
  "timer-based update polling exists" \
  "idempotent installer exists" \
  "rollback path exists" \
  "doctor check exists" \
  "release artifacts buildable" \
  "tests runnable"
do
  grep -q "$required" README.md
done
