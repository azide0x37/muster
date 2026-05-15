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
test ! -e "$ROOT/opt/dvd-ingester/current.next"
test -x "$ROOT/opt/dvd-ingester/current/bin/doctor.sh"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-rip-one"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-publish-one"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-control"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-ha-mqtt-bridge"
test -f "$ROOT/etc/systemd/system/dvd-rip@.service"
test -f "$ROOT/etc/systemd/system/dvd-publish-one.service"
test -f "$ROOT/etc/systemd/system/dvd-publish-one.timer"
test -f "$ROOT/etc/systemd/system/dvd-ingester-ha-mqtt.service"
test -f "$ROOT/etc/systemd/system/dvd-ingester-ha-mqtt.timer"
test -f "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules"
test -f "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env"
test "$(stat -f %Lp "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env" 2>/dev/null || stat -c %a "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env")" = "600"
MUSTER_MOCK_ROOT="$ROOT/doctor-mock" "$ROOT/opt/dvd-ingester/current/bin/doctor.sh"
