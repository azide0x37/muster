#!/bin/sh
set -eu

ROOT="$(mktemp -d)"
trap 'rm -rf "$ROOT"' EXIT INT TERM

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

mkdir -p "$ROOT/etc/dvd-ingester"
mkdir -p "$ROOT/opt/dvd-ingester/releases/1.0.2/bin"
mkdir -p "$ROOT/package/dvd-ingester-1.0.3/bin"
mkdir -p "$ROOT/tmp"

cat > "$ROOT/opt/dvd-ingester/releases/1.0.2/bin/doctor.sh" <<'EOF'
#!/bin/sh
exit 0
EOF
chmod 0755 "$ROOT/opt/dvd-ingester/releases/1.0.2/bin/doctor.sh"

ln -s releases/1.0.2 "$ROOT/opt/dvd-ingester/current"

cat > "$ROOT/package/dvd-ingester-1.0.3/bin/doctor.sh" <<'EOF'
#!/bin/sh
echo "stdout detail from failed doctor"
echo "stderr detail from failed doctor" >&2
exit 42
EOF
chmod 0755 "$ROOT/package/dvd-ingester-1.0.3/bin/doctor.sh"

tar -czf "$ROOT/dvd-ingester-1.0.3.tar.gz" -C "$ROOT/package" dvd-ingester-1.0.3
SHA="$(sha256_file "$ROOT/dvd-ingester-1.0.3.tar.gz")"

cat > "$ROOT/manifest.json" <<EOF
{
  "project": "dvd-ingester",
  "version": "1.0.3",
  "artifact": "dvd-ingester-1.0.3.tar.gz",
  "artifact_url": "file://$ROOT/dvd-ingester-1.0.3.tar.gz",
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
