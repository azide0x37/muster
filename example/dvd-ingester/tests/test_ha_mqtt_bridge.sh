#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

mkdir -p "$ROOT/run/dvd-ingester" "$ROOT/var/cache/dvd-ingester/hot" "$ROOT/mnt/dvd-ingester"
printf '{"state":"healthy","reason":"capability_available"}\n' > "$ROOT/run/dvd-ingester/capability.json"
printf '{"state":"healthy","reason":"capacity_available","available_bytes":1,"required_bytes":1}\n' > "$ROOT/run/dvd-ingester/hot-capacity.json"
printf '{"state":"healthy","reason":"ready_for_cold_publish","run_id":"sr0-test"}\n' > "$ROOT/run/dvd-ingester/rip.json"
printf '{"state":"ready_for_cold_publish","run_id":"sr0-test"}\n' > "$ROOT/run/dvd-ingester/handoff.json"
printf '{"state":"healthy","reason":"drain_complete","published":1,"failed":0}\n' > "$ROOT/run/dvd-ingester/publish.json"
mkdir -p "$ROOT/mnt/dvd-ingester/title-1"

MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-ha-mqtt-bridge --once >"$ROOT/bridge.out"
grep -q 'ok: dvd-ingester Home Assistant MQTT bridge updated' "$ROOT/bridge.out"
test -s "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"health":"healthy"' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"extracted_titles":1' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"

DISCOVERY="$ROOT/run/dvd-ingester/ha-mqtt-outbox/homeassistant_device_dvd_ingester_config.json"
test -s "$DISCOVERY"
grep -q '"restart_service"' "$DISCOVERY"
grep -q '"enabled"' "$DISCOVERY"
grep -q 'muster/dvd-ingester/control/restart/set' "$DISCOVERY"
grep -q 'muster/dvd-ingester/control/enabled/set' "$DISCOVERY"

printf '%s\n' "OFF" > "$ROOT/run/dvd-ingester/ha-mqtt-control/enabled.cmd"
MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-ha-mqtt-bridge --control >"$ROOT/control-disable.out"
grep -q '"enabled":"OFF"' "$ROOT/run/dvd-ingester/control.json"
test -f "$ROOT/run/dvd-ingester/disabled"
test -f "$ROOT/run/dvd-ingester/ha-mqtt-control/enabled.cmd.processed"

printf '%s\n' "PRESS" > "$ROOT/run/dvd-ingester/ha-mqtt-control/restart.cmd"
MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-ha-mqtt-bridge --control >"$ROOT/control-restart.out"
grep -q '"action":"restart"' "$ROOT/run/dvd-ingester/control.json"
test -f "$ROOT/run/dvd-ingester/ha-mqtt-control/restart.cmd.processed"

printf '%s\n' "rm -rf /" > "$ROOT/run/dvd-ingester/ha-mqtt-control/enabled.cmd"
if MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-ha-mqtt-bridge --control >"$ROOT/reject.out" 2>"$ROOT/reject.err"; then
  echo "invalid MQTT control payload was accepted" >&2
  exit 1
fi
test -f "$ROOT/run/dvd-ingester/ha-mqtt-control/enabled.cmd.rejected"
