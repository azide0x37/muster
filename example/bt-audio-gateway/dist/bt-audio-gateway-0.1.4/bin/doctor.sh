#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_FILE="$ROOT/etc/$PROJECT/$PROJECT.env"
FAILED=0

say() {
  printf '%s\n' "$*"
}

fail() {
  say "FAIL: $*"
  FAILED=1
}

pass() {
  say "PASS: $*"
}

need_command() {
  if command -v "$1" >/dev/null 2>&1; then
    pass "command exists: $1"
  else
    fail "missing command: $1"
  fi
}

check_required_config() {
  if [ ! -f "$CONFIG_FILE" ]; then
    fail "missing config: $CONFIG_FILE"
    return
  fi

  # shellcheck disable=SC1090
  . "$CONFIG_FILE"

  for var in BT_MAC AUDIO_USER SNAPSERVER_HOST SNAPSERVER_PORT SNAPCLIENT_ID; do
    eval "value=\${$var:-}"
    if [ -n "$value" ]; then
      pass "config $var is set"
    else
      fail "config $var is required"
    fi
  done
}

check_systemd_unit() {
  unit="$1"
  if [ -f "$ROOT/etc/systemd/system/$unit" ] || [ -f "/etc/systemd/system/$unit" ]; then
    pass "systemd unit present: $unit"
  else
    fail "systemd unit missing: $unit"
  fi
}

check_runtime() {
  if [ -n "$ROOT" ]; then
    say "INFO: staged root set; skipping live Bluetooth, PipeWire, and network checks"
    return
  fi

  # shellcheck disable=SC1090
  . "$CONFIG_FILE"
  UID_NUM="$(id -u "$AUDIO_USER" 2>/dev/null || true)"
  if [ -z "$UID_NUM" ]; then
    fail "audio user does not exist: $AUDIO_USER"
    return
  fi

  if systemctl is-active --quiet bluetooth.service; then
    pass "bluetooth.service is active"
  else
    fail "bluetooth.service is not active"
  fi

  if bluetoothctl info "$BT_MAC" 2>/dev/null | grep -q "Trusted: yes"; then
    pass "Bluetooth device is trusted"
  else
    fail "Bluetooth device is not trusted or not known: $BT_MAC"
  fi

  if [ -S "/run/user/$UID_NUM/pulse/native" ]; then
    pass "PipeWire/Pulse socket exists"
  else
    fail "PipeWire/Pulse socket missing for $AUDIO_USER"
  fi

  if command -v nc >/dev/null 2>&1; then
    if nc -z "$SNAPSERVER_HOST" "$SNAPSERVER_PORT" >/dev/null 2>&1; then
      pass "Snapserver reachable at $SNAPSERVER_HOST:$SNAPSERVER_PORT"
    else
      fail "Snapserver not reachable at $SNAPSERVER_HOST:$SNAPSERVER_PORT"
    fi
  else
    say "INFO: nc not installed; skipping Snapserver port check"
  fi
}

need_command bluetoothctl
need_command pactl
need_command snapclient
need_command systemctl
check_required_config
check_systemd_unit bt-audio-watch.service
check_systemd_unit snapclient-bt@.service
check_systemd_unit bt-audio-doctor.service
check_systemd_unit bt-audio-doctor.timer
check_systemd_unit bt-audio-update.service
check_systemd_unit bt-audio-update.timer
check_runtime

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
