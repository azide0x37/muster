#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

: "${MUSTER_CLI_SOURCE:?MUSTER_CLI_SOURCE must point to the tested Muster core binary}"
NEW_VERSION=$(cat VERSION)
PACKAGE_ROOT="$ROOT/package/bt-audio-gateway-$NEW_VERSION"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

mkdir -p "$ROOT/etc/bt-audio-gateway"
mkdir -p "$ROOT/etc/systemd/system"
mkdir -p "$ROOT/opt/bt-audio-gateway/releases/0.1.3/bin"
mkdir -p "$PACKAGE_ROOT"
mkdir -p "$ROOT/tmp"

cat > "$ROOT/opt/bt-audio-gateway/releases/0.1.3/bin/doctor.sh" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 0755 "$ROOT/opt/bt-audio-gateway/releases/0.1.3/bin/doctor.sh"
ln -s releases/0.1.3 "$ROOT/opt/bt-audio-gateway/current"
printf '%s\n' 'old managed watcher unit' > "$ROOT/etc/systemd/system/bt-audio-watch.service"

cp -R bin etc src systemd "$PACKAGE_ROOT/"
cp src/bt-audio-watch src/bt-audio-route "$PACKAGE_ROOT/bin/"
cp muster.yaml muster.lock.json VERSION "$PACKAGE_ROOT/"
cat > "$PACKAGE_ROOT/bin/doctor.sh" <<'EOF'
#!/bin/sh
exit 42
EOF
chmod 0755 "$PACKAGE_ROOT/bin/doctor.sh"
find "$PACKAGE_ROOT/bin" -type f -name '*.sh' -exec chmod 0755 {} \;
chmod 0755 "$PACKAGE_ROOT/bin/bt-audio-watch" "$PACKAGE_ROOT/bin/bt-audio-route"

tar -czf "$ROOT/bt-audio-gateway-$NEW_VERSION.tar.gz" -C "$ROOT/package" "bt-audio-gateway-$NEW_VERSION"
SHA="$(sha256_file "$ROOT/bt-audio-gateway-$NEW_VERSION.tar.gz")"
cat > "$ROOT/manifest.json" <<EOF
{"project":"bt-audio-gateway","version":"$NEW_VERSION","artifact_url":"file://$ROOT/bt-audio-gateway-$NEW_VERSION.tar.gz","sha256":"$SHA"}
EOF
cat > "$ROOT/etc/bt-audio-gateway/bt-audio-gateway.env" <<EOF
AUTOUPDATE=1
UPDATE_MANIFEST_URL=file://$ROOT/manifest.json
AUDIO_USER=pi
EOF

if MUSTER_ROOT="$ROOT" TMPDIR="$ROOT/tmp" ./bin/update.sh > "$ROOT/update.log" 2>&1; then
  cat "$ROOT/update.log"
  echo "expected update failure"
  exit 1
fi

grep -q "Health check failed after update; rolling back" "$ROOT/update.log"
test "$(readlink "$ROOT/opt/bt-audio-gateway/current")" = "releases/0.1.3"
test ! -e "$ROOT/etc/muster/implementations.d/bt-audio-gateway.json"
test -x "$ROOT/opt/muster/current/bin/muster"
grep -q '^old managed watcher unit$' "$ROOT/etc/systemd/system/bt-audio-watch.service"
test ! -e "$ROOT/etc/systemd/system/bt-audio-doctor.timer"
