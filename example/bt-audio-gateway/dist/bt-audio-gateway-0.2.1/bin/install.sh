#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
CONFIG_FILE="$CONFIG_DIR/$PROJECT.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DEFAULT_MANIFEST_URL="https://github.com/azide0x37/bt-audio-gateway/releases/latest/download/manifest.json"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_DIR=""
TMP_CREATED=0
REQUESTED_VERSION="${MUSTER_VERSION:-}"
VERSION=""
RELEASE_DIR=""
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
TRANSACTION_ACTIVE=0
AUDIO_USER_VALUE=""
MIGRATED_AUDIO_USER_FROM=""

# shellcheck disable=SC1091
. "$SCRIPT_DIR/release-transaction.sh"

log() {
  printf '%s\n' "$*" || true
}

create_private_tmp() {
  old_umask=$(umask)
  umask 077
  TMP_DIR=$(mktemp -d "$TMP_PARENT/$PROJECT-install.XXXXXX") || {
    umask "$old_umask"
    release_die "could not create private install workspace"
  }
  umask "$old_umask"
  TMP_CREATED=1
}

reload_managers() {
  if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
  fi
}

rollback_transaction() {
  [ "$TRANSACTION_ACTIVE" = "1" ] || return 0
  TRANSACTION_ACTIVE=0
  managed_restore systemd "$SYSTEMD_DIR"
  release_restore_state
  reload_managers
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

need_root() {
  if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
    release_die "install.sh must run as root; use sudo or set MUSTER_ROOT for a staged install"
  fi
}

install_packages() {
  if [ "${MUSTER_SKIP_PACKAGES:-0}" = "1" ]; then
    log "Skipping package install because MUSTER_SKIP_PACKAGES=1"
    return 0
  fi
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update
    apt-get install -y \
      bluez \
      pipewire pipewire-pulse wireplumber libspa-0.2-bluetooth \
      pulseaudio-utils \
      snapclient \
      curl ca-certificates
  else
    log "apt-get not found; skipping package install"
  fi
}

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

stage_checkout() {
  release_new_stage "$VERSION"
  mkdir -p "$RELEASE_STAGE/bin" "$RELEASE_STAGE/systemd" "$RELEASE_STAGE/etc" "$RELEASE_STAGE/doc"
  cp "$SRC_ROOT"/bin/*.sh "$RELEASE_STAGE/bin/"
  cp "$SRC_ROOT/src/bt-audio-watch" "$SRC_ROOT/src/bt-audio-route" "$RELEASE_STAGE/bin/"
  cp "$SRC_ROOT"/systemd/* "$RELEASE_STAGE/systemd/"
  cp "$SRC_ROOT"/etc/* "$RELEASE_STAGE/etc/"
  cp "$SRC_ROOT/muster.yaml" "$SRC_ROOT/muster.lock.json" "$SRC_ROOT/VERSION" "$RELEASE_STAGE/"
  for document in README.md MUSTER.md RELEASE.md SECURITY.md CHANGELOG.md; do
    [ -f "$SRC_ROOT/$document" ] && cp "$SRC_ROOT/$document" "$RELEASE_STAGE/doc/"
  done
  chmod 0755 "$RELEASE_STAGE"/bin/*.sh "$RELEASE_STAGE/bin/bt-audio-watch" "$RELEASE_STAGE/bin/bt-audio-route"
}

prepare_release() {
  if [ "${MUSTER_STANDALONE:-0}" != "1" ] && [ -f "$SRC_ROOT/muster.yaml" ] && [ -f "$SRC_ROOT/src/bt-audio-watch" ]; then
    embedded_version=$(cat "$SRC_ROOT/VERSION")
    release_require_version "$embedded_version"
    if [ -n "$REQUESTED_VERSION" ] && [ "$REQUESTED_VERSION" != "$embedded_version" ]; then
      release_die "requested version $REQUESTED_VERSION does not match checkout VERSION $embedded_version"
    fi
    VERSION="$embedded_version"
    stage_checkout
  else
    manifest_url="${INSTALL_MANIFEST_URL:-$DEFAULT_MANIFEST_URL}"
    case "$manifest_url" in ''|*'<'*|*'>'*) release_die "INSTALL_MANIFEST_URL is not configured" ;; esac
    curl -fsSL "$manifest_url" -o "$TMP_DIR/manifest.json" || release_die "could not fetch release manifest"
    manifest_project=$(release_json_value project "$TMP_DIR/manifest.json")
    VERSION=$(release_json_value version "$TMP_DIR/manifest.json")
    artifact_url=$(release_json_value artifact_url "$TMP_DIR/manifest.json")
    artifact_sha=$(release_json_value sha256 "$TMP_DIR/manifest.json")
    [ "$manifest_project" = "$PROJECT" ] || release_die "release manifest project $manifest_project does not match $PROJECT"
    release_require_version "$VERSION"
    release_require_sha "$artifact_sha"
    [ -n "$artifact_url" ] || release_die "release manifest has no artifact_url"
    if [ -n "$REQUESTED_VERSION" ] && [ "$REQUESTED_VERSION" != "$VERSION" ]; then
      release_die "requested version $REQUESTED_VERSION does not match manifest version $VERSION"
    fi
    curl -fsSL "$artifact_url" -o "$TMP_DIR/artifact.tar.gz" || release_die "could not fetch release artifact"
    [ "$(release_sha256 "$TMP_DIR/artifact.tar.gz")" = "$artifact_sha" ] || release_die "downloaded artifact SHA256 mismatch"
    release_extract_archive "$TMP_DIR/artifact.tar.gz" "$VERSION"
  fi

  project_release_valid "$RELEASE_STAGE" "$VERSION" || release_die "staged release $VERSION failed project validation"
  release_publish_stage "$VERSION"
  RELEASE_DIR="$RELEASES_DIR/$VERSION"
}

install_config() {
  mkdir -p "$CONFIG_DIR"
  if [ -f "$CONFIG_FILE" ]; then
    log "Preserving existing $CONFIG_FILE"
    return 0
  fi
  cp "$RELEASE_DIR/etc/$PROJECT.env.example" "$CONFIG_FILE"
  chmod 0644 "$CONFIG_FILE"
  log "Installed example config at $CONFIG_FILE"
}

audio_user_exists() {
  candidate="$1"
  [ -n "$candidate" ] && [ "$candidate" != "root" ] && id -u "$candidate" >/dev/null 2>&1
}

replacement_audio_user() {
  if [ -n "${MUSTER_AUDIO_USER:-}" ]; then
    printf '%s\n' "$MUSTER_AUDIO_USER"
    return 0
  fi
  if audio_user_exists "${SUDO_USER:-}"; then
    printf '%s\n' "$SUDO_USER"
    return 0
  fi
  if [ -n "$ROOT" ]; then
    staged_user=$(id -un 2>/dev/null || true)
    if audio_user_exists "$staged_user"; then
      printf '%s\n' "$staged_user"
      return 0
    fi
  fi
  return 1
}

write_audio_user() {
  replacement="$1"
  case "$replacement" in
    ''|*[!A-Za-z0-9_.-]*) release_die "invalid audio user name: $replacement" ;;
  esac
  audio_user_exists "$replacement" || release_die "audio user does not exist or is not allowed: $replacement"
  config_stage=$(mktemp "$CONFIG_DIR/.audio-user.XXXXXX") || release_die "could not stage audio user configuration"
  if ! awk -v user="$replacement" '
    BEGIN { replaced = 0 }
    /^AUDIO_USER=/ { print "AUDIO_USER=" user; replaced = 1; next }
    { print }
    END { if (!replaced) print "AUDIO_USER=" user }
  ' "$CONFIG_FILE" > "$config_stage"; then
    rm -f "$config_stage"
    release_die "could not write audio user configuration"
  fi
  chmod 0644 "$config_stage"
  mv -f "$config_stage" "$CONFIG_FILE"
}

resolve_audio_user() {
  [ -f "$CONFIG_FILE" ] || release_die "missing configuration: $CONFIG_FILE"
  # shellcheck disable=SC1090
  . "$CONFIG_FILE"
  configured="${AUDIO_USER:-}"

  if [ -n "${MUSTER_AUDIO_USER:-}" ]; then
    replacement=$(replacement_audio_user) || release_die "MUSTER_AUDIO_USER must name an existing non-root user"
    audio_user_exists "$replacement" || release_die "MUSTER_AUDIO_USER must name an existing non-root user"
    if [ "$configured" != "$replacement" ]; then
      MIGRATED_AUDIO_USER_FROM="$configured"
      write_audio_user "$replacement"
      log "Configured audio user $replacement from MUSTER_AUDIO_USER"
    fi
    AUDIO_USER_VALUE="$replacement"
    return 0
  fi

  if audio_user_exists "$configured"; then
    AUDIO_USER_VALUE="$configured"
    return 0
  fi

  replacement=$(replacement_audio_user) || release_die "audio user $configured does not exist; rerun with MUSTER_AUDIO_USER=<existing-user>"
  if [ "$configured" != "pi" ]; then
    release_die "audio user $configured does not exist; refusing to rewrite an intentional setting without MUSTER_AUDIO_USER"
  fi
  MIGRATED_AUDIO_USER_FROM="$configured"
  write_audio_user "$replacement"
  AUDIO_USER_VALUE="$replacement"
  log "Migrated legacy audio user $configured to $replacement"
}

validate_registration() {
  if [ -n "$ROOT" ]; then
    "$ROOT/usr/local/bin/muster" --root "$ROOT" validate
  else
    /usr/local/bin/muster validate
  fi
}

activate_release() {
  old_release="$TMP_DIR/no-old"
  if [ "$HAD_CURRENT" = "1" ]; then
    old_release="$INSTALL_DIR/$PREVIOUS_TARGET"
  fi
  managed_snapshot systemd "$SYSTEMD_DIR" "$RELEASE_DIR/systemd" "$old_release/systemd"
  TRANSACTION_ACTIVE=1
  managed_apply systemd "$SYSTEMD_DIR" "$RELEASE_DIR/systemd"
  release_switch_current "$VERSION"
  if ! "$RELEASE_DIR/bin/muster-bootstrap.sh" register "$PROJECT"; then
    rollback_transaction
    release_die "Muster registration failed; restored the prior release transaction"
  fi
  if ! validate_registration; then
    rollback_transaction
    release_die "Muster graph validation failed; restored the prior release transaction"
  fi
  TRANSACTION_ACTIVE=0
}

enable_audio_user() {
  [ -z "$ROOT" ] || return 0
  uid_num=$(id -u "$AUDIO_USER_VALUE")
  loginctl enable-linger "$AUDIO_USER_VALUE" || release_die "could not enable linger for audio user $AUDIO_USER_VALUE"
  if ! sudo -u "$AUDIO_USER_VALUE" XDG_RUNTIME_DIR="/run/user/$uid_num" \
    systemctl --user enable --now pipewire pipewire-pulse wireplumber; then
    release_die "could not enable PipeWire services for audio user $AUDIO_USER_VALUE"
  fi
}

enable_systemd() {
  [ -z "$ROOT" ] || return 0
  command -v systemctl >/dev/null 2>&1 || return 0
  systemctl daemon-reload
  systemctl enable --now bt-audio-watch.service
  systemctl enable --now bt-audio-doctor.timer bt-audio-update.timer
  if [ -n "$MIGRATED_AUDIO_USER_FROM" ] && [ "$MIGRATED_AUDIO_USER_FROM" != "$AUDIO_USER_VALUE" ]; then
    systemctl disable --now "snapclient-bt@$MIGRATED_AUDIO_USER_FROM.service" >/dev/null 2>&1 || true
  fi
  if ! systemctl enable --now "snapclient-bt@$AUDIO_USER_VALUE.service"; then
    release_die "snapclient-bt@$AUDIO_USER_VALUE.service failed to start; inspect it with systemctl status and journalctl"
  fi
}

need_root
create_private_tmp
release_acquire_lock
prepare_release
install_packages
install_config
resolve_audio_user
"$RELEASE_DIR/bin/muster-bootstrap.sh" ensure
release_snapshot_state
activate_release
enable_audio_user
enable_systemd
log "$PROJECT $VERSION installed"
