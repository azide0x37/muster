#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

: "${MUSTER_CLI_SOURCE:?MUSTER_CLI_SOURCE must point to the tested Muster core binary}"
NEW_VERSION=$(cat VERSION)
PACKAGE_ROOT="$ROOT/package/dvd-ingester-$NEW_VERSION"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

mkdir -p "$ROOT/etc/dvd-ingester"
mkdir -p "$ROOT/etc/systemd/system" "$ROOT/etc/udev/rules.d"
mkdir -p "$ROOT/opt/dvd-ingester/releases/1.0.2/bin"
mkdir -p "$PACKAGE_ROOT"
mkdir -p "$ROOT/tmp"

cat > "$ROOT/opt/dvd-ingester/releases/1.0.2/bin/doctor.sh" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 0755 "$ROOT/opt/dvd-ingester/releases/1.0.2/bin/doctor.sh"

ln -s releases/1.0.2 "$ROOT/opt/dvd-ingester/current"
printf '%s\n' 'old managed publish timer' > "$ROOT/etc/systemd/system/dvd-publish-one.timer"
printf '%s\n' 'old managed optical rule' > "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules"

cp -R bin etc src systemd udev "$PACKAGE_ROOT/"
cp src/dvd-rip-one src/dvd-publish-one src/dvd-control src/dvd-ha-mqtt-bridge "$PACKAGE_ROOT/bin/"
cp muster.yaml muster.lock.json VERSION "$PACKAGE_ROOT/"
cat > "$PACKAGE_ROOT/bin/doctor.sh" <<'EOF'
#!/bin/sh
echo "stdout detail from failed doctor"
echo "stderr detail from failed doctor" >&2
exit 42
EOF
chmod 0755 "$PACKAGE_ROOT/bin/doctor.sh"
find "$PACKAGE_ROOT/bin" -type f -name '*.sh' -exec chmod 0755 {} \;
chmod 0755 "$PACKAGE_ROOT/bin/dvd-rip-one" "$PACKAGE_ROOT/bin/dvd-publish-one" \
  "$PACKAGE_ROOT/bin/dvd-control" "$PACKAGE_ROOT/bin/dvd-ha-mqtt-bridge"

tar -czf "$ROOT/dvd-ingester-$NEW_VERSION.tar.gz" -C "$ROOT/package" "dvd-ingester-$NEW_VERSION"
SHA="$(sha256_file "$ROOT/dvd-ingester-$NEW_VERSION.tar.gz")"

cat > "$ROOT/manifest.json" <<EOF
{
  "project": "dvd-ingester",
  "version": "$NEW_VERSION",
  "artifact": "dvd-ingester-$NEW_VERSION.tar.gz",
  "artifact_url": "file://$ROOT/dvd-ingester-$NEW_VERSION.tar.gz",
  "sha256": "$SHA",
  "installer": "install.sh"
}
EOF

cat > "$ROOT/etc/dvd-ingester/dvd-ingester.env" <<EOF
AUTOUPDATE=1
UPDATE_MANIFEST_URL=file://$ROOT/manifest.json
EOF

if MUSTER_ROOT="$ROOT" TMPDIR="$ROOT/tmp" ./bin/update.sh > "$ROOT/update.log" 2>&1; then
  cat "$ROOT/update.log"
  echo "expected update failure"
  exit 1
fi

grep -q "Doctor failed with exit status 42" "$ROOT/update.log"
grep -q "doctor: stdout detail from failed doctor" "$ROOT/update.log"
grep -q "doctor: stderr detail from failed doctor" "$ROOT/update.log"
grep -q "Health check failed after update; rolling back" "$ROOT/update.log"
test "$(readlink "$ROOT/opt/dvd-ingester/current")" = "releases/1.0.2"
test ! -e "$ROOT/etc/muster/implementations.d/dvd-ingester.json"
test -x "$ROOT/opt/muster/current/bin/muster"
grep -q '^old managed publish timer$' "$ROOT/etc/systemd/system/dvd-publish-one.timer"
grep -q '^old managed optical rule$' "$ROOT/etc/udev/rules.d/90-dvd-ingester.rules"
test ! -e "$ROOT/etc/systemd/system/dvd-ingester-doctor.timer"
