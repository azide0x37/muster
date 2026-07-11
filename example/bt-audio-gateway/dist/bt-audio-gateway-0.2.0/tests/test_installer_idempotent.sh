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

CONFIG="$ROOT/etc/bt-audio-gateway/bt-audio-gateway.env"
RELEASE="$ROOT/opt/bt-audio-gateway/releases/$(cat VERSION)"
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
test ! -e "$ROOT/var/lock/muster/bt-audio-gateway.release.lock"
for leftover in "$ROOT/opt/bt-audio-gateway/releases"/.*.stage.*; do
  test ! -e "$leftover"
done
release_inventory "$RELEASE" "$ROOT/release.concurrent"
cmp "$ROOT/release.before" "$ROOT/release.concurrent"

grep -q "user retained setting" "$CONFIG"
test -L "$ROOT/opt/bt-audio-gateway/current"
test "$(readlink "$ROOT/opt/bt-audio-gateway/current")" = "releases/$(cat VERSION)"
test ! -e "$ROOT/opt/bt-audio-gateway/releases/$(cat VERSION)/current.next"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/doctor.sh"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/bt-audio-watch"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/bt-audio-route"
test -f "$ROOT/opt/bt-audio-gateway/current/muster.yaml"
test -f "$ROOT/opt/bt-audio-gateway/current/muster.lock.json"
test -f "$ROOT/etc/systemd/system/bt-audio-watch.service"
test -x "$ROOT/opt/muster/current/bin/muster"
test "$(readlink "$ROOT/usr/local/bin/muster")" = "../../../opt/muster/current/bin/muster"
grep -q 'implementation:bt-audio-gateway' "$ROOT/etc/muster/implementations.d/bt-audio-gateway.json"
"$ROOT/usr/local/bin/muster" --root "$ROOT" validate >/dev/null
"$ROOT/usr/local/bin/muster" --root "$ROOT" list --json | grep -q 'implementation:bt-audio-gateway'
