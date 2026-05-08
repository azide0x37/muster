#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
INSTALL_DIR="$ROOT/opt/$PROJECT"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
PURGE=0

if [ "${1:-}" = "--purge" ]; then
  PURGE=1
fi

if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
  echo "uninstall.sh must run as root. Use sudo, or set MUSTER_ROOT for a staged uninstall." >&2
  exit 1
fi

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  if [ -f "$CONFIG_DIR/$PROJECT.env" ]; then
    # shellcheck disable=SC1090
    . "$CONFIG_DIR/$PROJECT.env"
  fi
  systemctl disable --now bt-audio-watch.service bt-audio-doctor.timer bt-audio-update.timer || true
  if [ -n "${AUDIO_USER:-}" ]; then
    systemctl disable --now "snapclient-bt@$AUDIO_USER.service" || true
  fi
fi

rm -f "$SYSTEMD_DIR/bt-audio-watch.service"
rm -f "$SYSTEMD_DIR/snapclient-bt@.service"
rm -f "$SYSTEMD_DIR/bt-audio-doctor.service"
rm -f "$SYSTEMD_DIR/bt-audio-doctor.timer"
rm -f "$SYSTEMD_DIR/bt-audio-update.service"
rm -f "$SYSTEMD_DIR/bt-audio-update.timer"
rm -rf "$INSTALL_DIR"

if [ "$PURGE" = "1" ]; then
  rm -rf "$CONFIG_DIR"
else
  printf 'Preserved %s. Pass --purge to remove config.\n' "$CONFIG_DIR"
fi

if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
fi
