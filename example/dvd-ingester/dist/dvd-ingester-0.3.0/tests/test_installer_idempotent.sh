#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

CONFIG="$ROOT/etc/dvd-ingester/dvd-ingester.env"
test -f "$CONFIG"
printf '\n# user retained setting\n' >> "$CONFIG"

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

grep -q "user retained setting" "$CONFIG"
test -L "$ROOT/opt/dvd-ingester/current"
test "$(readlink "$ROOT/opt/dvd-ingester/current")" = "releases/$(cat VERSION)"
test ! -e "$ROOT/opt/dvd-ingester/releases/$(cat VERSION)/current.next"
test -x "$ROOT/opt/dvd-ingester/current/bin/doctor.sh"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-rip-one"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-publish-one"
test -f "$ROOT/etc/systemd/system/dvd-rip@.service"
test -f "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules"
