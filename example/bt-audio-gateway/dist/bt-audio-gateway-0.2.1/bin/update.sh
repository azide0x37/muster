#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_FILE="$ROOT/etc/$PROJECT/$PROJECT.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_DIR=""
TMP_CREATED=0
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TRANSACTION_ACTIVE=0
NEW_VERSION=""
NEW_RELEASE=""

# shellcheck disable=SC1091
. "$SCRIPT_DIR/release-transaction.sh"

log() {
  message="$*"
  if command -v systemd-cat >/dev/null 2>&1 && [ -z "$ROOT" ]; then
    printf '%s\n' "$message" | systemd-cat -t "$PROJECT-update" >/dev/null 2>&1 || \
      printf '%s\n' "$message" || true
  else
    printf '%s\n' "$message" || true
  fi
}

create_private_tmp() {
  old_umask=$(umask)
  umask 077
  TMP_DIR=$(mktemp -d "$TMP_PARENT/$PROJECT-update.XXXXXX") || {
    umask "$old_umask"
    release_die "could not create private update workspace"
  }
  umask "$old_umask"
  TMP_CREATED=1
}

restart_services() {
  if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    systemctl restart bt-audio-watch.service >/dev/null 2>&1 || true
    systemctl restart "snapclient-bt@${AUDIO_USER:-pi}.service" >/dev/null 2>&1 || true
  fi
}

rollback_transaction() {
  [ "$TRANSACTION_ACTIVE" = "1" ] || return 0
  TRANSACTION_ACTIVE=0
  managed_restore systemd "$SYSTEMD_DIR"
  release_restore_state
  restart_services
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  rollback_transaction || true
  if [ -n "$RELEASE_STAGE" ]; then
    rm -rf "$RELEASE_STAGE"
    RELEASE_STAGE=""
  fi
  if [ "$TMP_CREATED" = "1" ] && [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
    TMP_CREATED=0
  fi
  release_unlock
  exit "$status"
}
trap cleanup EXIT INT TERM

project_release_valid() {
  directory="$1"
  version="$2"
  release_dir_valid "$directory" "$version" || return 1
  for required in \
    bin/release-transaction.sh bin/bt-audio-watch bin/bt-audio-route \
    systemd/bt-audio-watch.service systemd/bt-audio-doctor.timer \
    etc/bt-audio-gateway.env.example; do
    [ -f "$directory/$required" ] && [ ! -L "$directory/$required" ] || return 1
  done
  [ -x "$directory/bin/muster-bootstrap.sh" ] && [ -x "$directory/bin/doctor.sh" ] && \
    [ -x "$directory/bin/bt-audio-watch" ] && [ -x "$directory/bin/bt-audio-route" ]
}

validate_registration() {
  if [ -n "$ROOT" ]; then
    "$ROOT/usr/local/bin/muster" --root "$ROOT" validate
  else
    /usr/local/bin/muster validate
  fi
}

run_doctor() {
  doctor_log="$TMP_DIR/doctor.log"
  if "$CURRENT_LINK/bin/doctor.sh" --runtime > "$doctor_log" 2>&1; then
    return 0
  else
    status=$?
  fi
  log "Doctor failed with exit status $status"
  while IFS= read -r line || [ -n "$line" ]; do
    log "doctor: $line"
  done < "$doctor_log"
  return "$status"
}

[ -f "$CONFIG_FILE" ] || release_die "missing config: $CONFIG_FILE"
# shellcheck disable=SC1090
. "$CONFIG_FILE"

if [ "${AUTOUPDATE:-1}" = "0" ]; then
  log "AUTOUPDATE=0; skipping update"
  exit 0
fi

manifest_url="${UPDATE_MANIFEST_URL:-}"
case "$manifest_url" in
  ''|*'<'*|*'>'*) log "UPDATE_MANIFEST_URL is not configured for a real release"; exit 0 ;;
esac

create_private_tmp
release_acquire_lock
[ -L "$CURRENT_LINK" ] || release_die "$PROJECT is not installed; refusing to update an absent current release"
curl -fsSL "$manifest_url" -o "$TMP_DIR/manifest.json" || release_die "could not fetch release manifest"

manifest_project=$(release_json_value project "$TMP_DIR/manifest.json")
NEW_VERSION=$(release_json_value version "$TMP_DIR/manifest.json")
artifact_url=$(release_json_value artifact_url "$TMP_DIR/manifest.json")
artifact_sha=$(release_json_value sha256 "$TMP_DIR/manifest.json")
[ "$manifest_project" = "$PROJECT" ] || release_die "release manifest project $manifest_project does not match $PROJECT"
release_require_version "$NEW_VERSION"
release_require_sha "$artifact_sha"
[ -n "$artifact_url" ] || release_die "release manifest has no artifact_url"

current_target=$(readlink "$CURRENT_LINK" 2>/dev/null || true)
current_version=${current_target#releases/}
if [ "$current_target" = "releases/$NEW_VERSION" ]; then
  project_release_valid "$RELEASES_DIR/$NEW_VERSION" "$NEW_VERSION" || \
    release_die "current release $NEW_VERSION is not valid"
  log "$PROJECT already at $NEW_VERSION"
  exit 0
fi
if [ -n "$current_target" ] && [ "$current_target" = "$current_version" ]; then
  release_die "refusing unmanaged current target $current_target"
fi

curl -fsSL "$artifact_url" -o "$TMP_DIR/artifact.tar.gz" || release_die "could not fetch release artifact"
[ "$(release_sha256 "$TMP_DIR/artifact.tar.gz")" = "$artifact_sha" ] || release_die "downloaded artifact SHA256 mismatch"
release_extract_archive "$TMP_DIR/artifact.tar.gz" "$NEW_VERSION"
project_release_valid "$RELEASE_STAGE" "$NEW_VERSION" || release_die "staged release $NEW_VERSION failed project validation"
release_publish_stage "$NEW_VERSION"
NEW_RELEASE="$RELEASES_DIR/$NEW_VERSION"

"$NEW_RELEASE/bin/muster-bootstrap.sh" ensure
release_snapshot_state
old_release="$TMP_DIR/no-old"
if [ "$HAD_CURRENT" = "1" ]; then
  old_release="$INSTALL_DIR/$PREVIOUS_TARGET"
fi
managed_snapshot systemd "$SYSTEMD_DIR" "$NEW_RELEASE/systemd" "$old_release/systemd"

TRANSACTION_ACTIVE=1
managed_apply systemd "$SYSTEMD_DIR" "$NEW_RELEASE/systemd"
release_switch_current "$NEW_VERSION"
if ! "$NEW_RELEASE/bin/muster-bootstrap.sh" register "$PROJECT"; then
  rollback_transaction
  release_die "Muster registration failed after update; restored the prior transaction"
fi
if ! validate_registration; then
  rollback_transaction
  release_die "Muster graph validation failed after update; restored the prior transaction"
fi
restart_services

if ! run_doctor; then
  log "Health check failed after update; rolling back"
  rollback_transaction
  exit 1
fi

TRANSACTION_ACTIVE=0
log "Updated $PROJECT to $NEW_VERSION"
