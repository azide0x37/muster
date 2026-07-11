#!/bin/sh
set -eu

PROJECT="dvd-ingester"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
ROOT="${MUSTER_ROOT:-}"
MOCK_ROOT="${MUSTER_MOCK_ROOT:-${TMPDIR:-/tmp}/$PROJECT-doctor.$$}"
REMOVE_MOCK=0
MODE=runtime
JSON_ONLY=0

if [ -n "${MUSTER_MOCK_ROOT:-}" ]; then
  MODE=self-test
else
  REMOVE_MOCK=1
fi

for argument in "$@"; do
  case "$argument" in
    --runtime) MODE=runtime ;;
    --self-test) MODE=self-test ;;
    --json) JSON_ONLY=1 ;;
    *) echo "usage: doctor.sh [--runtime|--self-test] [--json]" >&2; exit 2 ;;
  esac
done

# shellcheck disable=SC1091
. "$SCRIPT_DIR/muster-observation.sh"

if [ -n "${MUSTER_DOCTOR_OUTPUT:-}" ]; then
  OBSERVATION_FILE="$MUSTER_DOCTOR_OUTPUT"
elif [ -n "$ROOT" ]; then
  OBSERVATION_FILE="$ROOT/run/muster/$PROJECT/observations/doctor.json"
elif [ -n "${MUSTER_MOCK_ROOT:-}" ]; then
  OBSERVATION_FILE="$MOCK_ROOT/run/muster/$PROJECT/observations/doctor.json"
else
  OBSERVATION_FILE="/run/muster/$PROJECT/observations/doctor.json"
fi

muster_observation_begin "$PROJECT" doctor "$OBSERVATION_FILE" "$JSON_ONLY"

RIP_SCRIPT="$SRC_ROOT/src/dvd-rip-one"
PUBLISH_SCRIPT="$SRC_ROOT/src/dvd-publish-one"
if [ ! -x "$RIP_SCRIPT" ]; then
  RIP_SCRIPT="$SCRIPT_DIR/dvd-rip-one"
fi
if [ ! -x "$PUBLISH_SCRIPT" ]; then
  PUBLISH_SCRIPT="$SCRIPT_DIR/dvd-publish-one"
fi

cleanup() {
	muster_observation_cleanup
  if [ "$REMOVE_MOCK" = "1" ] && [ "$MODE" = "self-test" ]; then
    rm -rf "$MOCK_ROOT"
  fi
}
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

check_path() {
  id="$1"
  path="$2"
  if [ -e "$path" ]; then
    muster_check healthy "$id" "present: $path"
  else
    muster_check unhealthy "$id" "missing: $path"
  fi
}

check_unit_file() {
  unit="$1"
  if [ -f "$ROOT/etc/systemd/system/$unit" ] || [ -f "$SRC_ROOT/systemd/$unit" ] || { [ -z "$ROOT" ] && [ -f "/etc/systemd/system/$unit" ]; }; then
    muster_check healthy "unit-$unit" "systemd unit present: $unit"
  else
    muster_check unhealthy "unit-$unit" "systemd unit missing: $unit"
  fi
}

check_runtime() {
  config="$ROOT/etc/$PROJECT/$PROJECT.env"
  if [ -z "$ROOT" ]; then
    config="/etc/$PROJECT/$PROJECT.env"
  fi
  check_path config-file "$config"
  check_path release-manifest "$SRC_ROOT/muster.yaml"
  check_path rip-command "$RIP_SCRIPT"
  check_path publish-command "$PUBLISH_SCRIPT"

  for unit in \
    dvd-rip@.service dvd-publish-one.service dvd-publish-one.timer \
    dvd-ingester-ha-mqtt.service dvd-ingester-ha-mqtt.timer \
    dvd-ingester-doctor.service dvd-ingester-doctor.timer \
    dvd-ingester-update.service dvd-ingester-update.timer; do
    check_unit_file "$unit"
  done

  if [ -n "$ROOT" ]; then
    muster_check unknown live-systemd "staged root: live unit state was not queried"
  else
    for timer in dvd-publish-one.timer dvd-ingester-ha-mqtt.timer dvd-ingester-doctor.timer dvd-ingester-update.timer; do
      if systemctl is-active --quiet "$timer"; then
        muster_check healthy "active-$timer" "$timer is active"
      else
        muster_check unhealthy "active-$timer" "$timer is not active"
      fi
    done
  fi

  if [ -f "$config" ]; then
    # shellcheck disable=SC1090
    . "$config"
    destination="${DEST_DIR:-}"
    if [ -z "$destination" ]; then
      muster_check unhealthy destination-config "DEST_DIR is not configured"
    elif [ -n "$ROOT" ]; then
      muster_check unknown destination-capability "staged root: destination capability was not probed"
    elif [ -d "$destination" ] && findmnt -T "$destination" >/dev/null 2>&1; then
      muster_check healthy destination-capability "cold destination is mounted"
    else
      muster_check degraded destination-capability "cold destination is not currently mounted"
    fi
  fi

  runtime_dir="$ROOT/run/dvd-ingester"
  if [ -z "$ROOT" ]; then
    runtime_dir=/run/dvd-ingester
  fi
  for state in rip handoff publish ha-mqtt-state; do
    file="$runtime_dir/$state.json"
    if [ ! -e "$file" ]; then
      muster_check healthy "state-$state" "no $state observation has been produced yet"
    elif [ -s "$file" ] && grep -q '^[[:space:]]*{' "$file"; then
      muster_check healthy "state-$state" "$state state is readable JSON"
    else
      muster_check degraded "state-$state" "$state state is empty or malformed"
    fi
  done
}

check_mock_flow() {
  mkdir -p "$MOCK_ROOT/run/dvd-ingester" || return 1
  MUSTER_MOCK_ROOT="$MOCK_ROOT" \
    MUSTER_MOCK_BACKPRESSURE=1 \
    MIN_HOT_FREE_BYTES=1 \
    CAPACITY_TIMEOUT_SECONDS=5 \
    CAPACITY_INTERVAL_SECONDS=1 \
    DRAIN_COMMAND="touch '$MOCK_ROOT/run/dvd-ingester/capacity-ready'" \
    "$RIP_SCRIPT" /dev/sr0 >/dev/null || return 1
  test -s "$MOCK_ROOT/run/dvd-ingester/rip.json" || return 1
  test -s "$MOCK_ROOT/run/dvd-ingester/handoff.json" || return 1
  grep -q ready_for_cold_publish "$MOCK_ROOT/run/dvd-ingester/handoff.json" || return 1

  MUSTER_MOCK_ROOT="$MOCK_ROOT" "$PUBLISH_SCRIPT" --once >/dev/null || return 1
  test -s "$MOCK_ROOT/run/dvd-ingester/publish.json" || return 1
  grep -q '"published":1' "$MOCK_ROOT/run/dvd-ingester/publish.json" || return 1

  bridge_script="$SRC_ROOT/src/dvd-ha-mqtt-bridge"
  control_script="$SRC_ROOT/src/dvd-control"
  if [ ! -x "$bridge_script" ]; then
    bridge_script="$SCRIPT_DIR/dvd-ha-mqtt-bridge"
    control_script="$SCRIPT_DIR/dvd-control"
  fi
  MUSTER_MOCK_ROOT="$MOCK_ROOT" "$bridge_script" --once >/dev/null || return 1
  test -s "$MOCK_ROOT/run/dvd-ingester/ha-mqtt-state.json" || return 1
  grep -q '"restart_service"' "$MOCK_ROOT/run/dvd-ingester/ha-mqtt-outbox/homeassistant_device_dvd_ingester_config.json" || return 1
  printf '%s\n' OFF > "$MOCK_ROOT/run/dvd-ingester/ha-mqtt-control/enabled.cmd"
  MUSTER_MOCK_ROOT="$MOCK_ROOT" CONTROL_COMMAND="$control_script" "$bridge_script" --control >/dev/null || return 1
  grep -q '"enabled":"OFF"' "$MOCK_ROOT/run/dvd-ingester/control.json" || return 1
}

check_self_test() {
  check_path manifest "$SRC_ROOT/muster.yaml"
  check_path udev-rule "$SRC_ROOT/udev/90-dvd-ingester.rules"
  check_path rip-unit "$SRC_ROOT/systemd/dvd-rip@.service"
  check_path publish-timer "$SRC_ROOT/systemd/dvd-publish-one.timer"
  check_path mqtt-timer "$SRC_ROOT/systemd/dvd-ingester-ha-mqtt.timer"
  check_path rip-command "$RIP_SCRIPT"
  check_path publish-command "$PUBLISH_SCRIPT"

  if command -v systemd-analyze >/dev/null 2>&1; then
    if systemd-analyze verify "$SRC_ROOT"/systemd/*.service "$SRC_ROOT"/systemd/*.timer >/dev/null; then
      muster_check healthy unit-verification "systemd-analyze verified all units"
    else
      muster_check unhealthy unit-verification "systemd-analyze rejected one or more units"
    fi
  else
    muster_check unknown unit-verification "systemd-analyze is unavailable"
  fi

  if check_mock_flow; then
    muster_check healthy mock-flow "bounded rip, handoff, publish, MQTT, and control flow passed"
  else
    muster_check unhealthy mock-flow "mock conveyor flow failed"
  fi
}

if [ "$MODE" = self-test ]; then
  check_self_test
else
  check_runtime
fi

if muster_observation_emit; then
  exit 0
fi
exit 1
