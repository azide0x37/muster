#!/bin/sh
set -eu

PROJECT="dvd-ingester"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
CONFIG_FILE="$CONFIG_DIR/$PROJECT.env"
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

prepare_source() {
  if [ -f "$SRC_ROOT/muster.yaml" ] && [ -f "$SRC_ROOT/src/dvd-rip-one" ]; then
    VERSION="${VERSION:-$(cat "$SRC_ROOT/VERSION" 2>/dev/null || echo 0.1.0)}"
    RELEASE_DIR="$INSTALL_DIR/releases/$VERSION"
    return 0
  fi

  MANIFEST_URL="${INSTALL_MANIFEST_URL:-$DEFAULT_MANIFEST_URL}"
  if printf '%s' "$MANIFEST_URL" | grep -q '[<>]'; then
    echo "No checkout files found and INSTALL_MANIFEST_URL is not configured." >&2
    exit 1
  fi

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

install_packages() {
  if [ "${MUSTER_SKIP_PACKAGES:-0}" = "1" ]; then
    log "Skipping package install because MUSTER_SKIP_PACKAGES=1"
    return 0
  fi

  if command -v apt-get >/dev/null 2>&1; then
    apt-get update
    apt-get install -y handbrake-cli dvdbackup lsdvd dvd+rw-tools jq eject util-linux rsync curl ca-certificates
  else
    log "apt-get not found; skipping package install"
  fi
}

copy_release() {
  mkdir -p "$RELEASE_DIR/bin" "$RELEASE_DIR/systemd" "$RELEASE_DIR/etc" "$RELEASE_DIR/udev" "$RELEASE_DIR/doc"
  cp "$SRC_ROOT"/bin/*.sh "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/src/dvd-rip-one "$SRC_ROOT"/src/dvd-publish-one "$RELEASE_DIR/bin/"
  cp "$SRC_ROOT"/systemd/* "$RELEASE_DIR/systemd/"
  cp "$SRC_ROOT"/udev/* "$RELEASE_DIR/udev/"
  cp "$SRC_ROOT"/etc/* "$RELEASE_DIR/etc/"
  cp "$SRC_ROOT"/README.md "$SRC_ROOT"/MUSTER.md "$SRC_ROOT"/RELEASE.md "$RELEASE_DIR/doc/"
  chmod 0755 "$RELEASE_DIR/bin"/*.sh "$RELEASE_DIR/bin/dvd-rip-one" "$RELEASE_DIR/bin/dvd-publish-one"
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

  # shellcheck disable=SC1090
  . "$CONFIG_FILE"
  mkdir -p "$ROOT${BASE_DIR:-/var/lib/dvd-ingester}" "$ROOT${WORK_DIR:-/var/lib/dvd-ingester/work}" "$ROOT${LOG_DIR:-/var/lib/dvd-ingester/logs}" "$ROOT${DEST_DIR:-/mnt/nas/DVD_Rips}"
  touch "$ROOT${RIPPED_DB:-/var/lib/dvd-ingester/ripped.jsonl}"
}

install_units_and_udev() {
  mkdir -p "$SYSTEMD_DIR" "$UDEV_DIR"
  cp "$SRC_ROOT"/systemd/* "$SYSTEMD_DIR/"
  cp "$SRC_ROOT"/udev/* "$UDEV_DIR/"
}

switch_current() {
  mkdir -p "$INSTALL_DIR/releases"
  rm -f "$CURRENT_LINK.next"
  ln -s "releases/$VERSION" "$CURRENT_LINK.next"
  rm -f "$CURRENT_LINK"
  mv -f "$CURRENT_LINK.next" "$CURRENT_LINK"
}

reload_managers() {
  if [ -n "$ROOT" ]; then
    return 0
  fi

  systemctl daemon-reload
  systemctl enable --now dvd-ingester-doctor.timer dvd-ingester-update.timer
  udevadm control --reload-rules
  udevadm trigger
}

need_root
prepare_source
install_packages
copy_release
install_config
install_units_and_udev
switch_current
reload_managers

log "$PROJECT $VERSION installed"
