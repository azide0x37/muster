#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_FILE="$ROOT/etc/$PROJECT/$PROJECT.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
TMP_DIR="${TMPDIR:-/tmp}/$PROJECT-update.$$"

log() {
  if command -v systemd-cat >/dev/null 2>&1 && [ -z "$ROOT" ]; then
    printf '%s\n' "$*" | systemd-cat -t "$PROJECT-update"
  else
    printf '%s\n' "$*"
  fi
}

json_value() {
  key="$1"
  sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$2" | head -n 1
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

if [ ! -f "$CONFIG_FILE" ]; then
  log "Missing config: $CONFIG_FILE"
  exit 1
fi

# shellcheck disable=SC1090
. "$CONFIG_FILE"

if [ "${AUTOUPDATE:-1}" = "0" ]; then
  log "AUTOUPDATE=0; skipping update"
  exit 0
fi

MANIFEST_URL="${UPDATE_MANIFEST_URL:-}"
if [ -z "$MANIFEST_URL" ] || printf '%s' "$MANIFEST_URL" | grep -q '[<>]'; then
  log "UPDATE_MANIFEST_URL is not configured for a real release"
  exit 0
fi

mkdir -p "$TMP_DIR" "$RELEASES_DIR"
curl -fsSL "$MANIFEST_URL" -o "$TMP_DIR/manifest.json"

NEW_VERSION="$(json_value version "$TMP_DIR/manifest.json")"
ARTIFACT_URL="$(json_value artifact_url "$TMP_DIR/manifest.json")"
ARTIFACT_SHA="$(json_value sha256 "$TMP_DIR/manifest.json")"

if [ -z "$NEW_VERSION" ] || [ -z "$ARTIFACT_URL" ] || [ -z "$ARTIFACT_SHA" ]; then
  log "Manifest missing version, artifact_url, or sha256"
  exit 1
fi

CURRENT_VERSION="$(basename "$(readlink "$CURRENT_LINK" 2>/dev/null || echo none)")"
if [ "$CURRENT_VERSION" = "$NEW_VERSION" ]; then
  log "$PROJECT already at $NEW_VERSION"
  exit 0
fi

curl -fsSL "$ARTIFACT_URL" -o "$TMP_DIR/artifact.tar.gz"
ACTUAL_SHA="$(sha256_file "$TMP_DIR/artifact.tar.gz")"
if [ "$ACTUAL_SHA" != "$ARTIFACT_SHA" ]; then
  log "SHA256 mismatch for downloaded artifact"
  exit 1
fi

PREVIOUS_TARGET="$(readlink "$CURRENT_LINK" 2>/dev/null || true)"
mkdir -p "$RELEASES_DIR/$NEW_VERSION"
tar -xzf "$TMP_DIR/artifact.tar.gz" -C "$RELEASES_DIR/$NEW_VERSION" --strip-components=1

rm -f "$CURRENT_LINK.next"
ln -s "releases/$NEW_VERSION" "$CURRENT_LINK.next"
rm -f "$CURRENT_LINK"
mv -f "$CURRENT_LINK.next" "$CURRENT_LINK"

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  systemctl restart bt-audio-watch.service || true
  systemctl restart "snapclient-bt@${AUDIO_USER:-pi}.service" || true
fi

if ! "$CURRENT_LINK/bin/doctor.sh"; then
  log "Health check failed after update; rolling back"
  if [ -n "$PREVIOUS_TARGET" ]; then
    rm -f "$CURRENT_LINK.next"
    ln -s "$PREVIOUS_TARGET" "$CURRENT_LINK.next"
    rm -f "$CURRENT_LINK"
    mv -f "$CURRENT_LINK.next" "$CURRENT_LINK"
    if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
      systemctl restart bt-audio-watch.service || true
      systemctl restart "snapclient-bt@${AUDIO_USER:-pi}.service" || true
    fi
  fi
  exit 1
fi

log "Updated $PROJECT to $NEW_VERSION"
