#!/bin/sh
set -eu

PROJECT="dvd-ingester"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
MOCK_ROOT="${MUSTER_MOCK_ROOT:-${TMPDIR:-/tmp}/$PROJECT-doctor.$$}"
REMOVE_MOCK=0
RIP_SCRIPT="$SRC_ROOT/src/dvd-rip-one"
PUBLISH_SCRIPT="$SRC_ROOT/src/dvd-publish-one"

if [ ! -x "$RIP_SCRIPT" ]; then
  RIP_SCRIPT="$SCRIPT_DIR/dvd-rip-one"
fi

if [ ! -x "$PUBLISH_SCRIPT" ]; then
  PUBLISH_SCRIPT="$SCRIPT_DIR/dvd-publish-one"
fi

if [ -z "${MUSTER_MOCK_ROOT:-}" ]; then
  REMOVE_MOCK=1
fi

cleanup() {
  if [ "$REMOVE_MOCK" = "1" ]; then
    rm -rf "$MOCK_ROOT"
  fi
}
trap cleanup EXIT INT TERM

check_files() {
  test -f "$SRC_ROOT/muster.yaml"
  test -f "$SRC_ROOT/udev/90-dvd-ingester.rules"
  test -f "$SRC_ROOT/systemd/dvd-rip@.service"
  test -f "$SRC_ROOT/systemd/dvd-publish-one.service"
  test -f "$SRC_ROOT/systemd/dvd-publish-one.timer"
  test -x "$RIP_SCRIPT"
  test -x "$PUBLISH_SCRIPT"
  grep -q 'SYSTEMD_WANTS' "$SRC_ROOT/udev/90-dvd-ingester.rules"
}

check_units() {
  if command -v systemd-analyze >/dev/null 2>&1; then
    systemd-analyze verify "$SRC_ROOT"/systemd/*.service "$SRC_ROOT"/systemd/*.timer
  else
    printf '%s\n' "systemd-analyze not installed; skipping unit verification"
  fi
}

check_mock_flow() {
  mkdir -p "$MOCK_ROOT/run/dvd-ingester"
  MUSTER_MOCK_ROOT="$MOCK_ROOT" \
    MUSTER_MOCK_BACKPRESSURE=1 \
    MIN_HOT_FREE_BYTES=1 \
    CAPACITY_TIMEOUT_SECONDS=5 \
    CAPACITY_INTERVAL_SECONDS=1 \
    DRAIN_COMMAND="touch '$MOCK_ROOT/run/dvd-ingester/capacity-ready'" \
    "$RIP_SCRIPT" /dev/sr0 >/dev/null

  test -s "$MOCK_ROOT/run/dvd-ingester/rip.json"
  test -s "$MOCK_ROOT/run/dvd-ingester/handoff.json"
  grep -q 'ready_for_cold_publish' "$MOCK_ROOT/run/dvd-ingester/handoff.json"

  MUSTER_MOCK_ROOT="$MOCK_ROOT" "$PUBLISH_SCRIPT" --once >/dev/null
  test -s "$MOCK_ROOT/run/dvd-ingester/publish.json"
  grep -q '"published":1' "$MOCK_ROOT/run/dvd-ingester/publish.json"
}

check_files
check_units
check_mock_flow

printf '%s\n' "ok: $PROJECT"
