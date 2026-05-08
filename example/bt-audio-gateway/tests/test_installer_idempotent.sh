#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

CONFIG="$ROOT/etc/bt-audio-gateway/bt-audio-gateway.env"
test -f "$CONFIG"
printf '\n# user retained setting\n' >> "$CONFIG"

MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

grep -q "user retained setting" "$CONFIG"
test -L "$ROOT/opt/bt-audio-gateway/current"
test "$(readlink "$ROOT/opt/bt-audio-gateway/current")" = "releases/$(cat VERSION)"
test ! -e "$ROOT/opt/bt-audio-gateway/releases/$(cat VERSION)/current.next"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/doctor.sh"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/bt-audio-watch"
test -x "$ROOT/opt/bt-audio-gateway/current/bin/bt-audio-route"
test -f "$ROOT/etc/systemd/system/bt-audio-watch.service"
