#!/bin/sh
set -eu

PROJECT="dvd-ingester"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
CONFIG_FILE="$CONFIG_DIR/$PROJECT.env"
MQTT_CONFIG_FILE="$CONFIG_DIR/$PROJECT.mqtt.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
UDEV_DIR="$ROOT/etc/udev/rules.d"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DEFAULT_MANIFEST_URL="https://github.com/azide0x37/dvd-ingester/releases/latest/download/manifest.json"
TMP_DIR="${TMPDIR:-/tmp}/$PROJECT-install.$$"
VERSION="${MUSTER_VERSION:-}"
RELEASE_DIR=""

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

log() {
  printf '%s\n' "$*"
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

need_root() {
  if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
    echo "install.sh must run as root. Use sudo, or set MUSTER_ROOT for a staged install." >&2
    exit 1
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
      curl ca-certificates \
      rsync util-linux eject dvdbackup lsdvd handbrake-cli
  else
    log "apt-get not found; skipping package install"
  fi
}

prepare_source() {
  if [ -f "$SRC_ROOT/muster.yaml" ] && [ -f "$SRC_ROOT/src/dvd-rip-one" ]; then
    VERSION="${VERSION:-$(cat "$SRC_ROOT/VERSION" 2>/dev/null || echo 0.4.0)}"
    RELEASE_DIR="$INSTALL_DIR/releases/$VERSION"
    return 0
  fi

  MANIFEST_URL="${INSTALL_MANIFEST_URL:-$DEFAULT_MANIFEST_URL}"
  mkdir -p "$TMP_DIR/src"
  curl -fsSL "$MANIFEST_URL" -o "$TMP_DIR/manifest.json"
  VERSION="${VERSION:-$(json_value version "$TMP_DIR/manifest.json")}"
  ARTIFACT_URL="$(json_value artifact_url "$TMP_DIR/manifest.json")"
  ARTIFACT_SHA="$(json_value sha256 "$TMP_DIR/manifest.json")"

  if [ -z "$VERSION" ] || [ -z "$ARTIFACT_URL" ] || [ -z "$ARTIFACT_SHA" ]; then
    echo "Release manifest is missing version, artifact_url, or sha256." >&2
    exit 1
  fi

  curl -fsSL "$ARTIFACT_URL" -o "$TMP_DIR/artifact.tar.gz"
  ACTUAL_SHA="$(sha256_file "$TMP_DIR/artifact.tar.gz")"
  if [ "$ACTUAL_SHA" != "$ARTIFACT_SHA" ]; then
    echo "Downloaded artifact SHA256 mismatch." >&2
    exit 1
  fi

  tar -xzf "$TMP_DIR/artifact.tar.gz" -C "$TMP_DIR/src" --strip-components=1
  SRC_ROOT="$TMP_DIR/src"
  RELEASE_DIR="$INSTALL_DIR/releases/$VERSION"
}

copy_release() {
  mkdir -p "$RELEASE_DIR/bin" "$RELEASE_DIR/src" "$RELEASE_DIR/systemd" "$RELEASE_DIR/udev" "$RELEASE_DIR/etc" "$RELEASE_DIR/doc"

  cp "$SRC_ROOT"/bin/*.sh "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-rip-one "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-publish-one "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-control "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-ha-mqtt-bridge "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-rip-one "$RELEASE_DIR/src/"
  cp "$SRC_ROOT"/src/dvd-publish-one "$RELEASE_DIR/src/"
  cp "$SRC_ROOT"/src/dvd-control "$RELEASE_DIR/src/"
  cp "$SRC_ROOT"/src/dvd-ha-mqtt-bridge "$RELEASE_DIR/src/"
  cp "$SRC_ROOT"/systemd/* "$RELEASE_DIR/systemd/"
  cp "$SRC_ROOT"/udev/* "$RELEASE_DIR/udev/"
  cp "$SRC_ROOT"/etc/* "$RELEASE_DIR/etc/"
  cp "$SRC_ROOT"/muster.yaml "$SRC_ROOT"/VERSION "$RELEASE_DIR/"
  cp "$SRC_ROOT"/README.md "$SRC_ROOT"/MUSTER.md "$SRC_ROOT"/RELEASE.md "$SRC_ROOT"/SECURITY.md "$RELEASE_DIR/doc/"
  chmod 0755 "$RELEASE_DIR/bin"/*.sh "$RELEASE_DIR/bin/dvd-rip-one" "$RELEASE_DIR/bin/dvd-publish-one" "$RELEASE_DIR/bin/dvd-control" "$RELEASE_DIR/bin/dvd-ha-mqtt-bridge" "$RELEASE_DIR/src/dvd-rip-one" "$RELEASE_DIR/src/dvd-publish-one" "$RELEASE_DIR/src/dvd-control" "$RELEASE_DIR/src/dvd-ha-mqtt-bridge"
}

install_config() {
  mkdir -p "$CONFIG_DIR"
  if [ -f "$CONFIG_FILE" ]; then
    log "Preserving existing $CONFIG_FILE"
  else
    cp "$SRC_ROOT/etc/$PROJECT.env.example" "$CONFIG_FILE"
    chmod 0644 "$CONFIG_FILE"
    log "Installed example config at $CONFIG_FILE"
  fi

  if [ -f "$MQTT_CONFIG_FILE" ]; then
    log "Preserving existing $MQTT_CONFIG_FILE"
  else
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
    } > "$MQTT_CONFIG_FILE"
    chmod 0600 "$MQTT_CONFIG_FILE"
    log "Installed MQTT config at $MQTT_CONFIG_FILE"
  fi
}

install_units() {
  mkdir -p "$SYSTEMD_DIR"
  cp "$SRC_ROOT"/systemd/* "$SYSTEMD_DIR/"
}

install_udev() {
  mkdir -p "$UDEV_DIR"
  cp "$SRC_ROOT"/udev/* "$UDEV_DIR/"
}

switch_current() {
  mkdir -p "$INSTALL_DIR/releases"
  rm -f "$CURRENT_LINK.next"
  ln -s "releases/$VERSION" "$CURRENT_LINK.next"
  rm -f "$CURRENT_LINK"
  mv -f "$CURRENT_LINK.next" "$CURRENT_LINK"
}

enable_systemd() {
  if [ -n "$ROOT" ]; then
    return 0
  fi

  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable --now dvd-publish-one.timer dvd-ingester-doctor.timer dvd-ingester-update.timer dvd-ingester-ha-mqtt.timer
  fi

  if command -v udevadm >/dev/null 2>&1; then
    udevadm control --reload || true
  fi
}

need_root
prepare_source
install_packages
copy_release
install_config
install_units
install_udev
switch_current
enable_systemd

log "$PROJECT $VERSION installed"
