#!/bin/sh
set -eu

PROJECT="dvd-ingester"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
INSTALL_DIR="$ROOT/opt/$PROJECT"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
UDEV_DIR="$ROOT/etc/udev/rules.d"
PURGE=0

if [ "${1:-}" = "--purge" ]; then
  PURGE=1
fi

if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
  echo "uninstall.sh must run as root. Use sudo, or set MUSTER_ROOT for a staged uninstall." >&2
  exit 1
fi

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now dvd-publish-one.timer dvd-ingester-doctor.timer dvd-ingester-update.timer dvd-ingester-ha-mqtt.timer || true
  systemctl stop 'dvd-rip@*.service' dvd-publish-one.service dvd-ingester-doctor.service dvd-ingester-update.service dvd-ingester-ha-mqtt.service || true
  systemctl unmask dvd-rip@.service || true
fi

rm -f "$SYSTEMD_DIR/dvd-rip@.service"
rm -f "$SYSTEMD_DIR/dvd-publish-one.service"
rm -f "$SYSTEMD_DIR/dvd-publish-one.timer"
rm -f "$SYSTEMD_DIR/dvd-ingester-doctor.service"
rm -f "$SYSTEMD_DIR/dvd-ingester-doctor.timer"
rm -f "$SYSTEMD_DIR/dvd-ingester-update.service"
rm -f "$SYSTEMD_DIR/dvd-ingester-update.timer"
rm -f "$SYSTEMD_DIR/dvd-ingester-ha-mqtt.service"
rm -f "$SYSTEMD_DIR/dvd-ingester-ha-mqtt.timer"
rm -f "$UDEV_DIR/90-dvd-ingester.rules"
rm -rf "$INSTALL_DIR"

if [ "$PURGE" = "1" ]; then
  rm -rf "$CONFIG_DIR"
else
  printf 'Preserved %s. Pass --purge to remove config.\n' "$CONFIG_DIR"
fi

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
fi

if [ -z "$ROOT" ] && command -v udevadm >/dev/null 2>&1; then
  udevadm control --reload || true
fi
