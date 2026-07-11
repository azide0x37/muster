#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

: "${MUSTER_CLI_SOURCE:?MUSTER_CLI_SOURCE must point to the tested Muster core binary}"

release_inventory() {
  release="$1"
  output="$2"
  : > "$output"
  find "$release" -type f -print | sort | while IFS= read -r file; do
    relative=${file#"$release/"}
    if command -v sha256sum >/dev/null 2>&1; then
      digest=$(sha256sum "$file" | awk '{print $1}')
    else
      digest=$(shasum -a 256 "$file" | awk '{print $1}')
    fi
    printf '%s  %s\n' "$digest" "$relative"
  done > "$output"
}

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

CONFIG="$ROOT/etc/dvd-ingester/dvd-ingester.env"
RELEASE="$ROOT/opt/dvd-ingester/releases/$(cat VERSION)"
test -f "$CONFIG"
printf '\n# user retained setting\n' >> "$CONFIG"
release_inventory "$RELEASE" "$ROOT/release.before"
test ! -w "$RELEASE/muster.yaml"

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
release_inventory "$RELEASE" "$ROOT/release.after"
cmp "$ROOT/release.before" "$ROOT/release.after"

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh > "$ROOT/concurrent-1.log" 2>&1 &
first_pid=$!
MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh > "$ROOT/concurrent-2.log" 2>&1 &
second_pid=$!
wait "$first_pid"
wait "$second_pid"
test ! -e "$ROOT/var/lock/muster/dvd-ingester.release.lock"
for leftover in "$ROOT/opt/dvd-ingester/releases"/.*.stage.*; do
  test ! -e "$leftover"
done
release_inventory "$RELEASE" "$ROOT/release.concurrent"
cmp "$ROOT/release.before" "$ROOT/release.concurrent"

grep -q "user retained setting" "$CONFIG"
test -L "$ROOT/opt/dvd-ingester/current"
test "$(readlink "$ROOT/opt/dvd-ingester/current")" = "releases/$(cat VERSION)"
test ! -e "$ROOT/opt/dvd-ingester/current.next"
test -x "$ROOT/opt/dvd-ingester/current/bin/doctor.sh"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-rip-one"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-publish-one"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-control"
test -x "$ROOT/opt/dvd-ingester/current/bin/dvd-ha-mqtt-bridge"
test -f "$ROOT/opt/dvd-ingester/current/muster.yaml"
test -f "$ROOT/opt/dvd-ingester/current/muster.lock.json"
test -f "$ROOT/etc/systemd/system/dvd-rip@.service"
test -f "$ROOT/etc/systemd/system/dvd-publish-one.service"
test -f "$ROOT/etc/systemd/system/dvd-publish-one.timer"
test -f "$ROOT/etc/systemd/system/dvd-ingester-ha-mqtt.service"
test -f "$ROOT/etc/systemd/system/dvd-ingester-ha-mqtt.timer"
test -f "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules"
test -f "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env"
test "$(stat -f %Lp "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env" 2>/dev/null || stat -c %a "$ROOT/etc/dvd-ingester/dvd-ingester.mqtt.env")" = "600"
MUSTER_MOCK_ROOT="$ROOT/doctor-mock" "$ROOT/opt/dvd-ingester/current/bin/doctor.sh"
test -s "$ROOT/doctor-mock/run/muster/dvd-ingester/observations/doctor.json"
grep -q 'muster.observation/v1' "$ROOT/doctor-mock/run/muster/dvd-ingester/observations/doctor.json"
test -x "$ROOT/opt/muster/current/bin/muster"
test "$(readlink "$ROOT/usr/local/bin/muster")" = "../../../opt/muster/current/bin/muster"
grep -q 'implementation:dvd-ingester' "$ROOT/etc/muster/implementations.d/dvd-ingester.json"
"$ROOT/usr/local/bin/muster" --root "$ROOT" validate >/dev/null
"$ROOT/usr/local/bin/muster" --root "$ROOT" list --json | grep -q 'implementation:dvd-ingester'
