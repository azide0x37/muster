#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_FILE="${MUSTER_CONFIG_FILE:-$ROOT/etc/$PROJECT/$PROJECT.env}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
JSON_ONLY=0
CONFIG_READY=0

for argument in "$@"; do
  case "$argument" in
    --runtime) ;;
    --json) JSON_ONLY=1 ;;
    *) echo "usage: doctor.sh [--runtime] [--json]" >&2; exit 2 ;;
  esac
done

# shellcheck disable=SC1091
. "$SCRIPT_DIR/muster-observation.sh"

if [ -n "${MUSTER_DOCTOR_OUTPUT:-}" ]; then
  OBSERVATION_FILE="$MUSTER_DOCTOR_OUTPUT"
elif [ -n "$ROOT" ]; then
  OBSERVATION_FILE="$ROOT/run/muster/$PROJECT/observations/doctor.json"
else
  OBSERVATION_FILE="/run/muster/$PROJECT/observations/doctor.json"
fi

muster_observation_begin "$PROJECT" "doctor" "$OBSERVATION_FILE" "$JSON_ONLY"

need_command() {
  command_name="$1"
  if command -v "$command_name" >/dev/null 2>&1; then
    muster_check healthy "command-$command_name" "command exists: $command_name"
  else
    muster_check unhealthy "command-$command_name" "missing command: $command_name"
  fi
}

check_required_config() {
  if [ ! -f "$CONFIG_FILE" ]; then
    muster_check unhealthy config-file "missing config: $CONFIG_FILE"
    return
  fi
  muster_check healthy config-file "configuration file is present"

  # shellcheck disable=SC1090
  . "$CONFIG_FILE"
  config_missing=0
  for var in BT_MAC AUDIO_USER SNAPSERVER_HOST SNAPSERVER_PORT SNAPCLIENT_ID; do
    eval "value=\${$var:-}"
    if [ -n "$value" ]; then
      muster_check healthy "config-$var" "config $var is set"
    else
      muster_check unhealthy "config-$var" "config $var is required"
      config_missing=1
    fi
  done
  if [ "$config_missing" -eq 0 ]; then
    CONFIG_READY=1
  fi
}

check_systemd_unit() {
  unit="$1"
  if [ -f "$ROOT/etc/systemd/system/$unit" ] || { [ -z "$ROOT" ] && [ -f "/etc/systemd/system/$unit" ]; }; then
    muster_check healthy "unit-$unit" "systemd unit present: $unit"
  else
    muster_check unhealthy "unit-$unit" "systemd unit missing: $unit"
  fi
}

check_runtime() {
  if [ -n "$ROOT" ]; then
    muster_check unknown runtime "staged root: live Bluetooth, audio, and network checks skipped"
    return
  fi
	if [ "$CONFIG_READY" -ne 1 ]; then
		muster_check unknown runtime "live checks skipped because required configuration is incomplete"
		return
	fi
  UID_NUM=$(id -u "$AUDIO_USER" 2>/dev/null || true)
  if [ -z "$UID_NUM" ]; then
    muster_check unhealthy audio-user "audio user does not exist: $AUDIO_USER"
    return
  fi
  muster_check healthy audio-user "audio user exists: $AUDIO_USER"

  if systemctl is-active --quiet bt-audio-watch.service; then
    muster_check healthy watcher-active "bt-audio-watch.service is active"
  else
    muster_check unhealthy watcher-active "bt-audio-watch.service is not active"
  fi

  if systemctl is-active --quiet "snapclient-bt@$AUDIO_USER.service"; then
    muster_check healthy snapclient-active "snapclient instance is active for $AUDIO_USER"
  else
    muster_check unhealthy snapclient-active "snapclient instance is not active for $AUDIO_USER"
  fi

  if systemctl is-active --quiet bluetooth.service; then
    muster_check healthy bluetooth-service "bluetooth.service is active"
    BT_INFO=$(timeout --signal=TERM 5 bluetoothctl info "$BT_MAC" 2>/dev/null || true)
  else
    muster_check unhealthy bluetooth-service "bluetooth.service is not active"
    BT_INFO=""
  fi

  if printf '%s\n' "$BT_INFO" | grep -q "Trusted: yes"; then
    muster_check healthy bluetooth-trusted "Bluetooth device is trusted"
  else
    muster_check unhealthy bluetooth-trusted "Bluetooth device is not trusted or known"
  fi

  if printf '%s\n' "$BT_INFO" | grep -q "Connected: yes"; then
    muster_check healthy bluetooth-connected "Bluetooth device is connected"
  else
    muster_check degraded bluetooth-connected "Bluetooth device is currently disconnected"
  fi

  if [ -S "/run/user/$UID_NUM/pulse/native" ]; then
    muster_check healthy audio-socket "PipeWire/Pulse socket exists"
  else
    muster_check unhealthy audio-socket "PipeWire/Pulse socket is missing"
  fi

  if command -v nc >/dev/null 2>&1; then
    if nc -w 3 -z "$SNAPSERVER_HOST" "$SNAPSERVER_PORT" >/dev/null 2>&1; then
      muster_check healthy snapserver "Snapserver is reachable"
    else
      muster_check unhealthy snapserver "Snapserver is unreachable"
    fi
  else
    muster_check unknown snapserver "nc is unavailable; Snapserver reachability was not tested"
  fi
}

need_command bluetoothctl
need_command pactl
need_command snapclient
need_command systemctl
need_command timeout
check_required_config
for unit in \
  bt-audio-watch.service snapclient-bt@.service \
  bt-audio-doctor.service bt-audio-doctor.timer \
  bt-audio-update.service bt-audio-update.timer; do
  check_systemd_unit "$unit"
done
check_runtime

if muster_observation_emit; then
  exit 0
fi
exit 1
