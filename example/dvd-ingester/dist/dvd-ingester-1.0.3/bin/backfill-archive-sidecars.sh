#!/bin/sh
set -eu

ROOT="${DEST_DIR:-/mnt/media/dvd-ingester}"
APPLY=0
LIMIT=0
SIDECAR_FILE="${ARCHIVE_SIDECAR_FILE:-main-feature.mkv}"
METADATA_FILE="${ARCHIVE_SIDECAR_METADATA_FILE:-main-feature.json}"
TITLE_POLICY="${ARCHIVE_SIDECAR_TITLE:-longest}"
HANDBRAKE_CLI="${HANDBRAKE_CLI:-HandBrakeCLI}"
HANDBRAKE_PRESET="${HANDBRAKE_PRESET:-Fast 480p30}"
PROCESSED=0
FAILED=0
CANDIDATES="${TMPDIR:-/tmp}/dvd-ingester-sidecar-candidates.$$"

cleanup() {
  rm -f "$CANDIDATES"
}
trap cleanup EXIT INT TERM

usage() {
  printf '%s\n' "Usage: backfill-archive-sidecars.sh [ROOT] [--apply] [--limit N]"
  printf '%s\n' "       [--title longest|title1|N] [--preset PRESET]"
}

if [ $# -gt 0 ] && [ "${1#-}" = "$1" ]; then
  ROOT="$1"
  shift
fi

while [ $# -gt 0 ]; do
  case "$1" in
    --apply) APPLY=1 ;;
    --limit)
      shift
      if [ $# -eq 0 ]; then
        printf '%s\n' "failed: --limit requires a value" >&2
        exit 2
      fi
      LIMIT="${1:-}"
      ;;
    --title)
      shift
      if [ $# -eq 0 ]; then
        printf '%s\n' "failed: --title requires a value" >&2
        exit 2
      fi
      TITLE_POLICY="${1:-}"
      ;;
    --preset)
      shift
      if [ $# -eq 0 ]; then
        printf '%s\n' "failed: --preset requires a value" >&2
        exit 2
      fi
      HANDBRAKE_PRESET="${1:-}"
      ;;
    -h|--help) usage; exit 0 ;;
    *) printf '%s\n' "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

case "$SIDECAR_FILE" in
  ""|*/*)
    printf '%s\n' "failed: ARCHIVE_SIDECAR_FILE must be a plain filename" >&2
    exit 2
    ;;
esac

case "$METADATA_FILE" in
  ""|*/*)
    printf '%s\n' "failed: ARCHIVE_SIDECAR_METADATA_FILE must be a plain filename" >&2
    exit 2
    ;;
esac

case "$LIMIT" in
  ''|*[!0-9]*)
    printf '%s\n' "failed: --limit must be a non-negative integer" >&2
    exit 2
    ;;
esac

if [ ! -d "$ROOT" ]; then
  printf '%s\n' "failed: archive root is not a directory: $ROOT" >&2
  exit 1
fi

normalize_title_number() {
  title=$(printf '%s' "$1" | tr -cd '0-9' | sed 's/^0*//')
  if [ -z "$title" ]; then
    title=1
  fi
  printf '%s\n' "$title"
}

sidecar_title() {
  archive="$1"
  case "$TITLE_POLICY" in
    longest|main|main-feature)
      if command -v lsdvd >/dev/null 2>&1; then
        longest=$(lsdvd "$archive" 2>/dev/null | awk -F': ' '/^Longest track:/ {print $2; exit}' | tr -cd '0-9')
        normalize_title_number "$longest"
      else
        normalize_title_number 1
      fi
      ;;
    title1|title-1)
      normalize_title_number 1
      ;;
    [0-9]*)
      normalize_title_number "$TITLE_POLICY"
      ;;
    *)
      printf '%s\n' "failed: invalid title policy: $TITLE_POLICY" >&2
      return 1
      ;;
  esac
}

skip_hidden_top_level() {
  archive="$1"
  rel="${archive#"$ROOT"/}"
  top="${rel%%/*}"
  case "$top" in
    .*) return 0 ;;
    *) return 1 ;;
  esac
}

encode_archive() {
  archive="$1"
  title="$2"
  output="$archive/$SIDECAR_FILE"
  tmp="$archive/.incoming-$SIDECAR_FILE.$$"

  if [ "$APPLY" != "1" ]; then
    printf 'dry-run: title=%s output=%s source=%s\n' "$title" "$output" "$archive"
    return 0
  fi

  if ! command -v "$HANDBRAKE_CLI" >/dev/null 2>&1; then
    printf '%s\n' "failed: $HANDBRAKE_CLI is required for sidecar generation" >&2
    return 1
  fi

  rm -f "$tmp"
  if ! "$HANDBRAKE_CLI" -i "$archive" -t "$title" -o "$tmp" --preset "$HANDBRAKE_PRESET"; then
    rm -f "$tmp"
    printf 'failed: handbrake title=%s source=%s\n' "$title" "$archive" >&2
    return 1
  fi

  mv "$tmp" "$output"
  printf '{"file":"%s","source":"%s","title":%s,"preset":"%s"}\n' \
    "$SIDECAR_FILE" "$archive" "$title" "$HANDBRAKE_PRESET" > "$archive/$METADATA_FILE"
  printf 'created: title=%s output=%s\n' "$title" "$output"
}

find "$ROOT" -mindepth 3 -maxdepth 5 -type f -path '*/VIDEO_TS/VIDEO_TS.IFO' 2>/dev/null | sort > "$CANDIDATES"

while IFS= read -r ifo; do
  archive="${ifo%/VIDEO_TS/VIDEO_TS.IFO}"

  if skip_hidden_top_level "$archive"; then
    continue
  fi

  if [ -f "$archive/$SIDECAR_FILE" ]; then
    continue
  fi

  title="$(sidecar_title "$archive")" || {
    FAILED=$((FAILED + 1))
    continue
  }

  if encode_archive "$archive" "$title"; then
    PROCESSED=$((PROCESSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi

  if [ "$LIMIT" -gt 0 ] && [ "$PROCESSED" -ge "$LIMIT" ]; then
    break
  fi
done < "$CANDIDATES"

printf 'summary: processed=%s failed=%s root=%s apply=%s\n' "$PROCESSED" "$FAILED" "$ROOT" "$APPLY"
if [ "$FAILED" -gt 0 ]; then
  exit 1
fi
