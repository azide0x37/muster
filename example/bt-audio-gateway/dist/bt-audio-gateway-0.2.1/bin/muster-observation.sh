#!/bin/sh
# Shared observation-envelope helpers for Muster doctors.

muster_observation_cleanup() {
  if [ -n "${MUSTER_OBS_TMPDIR:-}" ]; then
    rm -rf "$MUSTER_OBS_TMPDIR"
    MUSTER_OBS_TMPDIR=""
  fi
}

muster_observation_begin() {
  MUSTER_OBS_PROJECT="$1"
  MUSTER_OBS_COMPONENT="$2"
  MUSTER_OBS_OUTPUT="$3"
  MUSTER_OBS_JSON_ONLY="${4:-0}"
  MUSTER_OBS_STARTED=$(date +%s)
  MUSTER_OBS_FAILED=0
  MUSTER_OBS_WARNED=0
  MUSTER_OBS_UNKNOWN=0
  MUSTER_OBS_COUNT=0

  old_umask=$(umask)
  umask 077
  MUSTER_OBS_TMPDIR=$(mktemp -d "${TMPDIR:-/tmp}/muster-observation.XXXXXX") || {
    umask "$old_umask"
    return 1
  }
  umask "$old_umask"
  MUSTER_OBS_CHECKS="$MUSTER_OBS_TMPDIR/checks.json"
  : > "$MUSTER_OBS_CHECKS"
  trap 'muster_observation_cleanup' EXIT
  trap 'muster_observation_cleanup; exit 130' INT
  trap 'muster_observation_cleanup; exit 143' TERM
}

muster_json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n\r' '  '
}

muster_check() {
  status="$1"
  id="$2"
  summary="$3"
  case "$status" in
    healthy|pass)
      health=healthy
      label=PASS
      ;;
    degraded|warn)
      health=degraded
      label=WARN
      MUSTER_OBS_WARNED=1
      ;;
    unhealthy|fail)
      health=unhealthy
      label=FAIL
      MUSTER_OBS_FAILED=1
      ;;
    *)
      health=unknown
      label=INFO
      MUSTER_OBS_UNKNOWN=1
      ;;
  esac
  if [ "$MUSTER_OBS_JSON_ONLY" != "1" ]; then
    printf '%s: %s\n' "$label" "$summary"
  fi
  if [ "$MUSTER_OBS_COUNT" -gt 0 ]; then
    printf ',\n' >> "$MUSTER_OBS_CHECKS"
  fi
  printf '    {"id":"%s","health":"%s","summary":"%s"}' \
    "$(muster_json_escape "$id")" "$health" "$(muster_json_escape "$summary")" >> "$MUSTER_OBS_CHECKS"
  MUSTER_OBS_COUNT=$((MUSTER_OBS_COUNT + 1))
}

muster_observation_emit() {
  ended=$(date +%s)
  duration_ms=$(((ended - MUSTER_OBS_STARTED) * 1000))
  observed_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  if [ "$MUSTER_OBS_FAILED" -ne 0 ]; then
    health=unhealthy
    summary="One or more required checks failed"
  elif [ "$MUSTER_OBS_WARNED" -ne 0 ]; then
    health=degraded
    summary="Required checks passed with warnings"
  elif [ "$MUSTER_OBS_UNKNOWN" -ne 0 ]; then
    health=unknown
    summary="No required checks failed; one or more checks were inconclusive"
  else
    health=healthy
    summary="All required runtime checks passed"
  fi
  payload="$MUSTER_OBS_TMPDIR/payload.json"
  if ! {
    printf '{\n'
    printf '  "schema":"muster.observation/v1",\n'
    printf '  "implementation":"%s",\n' "$(muster_json_escape "$MUSTER_OBS_PROJECT")"
    printf '  "component":"%s",\n' "$(muster_json_escape "$MUSTER_OBS_COMPONENT")"
    printf '  "scope":"runtime",\n'
    printf '  "health":"%s",\n' "$health"
    printf '  "status":"complete",\n'
    printf '  "summary":"%s",\n' "$(muster_json_escape "$summary")"
    printf '  "observed_at":"%s",\n' "$observed_at"
    printf '  "duration_ms":%s,\n' "$duration_ms"
    printf '  "valid_for_seconds":7200,\n'
    printf '  "checks":[\n'
    cat "$MUSTER_OBS_CHECKS"
    printf '\n  ],\n'
    printf '  "artifacts":[]\n'
    printf '}\n'
  } > "$payload"; then
    muster_observation_cleanup
    printf 'muster observation: could not build evidence payload\n' >&2
    return 1
  fi

  output_dir=$(dirname "$MUSTER_OBS_OUTPUT")
  persisted=0
  output_safe=1
  if [ -e "$MUSTER_OBS_OUTPUT" ] || [ -L "$MUSTER_OBS_OUTPUT" ]; then
    [ -f "$MUSTER_OBS_OUTPUT" ] && [ ! -L "$MUSTER_OBS_OUTPUT" ] || output_safe=0
  fi
  if [ "$output_safe" = 1 ] && mkdir -p "$output_dir" 2>/dev/null; then
    old_umask=$(umask)
    umask 077
    output_tmp=$(mktemp "$MUSTER_OBS_OUTPUT.new.XXXXXX" 2>/dev/null || true)
    umask "$old_umask"
    if [ -n "$output_tmp" ]; then
      if cp "$payload" "$output_tmp" && chmod 0644 "$output_tmp" && mv -f "$output_tmp" "$MUSTER_OBS_OUTPUT"; then
        persisted=1
      else
        rm -f "$output_tmp"
      fi
    fi
  fi
  if [ "$MUSTER_OBS_JSON_ONLY" = "1" ]; then
    cat "$payload"
  fi
  muster_observation_cleanup
  if [ "$persisted" -ne 1 ]; then
    printf 'muster observation: could not persist evidence at %s\n' "$MUSTER_OBS_OUTPUT" >&2
    return 1
  fi
  [ "$MUSTER_OBS_FAILED" -eq 0 ]
}
