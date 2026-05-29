#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

BIN="$ROOT/bin"
DVD_ROOT="$ROOT/DVD_Rips"
mkdir -p "$BIN" "$DVD_ROOT/MOVIE_123/Movie/VIDEO_TS" "$DVD_ROOT/.incoming-bad/Bad/VIDEO_TS"
printf 'ifo\n' > "$DVD_ROOT/MOVIE_123/Movie/VIDEO_TS/VIDEO_TS.IFO"
printf 'ifo\n' > "$DVD_ROOT/.incoming-bad/Bad/VIDEO_TS/VIDEO_TS.IFO"

cat > "$BIN/lsdvd" <<'SH'
#!/bin/sh
printf '%s\n' 'Title: 01, Length: 00:10:00.000'
printf '%s\n' 'Title: 02, Length: 01:20:00.000'
printf '%s\n' 'Longest track: 02'
SH

cat > "$BIN/HandBrakeCLI" <<'SH'
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
printf '%s\n' "$args" >> "$HANDBRAKE_LOG"
printf 'mkv\n' > "$out"
SH

chmod 0755 "$BIN/lsdvd" "$BIN/HandBrakeCLI"

PATH="$BIN:$PATH" ./bin/backfill-archive-sidecars.sh "$DVD_ROOT" --limit 10 > "$ROOT/dry-run.out"
grep -q 'dry-run: title=2' "$ROOT/dry-run.out"
grep -q 'Movie/main-feature.mkv' "$ROOT/dry-run.out"
if grep -q '.incoming-bad' "$ROOT/dry-run.out"; then
  echo "dry run included hidden publish directory" >&2
  exit 1
fi

PATH="$BIN:$PATH" \
  HANDBRAKE_LOG="$ROOT/handbrake.log" \
  ./bin/backfill-archive-sidecars.sh "$DVD_ROOT" --apply --limit 10 > "$ROOT/apply.out"

grep -q 'created: title=2' "$ROOT/apply.out"
test -f "$DVD_ROOT/MOVIE_123/Movie/main-feature.mkv"
grep -q '"title":2' "$DVD_ROOT/MOVIE_123/Movie/main-feature.json"
grep -q -- '-t 2' "$ROOT/handbrake.log"
test ! -f "$DVD_ROOT/.incoming-bad/Bad/main-feature.mkv"

PATH="$BIN:$PATH" ./bin/backfill-archive-sidecars.sh "$DVD_ROOT" --apply --limit 10 > "$ROOT/resume.out"
grep -q 'summary: processed=0 failed=0' "$ROOT/resume.out"
