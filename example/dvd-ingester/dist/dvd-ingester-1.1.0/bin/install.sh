#!/bin/sh
set -eu

PROJECT="dvd-ingester"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
CONFIG_FILE="$CONFIG_DIR/$PROJECT.env"
MQTT_CONFIG_FILE="$CONFIG_DIR/$PROJECT.mqtt.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
UDEV_DIR="$ROOT/etc/udev/rules.d"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DEFAULT_MANIFEST_URL="https://github.com/azide0x37/dvd-ingester/releases/latest/download/manifest.json"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_DIR=""
TMP_CREATED=0
REQUESTED_VERSION="${MUSTER_VERSION:-}"
VERSION=""
RELEASE_DIR=""
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
TRANSACTION_ACTIVE=0

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
  if [ -z "$ROOT" ] && command -v udevadm >/dev/null 2>&1; then
    udevadm control --reload >/dev/null 2>&1 || true
  fi
}

rollback_transaction() {
  [ "$TRANSACTION_ACTIVE" = "1" ] || return 0
  TRANSACTION_ACTIVE=0
  managed_restore systemd "$SYSTEMD_DIR"
  managed_restore udev "$UDEV_DIR"
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
    apt-get install -y curl ca-certificates rsync util-linux eject dvdbackup
  else
    log "apt-get not found; skipping package install"
  fi
}

project_release_valid() {
  directory="$1"
  version="$2"
  release_dir_valid "$directory" "$version" || return 1
  for required in \
    bin/release-transaction.sh bin/dvd-rip-one bin/dvd-publish-one bin/dvd-control bin/dvd-ha-mqtt-bridge \
    systemd/dvd-rip@.service systemd/dvd-publish-one.timer \
    udev/90-dvd-ingester.rules etc/dvd-ingester.env.example; do
    [ -f "$directory/$required" ] && [ ! -L "$directory/$required" ] || return 1
  done
  for executable in muster-bootstrap.sh doctor.sh dvd-rip-one dvd-publish-one dvd-control dvd-ha-mqtt-bridge; do
    [ -x "$directory/bin/$executable" ] || return 1
  done
}

stage_checkout() {
  release_new_stage "$VERSION"
  mkdir -p "$RELEASE_STAGE/bin" "$RELEASE_STAGE/src" "$RELEASE_STAGE/systemd" \
    "$RELEASE_STAGE/udev" "$RELEASE_STAGE/etc" "$RELEASE_STAGE/doc"
  cp "$SRC_ROOT"/bin/*.sh "$RELEASE_STAGE/bin/"
  for program in dvd-rip-one dvd-publish-one dvd-control dvd-ha-mqtt-bridge; do
    cp "$SRC_ROOT/src/$program" "$RELEASE_STAGE/bin/$program"
    cp "$SRC_ROOT/src/$program" "$RELEASE_STAGE/src/$program"
  done
  cp "$SRC_ROOT"/systemd/* "$RELEASE_STAGE/systemd/"
  cp "$SRC_ROOT"/udev/* "$RELEASE_STAGE/udev/"
  cp "$SRC_ROOT"/etc/* "$RELEASE_STAGE/etc/"
  cp "$SRC_ROOT/muster.yaml" "$SRC_ROOT/muster.lock.json" "$SRC_ROOT/VERSION" "$RELEASE_STAGE/"
  for document in README.md MUSTER.md RELEASE.md SECURITY.md CHANGELOG.md; do
    [ -f "$SRC_ROOT/$document" ] && cp "$SRC_ROOT/$document" "$RELEASE_STAGE/doc/"
  done
  chmod 0755 "$RELEASE_STAGE"/bin/*.sh "$RELEASE_STAGE/bin/dvd-rip-one" \
    "$RELEASE_STAGE/bin/dvd-publish-one" "$RELEASE_STAGE/bin/dvd-control" \
    "$RELEASE_STAGE/bin/dvd-ha-mqtt-bridge" "$RELEASE_STAGE/src/dvd-rip-one" \
    "$RELEASE_STAGE/src/dvd-publish-one" "$RELEASE_STAGE/src/dvd-control" \
    "$RELEASE_STAGE/src/dvd-ha-mqtt-bridge"
}

prepare_release() {
  if [ "${MUSTER_STANDALONE:-0}" != "1" ] && [ -f "$SRC_ROOT/muster.yaml" ] && [ -f "$SRC_ROOT/src/dvd-rip-one" ]; then
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
  else
    cp "$RELEASE_DIR/etc/$PROJECT.env.example" "$CONFIG_FILE"
    chmod 0644 "$CONFIG_FILE"
    log "Installed example config at $CONFIG_FILE"
  fi

  if [ -f "$MQTT_CONFIG_FILE" ]; then
    log "Preserving existing $MQTT_CONFIG_FILE"
  else
    temporary="$CONFIG_DIR/.$PROJECT.mqtt.env.$$"
    {
      printf 'HA_MQTT_ENABLE=0\n'
      printf 'MQTT_HOST=127.0.0.1\n'
      printf 'MQTT_PORT=1883\n'
      printf 'MQTT_USERNAME=\n'
      printf 'MQTT_PASSWORD=\n'
      printf 'MQTT_PUBLISH_TIMEOUT_SECONDS=5\n'
      printf 'HA_DISCOVERY_PREFIX=homeassistant\n'
      printf 'HA_TOPIC_PREFIX=muster/dvd-ingester\n'
      printf 'HA_NODE_ID=dvd_ingester\n'
      printf 'HA_DEVICE_NAME=dvd-ingester\n'
    } > "$temporary"
    chmod 0600 "$temporary"
    mv -f "$temporary" "$MQTT_CONFIG_FILE"
    log "Installed MQTT config at $MQTT_CONFIG_FILE"
  fi
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
  managed_snapshot udev "$UDEV_DIR" "$RELEASE_DIR/udev" "$old_release/udev"
  TRANSACTION_ACTIVE=1
  managed_apply systemd "$SYSTEMD_DIR" "$RELEASE_DIR/systemd"
  managed_apply udev "$UDEV_DIR" "$RELEASE_DIR/udev"
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

enable_systemd() {
  [ -z "$ROOT" ] || return 0
  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable --now dvd-publish-one.timer dvd-ingester-doctor.timer dvd-ingester-update.timer dvd-ingester-ha-mqtt.timer
  fi
  if command -v udevadm >/dev/null 2>&1; then
    udevadm control --reload || true
  fi
}

need_root
create_private_tmp
release_acquire_lock
prepare_release
install_packages
install_config
"$RELEASE_DIR/bin/muster-bootstrap.sh" ensure
release_snapshot_state
activate_release
enable_systemd
log "$PROJECT $VERSION installed"
