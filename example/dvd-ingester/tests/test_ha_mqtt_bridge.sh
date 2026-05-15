#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

mkdir -p "$ROOT/run/dvd-ingester" "$ROOT/var/lib/dvd-ingester/work" "$ROOT/var/cache/dvd-ingester/hot" "$ROOT/mnt/dvd-ingester"
printf '{"state":"healthy","reason":"capability_available"}\n' > "$ROOT/run/dvd-ingester/capability.json"
printf '{"state":"healthy","reason":"capacity_available","available_bytes":1,"required_bytes":1}\n' > "$ROOT/run/dvd-ingester/hot-capacity.json"
printf '{"state":"healthy","reason":"ready_for_cold_publish","run_id":"sr0-test"}\n' > "$ROOT/run/dvd-ingester/rip.json"
printf '{"state":"ready_for_cold_publish","run_id":"sr0-test"}\n' > "$ROOT/run/dvd-ingester/handoff.json"
printf '{"state":"healthy","reason":"drain_complete","published":1,"failed":0}\n' > "$ROOT/run/dvd-ingester/publish.json"
mkdir -p "$ROOT/var/lib/dvd-ingester/work/title-2.work.123" "$ROOT/var/cache/dvd-ingester/hot/title-3" "$ROOT/mnt/dvd-ingester/title-1"

MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-ha-mqtt-bridge --once >"$ROOT/bridge.out"
grep -q 'ok: dvd-ingester Home Assistant MQTT bridge updated' "$ROOT/bridge.out"
test -s "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"health":"healthy"' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"extracted_titles":1' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"work_dir_items":1' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"hot_dir_items":1' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"entries":\["title-2.work.123"\]' "$ROOT/run/dvd-ingester/ha-mqtt-work-folder.json"
grep -q '"entries":\["title-3"\]' "$ROOT/run/dvd-ingester/ha-mqtt-hot-folder.json"
grep -q '"entries":\["title-1"\]' "$ROOT/run/dvd-ingester/ha-mqtt-nas-folder.json"

DISCOVERY="$ROOT/run/dvd-ingester/ha-mqtt-outbox/homeassistant_device_dvd_ingester_config.json"
test -s "$DISCOVERY"
grep -q '"restart_service"' "$DISCOVERY"
grep -q '"enabled"' "$DISCOVERY"
grep -q '"local_free"' "$DISCOVERY"
grep -q '"nas_total"' "$DISCOVERY"
grep -q '"work_items"' "$DISCOVERY"
grep -q '"hot_items"' "$DISCOVERY"
grep -q '"json_attributes_topic":"muster/dvd-ingester/folders/nas/attributes"' "$DISCOVERY"
grep -q '"unit_of_measurement":"GiB"' "$DISCOVERY"
grep -q '"state_class":"measurement"' "$DISCOVERY"
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

cat > "$ROOT/apply.env" <<EOF
STATE_DIR=$ROOT/run/dvd-ingester
WORK_DIR=$ROOT/var/lib/dvd-ingester/work
HOT_DIR=$ROOT/var/cache/dvd-ingester/hot
DEST_DIR=$ROOT/mnt/dvd-ingester
VERSION_FILE=$ROOT/version
EOF
printf '%s\n' "1.0.0" > "$ROOT/version"
cat > "$ROOT/mqtt.env" <<EOF
HA_MQTT_ENABLE=1
MQTT_HOST=127.0.0.1
MQTT_PORT=1883
EOF

if CONFIG_FILE="$ROOT/apply.env" MQTT_CONFIG_FILE="$ROOT/mqtt.env" MOSQUITTO_PUB="$ROOT/missing-mosquitto-pub" ./src/dvd-ha-mqtt-bridge --apply --once >"$ROOT/mqtt-missing.out" 2>"$ROOT/mqtt-missing.err"; then
  echo "missing mosquitto_pub did not fail an enabled MQTT run" >&2
  exit 1
fi
grep -q "mqtt publish failed" "$ROOT/mqtt-missing.err"
test -s "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"local_dir_used_gib":' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
grep -q '"nas_dir_free_gib":' "$ROOT/run/dvd-ingester/ha-mqtt-state.json"
