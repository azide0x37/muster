#!/bin/sh
set -eu

PROJECT="dvd-ingester"
ROOT="${MUSTER_ROOT:-}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CONFIG_DIR="$ROOT/etc/$PROJECT"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
UDEV_DIR="$ROOT/etc/udev/rules.d"
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_DIR=""
TMP_CREATED=0
PURGE=0

# shellcheck disable=SC1091
. "$SCRIPT_DIR/release-transaction.sh"

cleanup() {
  status=$?
  trap - EXIT INT TERM
  release_unlock
  if [ "$TMP_CREATED" = "1" ] && [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM

case "${1:-}" in
  '') ;;
  --purge) PURGE=1 ;;
  *) release_die "usage: uninstall.sh [--purge]" ;;
esac

if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
  release_die "uninstall.sh must run as root; use sudo or set MUSTER_ROOT for a staged uninstall"
fi

# Stop the update trigger before joining the project transaction lock. An
# already-running update owns the lock and must finish or roll back first.
if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now dvd-ingester-update.timer >/dev/null 2>&1 || true
fi

old_umask=$(umask)
umask 077
TMP_DIR=$(mktemp -d "$TMP_PARENT/$PROJECT-uninstall.XXXXXX") || {
  umask "$old_umask"
  release_die "could not create private uninstall workspace"
}
umask "$old_umask"
TMP_CREATED=1
release_acquire_lock

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now dvd-publish-one.timer dvd-ingester-doctor.timer dvd-ingester-ha-mqtt.timer >/dev/null 2>&1 || true
  systemctl stop 'dvd-rip@*.service' dvd-publish-one.service dvd-ingester-doctor.service dvd-ingester-update.service dvd-ingester-ha-mqtt.service >/dev/null 2>&1 || true
  systemctl unmask dvd-rip@.service >/dev/null 2>&1 || true
fi

for unit in \
  dvd-rip@.service dvd-publish-one.service dvd-publish-one.timer \
  dvd-ingester-doctor.service dvd-ingester-doctor.timer \
  dvd-ingester-update.service dvd-ingester-update.timer \
  dvd-ingester-ha-mqtt.service dvd-ingester-ha-mqtt.timer; do
  rm -f "$SYSTEMD_DIR/$unit"
done
rm -f "$UDEV_DIR/90-dvd-ingester.rules"
"$SCRIPT_DIR/muster-bootstrap.sh" unregister "$PROJECT"
rm -rf "$INSTALL_DIR"

if [ "$PURGE" = "1" ]; then
  rm -rf "$CONFIG_DIR"
else
  printf 'Preserved %s. Pass --purge to remove config.\n' "$CONFIG_DIR" || true
fi

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || true
fi
if [ -z "$ROOT" ] && command -v udevadm >/dev/null 2>&1; then
  udevadm control --reload >/dev/null 2>&1 || true
fi
