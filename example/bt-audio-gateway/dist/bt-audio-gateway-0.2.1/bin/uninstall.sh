#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CONFIG_DIR="$ROOT/etc/$PROJECT"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
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
  systemctl disable --now bt-audio-update.timer >/dev/null 2>&1 || true
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
  if [ -f "$CONFIG_DIR/$PROJECT.env" ]; then
    # shellcheck disable=SC1090
    . "$CONFIG_DIR/$PROJECT.env"
  fi
  systemctl disable --now bt-audio-watch.service bt-audio-doctor.timer >/dev/null 2>&1 || true
  if [ -n "${AUDIO_USER:-}" ]; then
    systemctl disable --now "snapclient-bt@$AUDIO_USER.service" >/dev/null 2>&1 || true
  fi
fi

for unit in \
  bt-audio-watch.service snapclient-bt@.service \
  bt-audio-doctor.service bt-audio-doctor.timer \
  bt-audio-update.service bt-audio-update.timer; do
  rm -f "$SYSTEMD_DIR/$unit"
done
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
