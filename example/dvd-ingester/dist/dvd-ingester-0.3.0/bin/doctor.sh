#!/bin/sh
set -eu

PROJECT="dvd-ingester"
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

  for var in BASE_DIR WORK_DIR LOG_DIR RIPPED_DB DEST_DIR RIP_MODE; do
    eval "value=\${$var:-}"
    if [ -n "$value" ]; then
      pass "config $var is set"
    else
      fail "config $var is required"
    fi
  done
}

check_file() {
  path="$1"
  label="$2"
  if [ -e "$path" ]; then
    pass "$label exists: $path"
  else
    fail "$label missing: $path"
  fi
}

check_runtime() {
  if [ -n "$ROOT" ]; then
    say "INFO: staged root set; skipping live optical-drive and NAS checks"
    return
  fi

  # shellcheck disable=SC1090
  . "$CONFIG_FILE"

  if [ -d "$DEST_DIR" ]; then
    pass "destination directory exists: $DEST_DIR"
  else
    fail "destination directory missing: $DEST_DIR"
  fi

  if command -v findmnt >/dev/null 2>&1; then
    if findmnt "$DEST_DIR" >/dev/null 2>&1; then
      pass "destination path is a mountpoint: $DEST_DIR"
    else
      fail "destination path is not a mountpoint: $DEST_DIR"
    fi
  fi
}

need_command flock
need_command jq
need_command eject
need_command rsync
need_command blkid
need_command blockdev
need_command lsdvd
need_command dvdbackup
need_command HandBrakeCLI
need_command systemctl
need_command udevadm
check_required_config
check_file "$ROOT/etc/systemd/system/dvd-rip@.service" "systemd unit"
check_file "$ROOT/etc/systemd/system/dvd-ingester-doctor.service" "systemd unit"
check_file "$ROOT/etc/systemd/system/dvd-ingester-doctor.timer" "systemd timer"
check_file "$ROOT/etc/systemd/system/dvd-ingester-update.service" "systemd unit"
check_file "$ROOT/etc/systemd/system/dvd-ingester-update.timer" "systemd timer"
check_file "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules" "udev rule"
check_runtime

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
