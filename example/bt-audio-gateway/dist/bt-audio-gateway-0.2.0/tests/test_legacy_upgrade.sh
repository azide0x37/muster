#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
INVALID_ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT" "$INVALID_ROOT"' EXIT INT TERM

: "${MUSTER_CLI_SOURCE:?MUSTER_CLI_SOURCE must point to the tested Muster core binary}"

LEGACY_RELEASE="$ROOT/opt/bt-audio-gateway/releases/0.1.4"
CONFIG="$ROOT/etc/bt-audio-gateway/bt-audio-gateway.env"
mkdir -p "$LEGACY_RELEASE/bin" "$ROOT/etc/bt-audio-gateway" "$ROOT/etc/systemd/system"
printf '0.1.4\n' > "$LEGACY_RELEASE/VERSION"
printf '#!/bin/sh\nexit 0\n' > "$LEGACY_RELEASE/bin/doctor.sh"
chmod 0755 "$LEGACY_RELEASE/bin/doctor.sh"
ln -s releases/0.1.4 "$ROOT/opt/bt-audio-gateway/current"
cat > "$CONFIG" <<'EOF'
BT_MAC=AA:BB:CC:DD:EE:FF
AUDIO_USER=pi
SNAPSERVER_HOST=homeassistant.local
SNAPSERVER_PORT=1704
SNAPCLIENT_ID=bt-kitchen-speaker
AUTOUPDATE=1
EOF
printf '[Unit]\nDescription=Legacy unit\n' > "$ROOT/etc/systemd/system/bt-audio-watch.service"

AUDIO_USER_TEST=$(id -un)
SUDO_USER="$AUDIO_USER_TEST" MUSTER_ROOT="$ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh

test -L "$ROOT/opt/bt-audio-gateway/current"
test "$(readlink "$ROOT/opt/bt-audio-gateway/current")" = "releases/$(cat VERSION)"
test -f "$LEGACY_RELEASE/VERSION"
grep -q "^AUDIO_USER=$AUDIO_USER_TEST$" "$CONFIG"
grep -q '^BT_MAC=AA:BB:CC:DD:EE:FF$' "$CONFIG"
test -x "$ROOT/usr/local/bin/muster"
test -f "$ROOT/etc/muster/implementations.d/bt-audio-gateway.json"
"$ROOT/usr/local/bin/muster" --root "$ROOT" validate >/dev/null

mkdir -p "$INVALID_ROOT/etc/bt-audio-gateway"
cat > "$INVALID_ROOT/etc/bt-audio-gateway/bt-audio-gateway.env" <<'EOF'
BT_MAC=AA:BB:CC:DD:EE:FF
AUDIO_USER=deliberately-missing-user
SNAPSERVER_HOST=homeassistant.local
SNAPSERVER_PORT=1704
SNAPCLIENT_ID=bt-kitchen-speaker
EOF
if SUDO_USER="$AUDIO_USER_TEST" MUSTER_ROOT="$INVALID_ROOT" MUSTER_SKIP_PACKAGES=1 \
  ./bin/install.sh > "$INVALID_ROOT/install.log" 2>&1; then
  echo "intentional invalid audio user was silently rewritten" >&2
  exit 1
fi
grep -q 'refusing to rewrite an intentional setting' "$INVALID_ROOT/install.log"
grep -q '^AUDIO_USER=deliberately-missing-user$' \
  "$INVALID_ROOT/etc/bt-audio-gateway/bt-audio-gateway.env"
test ! -e "$INVALID_ROOT/opt/bt-audio-gateway/current"

printf 'ok: legacy 0.1.4 upgrade\n'
