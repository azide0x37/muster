#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

MUSTER_MOCK_ROOT="$ROOT" \
  MUSTER_MOCK_BACKPRESSURE=1 \
  MIN_HOT_FREE_BYTES=1 \
  CAPACITY_TIMEOUT_SECONDS=5 \
  CAPACITY_INTERVAL_SECONDS=1 \
  DRAIN_COMMAND="touch '$ROOT/run/dvd-ingester/capacity-ready'" \
  ./src/dvd-rip-one /dev/sr0 >"$ROOT/rip.out"

grep -q 'ok: dvd-ingester staged' "$ROOT/rip.out"
test -s "$ROOT/run/dvd-ingester/capability.json"
test -s "$ROOT/run/dvd-ingester/hot-capacity.json"
test -s "$ROOT/run/dvd-ingester/rip.json"
test -s "$ROOT/run/dvd-ingester/handoff.json"
grep -q 'ready_for_cold_publish' "$ROOT/run/dvd-ingester/handoff.json"
grep -q 'capacity_available' "$ROOT/run/dvd-ingester/hot-capacity.json"

hot_count=$(find "$ROOT/var/cache/dvd-ingester/hot" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')
test "$hot_count" = "1"

MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-publish-one --once >"$ROOT/publish.out"
grep -q 'ok: dvd-ingester publish drain complete' "$ROOT/publish.out"
test -s "$ROOT/run/dvd-ingester/publish.json"
grep -q '"published":1' "$ROOT/run/dvd-ingester/publish.json"

cold_count=$(find "$ROOT/mnt/dvd-ingester" -mindepth 1 -maxdepth 1 -type d ! -name '.incoming-*' | wc -l | tr -d ' ')
test "$cold_count" = "1"

if find "$ROOT/mnt/dvd-ingester" -mindepth 1 -maxdepth 1 -name '.incoming-*' | grep -q .; then
  echo "publish left destination temp directories" >&2
  exit 1
fi

if find "$ROOT/var/cache/dvd-ingester/hot" -mindepth 1 -maxdepth 1 | grep -q .; then
  echo "hot directory was not drained" >&2
  exit 1
fi

INTERRUPT_ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT" "$INTERRUPT_ROOT"' EXIT INT TERM
INTERRUPT_STATE="$INTERRUPT_ROOT/run/dvd-ingester"
INTERRUPT_WORK="$INTERRUPT_ROOT/work"
INTERRUPT_HOT="$INTERRUPT_ROOT/hot"
INTERRUPT_DEST="$INTERRUPT_ROOT/dest"
mkdir -p "$INTERRUPT_STATE" "$INTERRUPT_WORK" "$INTERRUPT_HOT" "$INTERRUPT_DEST"
  STATE_DIR="$INTERRUPT_STATE" \
  WORK_DIR="$INTERRUPT_WORK" \
  HOT_DIR="$INTERRUPT_HOT" \
  DEST_DIR="$INTERRUPT_DEST" \
  RIP_COMMAND="sleep 2" \
  ./src/dvd-rip-one /dev/sr0 >"$INTERRUPT_ROOT/interrupted.out" 2>"$INTERRUPT_ROOT/interrupted.err" &
pid=$!
sleep 1
kill -TERM "$pid"
wait "$pid" || interrupted_status=$?
test "${interrupted_status:-0}" = "143"
grep -q '"reason":"interrupted"' "$INTERRUPT_STATE/rip.json"
find "$INTERRUPT_WORK" -name .ingest-interrupted | grep -q .
