#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT" "${SIDECAR_ROOT:-}" "${INTERRUPT_ROOT:-}"' EXIT INT TERM

MUSTER_MOCK_ROOT="$ROOT" \
  MUSTER_MOCK_BACKPRESSURE=1 \
  DVD_DISC_LABEL="DVDTitle" \
  DVD_DISC_UUID="12345678" \
  DVD_DISC_TYPE="udf" \
  DVD_DISC_SIZE="2048" \
  DVD_DISC_FINGERPRINT="1234567890abcdefabcd" \
  MIN_HOT_FREE_BYTES=1 \
  CAPACITY_TIMEOUT_SECONDS=5 \
  CAPACITY_INTERVAL_SECONDS=1 \
  DRAIN_COMMAND="touch '$ROOT/run/dvd-ingester/capacity-ready'" \
  RIP_COMMAND='grep -q "\"state\":\"active\"" "$STATE_DIR/rip.json"; grep -q "\"reason\":\"ingest_in_progress\"" "$STATE_DIR/rip.json"; printf "mock ingest from %s\n" "$DEVICE" > "$RUN_DIR/payload.txt"' \
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
test -d "$ROOT/var/cache/dvd-ingester/hot/DVDTitle_1234567890abcdefabcd"
grep -q '"label":"DVDTitle"' "$ROOT/var/cache/dvd-ingester/hot/DVDTitle_1234567890abcdefabcd/metadata.json"

MUSTER_MOCK_ROOT="$ROOT" ./src/dvd-publish-one --once >"$ROOT/publish.out"
grep -q 'ok: dvd-ingester publish drain complete' "$ROOT/publish.out"
test -s "$ROOT/run/dvd-ingester/publish.json"
grep -q '"published":1' "$ROOT/run/dvd-ingester/publish.json"

cold_count=$(find "$ROOT/mnt/dvd-ingester" -mindepth 1 -maxdepth 1 -type d ! -name '.incoming-*' | wc -l | tr -d ' ')
test "$cold_count" = "1"
test -d "$ROOT/mnt/dvd-ingester/DVDTitle_1234567890abcdefabcd"

MUSTER_MOCK_ROOT="$ROOT" \
  DVD_DISC_LABEL="DVDTitle" \
  DVD_DISC_UUID="12345678" \
  DVD_DISC_TYPE="udf" \
  DVD_DISC_SIZE="2048" \
  DVD_DISC_FINGERPRINT="1234567890abcdefabcd" \
  ./src/dvd-rip-one /dev/sr0 >"$ROOT/duplicate.out"
grep -q 'output already exists for DVDTitle_1234567890abcdefabcd' "$ROOT/duplicate.out"
grep -q '"reason":"target_exists"' "$ROOT/run/dvd-ingester/rip.json"

if find "$ROOT/mnt/dvd-ingester" -mindepth 1 -maxdepth 1 -name '.incoming-*' | grep -q .; then
  echo "publish left destination temp directories" >&2
  exit 1
fi

if find "$ROOT/var/cache/dvd-ingester/hot" -mindepth 1 -maxdepth 1 | grep -q .; then
  echo "hot directory was not drained" >&2
  exit 1
fi

SIDECAR_ROOT="$(mktemp -d)"
SIDECAR_BIN="$SIDECAR_ROOT/bin"
mkdir -p "$SIDECAR_BIN"
cat > "$SIDECAR_BIN/dvdbackup" <<'SH'
#!/bin/sh
set -eu
out=
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; out="$1" ;;
  esac
  shift
done
mkdir -p "$out/Archive Disc/VIDEO_TS"
printf 'ifo\n' > "$out/Archive Disc/VIDEO_TS/VIDEO_TS.IFO"
printf 'vob\n' > "$out/Archive Disc/VIDEO_TS/VTS_01_1.VOB"
SH
cat > "$SIDECAR_BIN/lsdvd" <<'SH'
#!/bin/sh
printf '%s\n' 'Title: 01, Length: 00:05:00.000'
printf '%s\n' 'Title: 02, Length: 01:30:00.000'
printf '%s\n' 'Longest track: 02'
SH
cat > "$SIDECAR_BIN/HandBrakeCLI" <<'SH'
#!/bin/sh
set -eu
args="$*"
out=
while [ $# -gt 0 ]; do
  case "$1" in
    -o) shift; out="$1" ;;
  esac
  shift
done
printf '%s\n' "$args" > "$SIDECAR_LOG"
printf 'mkv\n' > "$out"
SH
chmod 0755 "$SIDECAR_BIN/dvdbackup" "$SIDECAR_BIN/lsdvd" "$SIDECAR_BIN/HandBrakeCLI"

PATH="$SIDECAR_BIN:$PATH" \
  MUSTER_MOCK_ROOT="$SIDECAR_ROOT" \
  STATE_DIR="$SIDECAR_ROOT/run/dvd-ingester" \
  WORK_DIR="$SIDECAR_ROOT/work" \
  HOT_DIR="$SIDECAR_ROOT/hot" \
  DEST_DIR="$SIDECAR_ROOT/dest" \
  DVD_DISC_LABEL="ArchiveDisc" \
  DVD_DISC_FINGERPRINT="archive1234567890" \
  MIN_HOT_FREE_BYTES=1 \
  ALLOW_UNMOUNTED_DEST=1 \
  EJECT_AFTER_RIP=0 \
  SIDECAR_LOG="$SIDECAR_ROOT/handbrake.args" \
  ./src/dvd-rip-one /dev/sr0 --apply > "$SIDECAR_ROOT/rip.out"

grep -q 'ok: dvd-ingester staged' "$SIDECAR_ROOT/rip.out"
test -f "$SIDECAR_ROOT/hot/ArchiveDisc_archive1234567890/Archive Disc/VIDEO_TS/VIDEO_TS.IFO"
test -f "$SIDECAR_ROOT/hot/ArchiveDisc_archive1234567890/Archive Disc/main-feature.mkv"
grep -q '"title":2' "$SIDECAR_ROOT/hot/ArchiveDisc_archive1234567890/Archive Disc/main-feature.json"
grep -q -- '-t 2' "$SIDECAR_ROOT/handbrake.args"

INTERRUPT_ROOT="$(mktemp -d)"
INTERRUPT_STATE="$INTERRUPT_ROOT/run/dvd-ingester"
INTERRUPT_WORK="$INTERRUPT_ROOT/work"
INTERRUPT_HOT="$INTERRUPT_ROOT/hot"
INTERRUPT_DEST="$INTERRUPT_ROOT/dest"
mkdir -p "$INTERRUPT_STATE" "$INTERRUPT_WORK" "$INTERRUPT_HOT" "$INTERRUPT_DEST"
STATE_DIR="$INTERRUPT_STATE" \
  WORK_DIR="$INTERRUPT_WORK" \
  HOT_DIR="$INTERRUPT_HOT" \
  DEST_DIR="$INTERRUPT_DEST" \
  DVD_DISC_LABEL="Interrupted DVD" \
  DVD_DISC_FINGERPRINT="interrupted1234567" \
  RIP_COMMAND="sleep 2" \
  ./src/dvd-rip-one /dev/sr0 >"$INTERRUPT_ROOT/interrupted.out" 2>"$INTERRUPT_ROOT/interrupted.err" &
pid=$!
sleep 1
kill -TERM "$pid"
wait "$pid" || interrupted_status=$?
test "${interrupted_status:-0}" = "143"
grep -q '"reason":"interrupted"' "$INTERRUPT_STATE/rip.json"
find "$INTERRUPT_WORK" -name .ingest-interrupted | grep -q .
