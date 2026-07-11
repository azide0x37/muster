#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
CLI_SOURCE="${MUSTER_CLI_SOURCE:-$REPO_ROOT/.cache/bin/muster}"
CLI_VERSION="${MUSTER_CLI_VERSION:-$(cat "$REPO_ROOT/VERSION")}"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

test -x "$CLI_SOURCE"
cmp "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" "$REPO_ROOT/example/dvd-ingester/bin/muster-bootstrap.sh"
cmp "$REPO_ROOT/example/bt-audio-gateway/bin/muster-observation.sh" "$REPO_ROOT/example/dvd-ingester/bin/muster-observation.sh"
cmp "$REPO_ROOT/example/bt-audio-gateway/bin/release-transaction.sh" "$REPO_ROOT/example/dvd-ingester/bin/release-transaction.sh"

# Observation scratch state is private, unpredictable, cleaned, and honest
# about checks that cannot be completed.
fixture=$(mktemp -d)
mkdir -p "$fixture/tmp"
OBSERVATION_HELPER="$REPO_ROOT/example/bt-audio-gateway/bin/muster-observation.sh" \
OBSERVATION_OUTPUT="$fixture/doctor.json" TMPDIR="$fixture/tmp" sh -c '
  set -eu
  . "$OBSERVATION_HELPER"
  muster_observation_begin example doctor "$OBSERVATION_OUTPUT" 1
  private=$MUSTER_OBS_TMPDIR
  case "$MUSTER_OBS_CHECKS" in "$private"/*) ;; *) exit 1 ;; esac
  mode=$(stat -f %Lp "$private" 2>/dev/null || stat -c %a "$private")
  test "$mode" = 700
  muster_check unknown live-state "live state was not queried"
  muster_observation_emit
  test ! -e "$private"
' > "$fixture/stdout.json"
grep -q '"health":"unknown"' "$fixture/doctor.json"
grep -q 'one or more checks were inconclusive' "$fixture/doctor.json"
grep -q '"health":"unknown"' "$fixture/stdout.json"
rm -rf "$fixture"

install_project() {
  root="$1"
  project="$2"
  source="$3"
  MUSTER_ROOT="$root" \
    MUSTER_SKIP_PACKAGES=1 \
    MUSTER_CLI_SOURCE="$source" \
    MUSTER_CLI_VERSION="$CLI_VERSION" \
    "$REPO_ROOT/example/$project/bin/install.sh" >/dev/null
}

assert_shared_host() {
  root="$1"
  test -x "$root/opt/muster/current/bin/muster"
  test "$(readlink "$root/usr/local/bin/muster")" = "../../../opt/muster/current/bin/muster"
  test -f "$root/etc/muster/implementations.d/bt-audio-gateway.json"
  test -f "$root/etc/muster/implementations.d/dvd-ingester.json"
  output=$("$root/usr/local/bin/muster" --root "$root" list --json)
  printf '%s' "$output" | grep -q 'implementation:bt-audio-gateway'
  printf '%s' "$output" | grep -q 'implementation:dvd-ingester'
  bt_status=$("$root/usr/local/bin/muster" --root "$root" status bt-audio-gateway --json)
  printf '%s' "$bt_status" | grep -q '"status": "degraded"'
  "$root/usr/local/bin/muster" --root "$root" validate >/dev/null
  doctor=$("$root/usr/local/bin/muster" --root "$root" doctor dvd-ingester --json)
  printf '%s' "$doctor" | grep -q '"kind": "doctor"'
  grep -q 'muster.observation/v1' "$root/run/muster/dvd-ingester/observations/doctor.json"
}

exercise_order() {
  first="$1"
  second="$2"
  root=$(mktemp -d)
  install_project "$root" "$first" "$CLI_SOURCE"
  core_target=$(readlink "$root/opt/muster/current")
  # A second implementation must not consult or require a core source once a
  # viable independently owned core exists.
  install_project "$root" "$second" "$root/does-not-exist"
  test "$(readlink "$root/opt/muster/current")" = "$core_target"
  assert_shared_host "$root"
  cp "$root/etc/muster/implementations.d/$first.json" "$root/registration.before"
  install_project "$root" "$first" "$root/does-not-exist"
  cmp "$root/registration.before" "$root/etc/muster/implementations.d/$first.json"

  MUSTER_ROOT="$root" "$REPO_ROOT/example/$first/bin/uninstall.sh" >/dev/null
  test ! -e "$root/etc/muster/implementations.d/$first.json"
  test -e "$root/etc/muster/implementations.d/$second.json"
  test -x "$root/usr/local/bin/muster"
  "$root/usr/local/bin/muster" --root "$root" validate >/dev/null

  MUSTER_ROOT="$root" "$REPO_ROOT/example/$second/bin/uninstall.sh" >/dev/null
  test ! -e "$root/etc/muster/implementations.d/$second.json"
  test -x "$root/opt/muster/current/bin/muster"
  test -L "$root/usr/local/bin/muster"
  output=$("$root/usr/local/bin/muster" --root "$root" list --json)
  test "$output" = "[]"
  rm -rf "$root"
}

exercise_order bt-audio-gateway dvd-ingester
exercise_order dvd-ingester bt-audio-gateway

# Concurrent first installs serialize shared-core publication while retaining
# two independently owned registrations.
root=$(mktemp -d)
install_project "$root" bt-audio-gateway "$CLI_SOURCE" &
first_pid=$!
install_project "$root" dvd-ingester "$CLI_SOURCE" &
second_pid=$!
wait "$first_pid"
wait "$second_pid"
assert_shared_host "$root"
rm -rf "$root"

# A foreign PATH entry is never overwritten.
root=$(mktemp -d)
mkdir -p "$root/usr/local/bin"
printf '%s\n' foreign > "$root/usr/local/bin/muster"
if install_project "$root" bt-audio-gateway "$CLI_SOURCE" >/dev/null 2>&1; then
  echo "foreign muster path should have rejected installation" >&2
  exit 1
fi
test "$(cat "$root/usr/local/bin/muster")" = foreign
test ! -e "$root/opt/muster/current"
test ! -e "$root/etc/muster/implementations.d/bt-audio-gateway.json"
rm -rf "$root"

# A foreign shared-core pointer is never executed or replaced.
root=$(mktemp -d)
mkdir -p "$root/opt/muster"
ln -s /foreign/muster-core "$root/opt/muster/current"
if MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null 2>&1; then
  echo "foreign core pointer should have rejected installation" >&2
  exit 1
fi
test "$(readlink "$root/opt/muster/current")" = /foreign/muster-core
test ! -e "$root/usr/local/bin/muster"
rm -rf "$root"

# A checksum failure publishes neither core pointer, PATH entry, nor registry.
root=$(mktemp -d)
fixture=$(mktemp -d)
mkdir -p "$fixture/tmp"
printf '%s\n' not-a-tarball > "$fixture/core.tar.gz"
cat > "$fixture/manifest.json" <<EOF
{"version":"$CLI_VERSION","platform":"linux-amd64","artifact_url":"file://$fixture/core.tar.gz","sha256":"0000000000000000000000000000000000000000000000000000000000000000"}
EOF
if TMPDIR="$fixture/tmp" MUSTER_ROOT="$root" MUSTER_CLI_SOURCE= MUSTER_CORE_PLATFORM=linux-amd64 MUSTER_CORE_MANIFEST_URL="file://$fixture/manifest.json" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null 2>&1; then
  echo "bad core checksum should have failed" >&2
  exit 1
fi
test ! -e "$root/opt/muster/current"
test ! -e "$root/usr/local/bin/muster"
test ! -e "$root/etc/muster/implementations.d"
test -z "$(find "$fixture/tmp" -mindepth 1 -print -quit)"
rm -rf "$root" "$fixture"

# Download scratch state and the shared lock are cleaned when bootstrap is
# interrupted after creating its private work directory.
root=$(mktemp -d)
fixture=$(mktemp -d)
mkdir -p "$fixture/bin" "$fixture/tmp"
cat > "$fixture/bin/curl" <<'EOF'
#!/bin/sh
kill -TERM "$PPID"
exit 1
EOF
chmod 0755 "$fixture/bin/curl"
if PATH="$fixture/bin:$PATH" TMPDIR="$fixture/tmp" MUSTER_ROOT="$root" \
  MUSTER_CLI_SOURCE= MUSTER_CORE_PLATFORM=linux-amd64 \
  MUSTER_CORE_MANIFEST_URL=https://invalid.example/muster.json \
  "$REPO_ROOT/example/dvd-ingester/bin/muster-bootstrap.sh" ensure >/dev/null 2>&1; then
  echo "interrupted core download should not have succeeded" >&2
  exit 1
fi
test -z "$(find "$fixture/tmp" -mindepth 1 -print -quit)"
test ! -e "$root/opt/muster/.bootstrap.lock"
test ! -e "$root/opt/muster/current"
test ! -e "$root/usr/local/bin/muster"
rm -rf "$root" "$fixture"

# A valid downloaded core manifest follows the same checksum, platform,
# archive-safety, version, current-pointer, and PATH publication path used on a
# real first installation.
root=$(mktemp -d)
fixture=$(mktemp -d)
mkdir -p "$fixture/tmp"
name="muster-$CLI_VERSION-linux-amd64"
mkdir -p "$fixture/$name/bin"
cp "$CLI_SOURCE" "$fixture/$name/bin/muster"
chmod 0755 "$fixture/$name/bin/muster"
printf '%s\n' "$CLI_VERSION" > "$fixture/$name/VERSION"
COPYFILE_DISABLE=1 tar -czf "$fixture/core.tar.gz" -C "$fixture" "$name"
sha=$(sha256_file "$fixture/core.tar.gz")
cat > "$fixture/manifest.json" <<EOF
{"version":"$CLI_VERSION","platform":"linux-amd64","artifact_url":"file://$fixture/core.tar.gz","sha256":"$sha"}
EOF
TMPDIR="$fixture/tmp" MUSTER_ROOT="$root" MUSTER_CLI_SOURCE= MUSTER_CORE_PLATFORM=linux-amd64 \
MUSTER_CORE_MANIFEST_URL="file://$fixture/manifest.json" \
  "$REPO_ROOT/example/dvd-ingester/bin/muster-bootstrap.sh" ensure >/dev/null
test "$("$root/opt/muster/current/bin/muster" version)" = "$CLI_VERSION"
test "$(readlink "$root/opt/muster/current")" = "releases/$CLI_VERSION"
test "$(readlink "$root/usr/local/bin/muster")" = ../../../opt/muster/current/bin/muster
test -f "$root/opt/muster/releases/$CLI_VERSION/release-manifest.json"
test -z "$(find "$fixture/tmp" -mindepth 1 -print -quit)"
rm -rf "$root" "$fixture"

# Even a correctly checksummed archive is rejected when it contains symbolic
# or hard links; core extraction accepts regular files and directories only.
root=$(mktemp -d)
fixture=$(mktemp -d)
mkdir -p "$fixture/tmp"
name="muster-$CLI_VERSION-linux-amd64"
mkdir -p "$fixture/$name/bin"
cp "$CLI_SOURCE" "$fixture/$name/bin/muster"
chmod 0755 "$fixture/$name/bin/muster"
ln "$fixture/$name/bin/muster" "$fixture/$name/bin/muster-hardlink"
ln -s muster "$fixture/$name/bin/muster-symlink"
printf '%s\n' "$CLI_VERSION" > "$fixture/$name/VERSION"
COPYFILE_DISABLE=1 tar -czf "$fixture/core.tar.gz" -C "$fixture" "$name"
sha=$(sha256_file "$fixture/core.tar.gz")
cat > "$fixture/manifest.json" <<EOF
{"version":"$CLI_VERSION","platform":"linux-amd64","artifact_url":"file://$fixture/core.tar.gz","sha256":"$sha"}
EOF
if TMPDIR="$fixture/tmp" MUSTER_ROOT="$root" MUSTER_CLI_SOURCE= MUSTER_CORE_PLATFORM=linux-amd64 \
  MUSTER_CORE_MANIFEST_URL="file://$fixture/manifest.json" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null 2>&1; then
  echo "linked core archive should have been rejected" >&2
  exit 1
fi
test ! -e "$root/opt/muster/current"
test ! -e "$root/usr/local/bin/muster"
test -z "$(find "$fixture/tmp" -mindepth 1 -print -quit)"
rm -rf "$root" "$fixture"

# The portable directory-lock fallback recovers locks whose recorded owner is
# dead, but never steals a lock from a live process.
root=$(mktemp -d)
mkdir -p "$root/opt/muster/.bootstrap.lock"
printf '%s\n' 99999999 > "$root/opt/muster/.bootstrap.lock/owner"
MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
MUSTER_BOOTSTRAP_DISABLE_FLOCK=1 \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null
test -x "$root/opt/muster/current/bin/muster"
test ! -e "$root/opt/muster/.bootstrap.lock"
rm -rf "$root"

root=$(mktemp -d)
sleep 5 &
holder=$!
mkdir -p "$root/opt/muster/.bootstrap.lock"
printf '%s\n' "$holder" > "$root/opt/muster/.bootstrap.lock/owner"
if MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
  MUSTER_BOOTSTRAP_DISABLE_FLOCK=1 MUSTER_BOOTSTRAP_LOCK_TIMEOUT_SECONDS=0 \
  "$REPO_ROOT/example/dvd-ingester/bin/muster-bootstrap.sh" ensure >/dev/null 2>&1; then
  echo "live bootstrap lock should not have been stolen" >&2
  kill "$holder" 2>/dev/null || true
  exit 1
fi
kill "$holder" 2>/dev/null || true
wait "$holder" 2>/dev/null || true
test ! -e "$root/opt/muster/current"
test ! -e "$root/usr/local/bin/muster"
rm -rf "$root"

# Registration accepts canonical paths owned by the implementation, rejects
# path interpolation outside that namespace, and refuses a foreign destination
# object without nesting a staging file inside it.
root=$(mktemp -d)
bootstrap="$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh"
MUSTER_ROOT="$root" "$bootstrap" register example
cp "$root/etc/muster/implementations.d/example.json" "$root/registration.before"
if MUSTER_ROOT="$root" "$bootstrap" register example '/etc/passwd"' /opt/example/current/muster.lock.json >/dev/null 2>&1; then
  echo "registry path outside project ownership should have been rejected" >&2
  exit 1
fi
cmp "$root/registration.before" "$root/etc/muster/implementations.d/example.json"
mkdir "$root/etc/muster/implementations.d/foreign.json"
if MUSTER_ROOT="$root" "$bootstrap" register foreign >/dev/null 2>&1; then
  echo "foreign registry destination should have been rejected" >&2
  exit 1
fi
test -d "$root/etc/muster/implementations.d/foreign.json"
test -z "$(find "$root/etc/muster/implementations.d/foreign.json" -mindepth 1 -print -quit)"
if MUSTER_ROOT="$root" "$bootstrap" register example /opt/example/current/muster.yaml \
  /opt/example/current/muster.lock.json extra >/dev/null 2>&1; then
  echo "register should reject more than four arguments" >&2
  exit 1
fi
cmp "$root/registration.before" "$root/etc/muster/implementations.d/example.json"
rm -rf "$root"

# A broken managed pointer is repaired without replacing the managed PATH link.
root=$(mktemp -d)
mkdir -p "$root/opt/muster" "$root/usr/local/bin"
ln -s releases/missing "$root/opt/muster/current"
ln -s ../../../opt/muster/current/bin/muster "$root/usr/local/bin/muster"
MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null
test -x "$root/opt/muster/current/bin/muster"
test "$(readlink "$root/usr/local/bin/muster")" = ../../../opt/muster/current/bin/muster
rm -rf "$root"

# A partial same-version release is replaced, never used as a directory into
# which the completed staging release is accidentally nested.
root=$(mktemp -d)
mkdir -p "$root/opt/muster/releases/$CLI_VERSION/bin"
printf '%s\n' partial > "$root/opt/muster/releases/$CLI_VERSION/PARTIAL"
ln -s "releases/$CLI_VERSION" "$root/opt/muster/current"
MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/muster-bootstrap.sh" ensure >/dev/null
test -x "$root/opt/muster/current/bin/muster"
test "$("$root/opt/muster/current/bin/muster" version)" = "$CLI_VERSION"
test ! -e "$root/opt/muster/releases/$CLI_VERSION/PARTIAL"
if find "$root/opt/muster/releases/$CLI_VERSION" -name ".${CLI_VERSION}.new.*" | grep -q .; then
  echo "completed core was nested inside a partial release" >&2
  exit 1
fi
if find "$root/opt/muster" -name '.current.next.*' | grep -q .; then
  echo "current pointer staging link was nested instead of atomically renamed" >&2
  exit 1
fi
rm -rf "$root"

# An executable in the requested release directory is reusable only when its
# reported version matches the directory. Otherwise bootstrap replaces it with
# a validated release before publishing current or PATH.
root=$(mktemp -d)
mkdir -p "$root/opt/muster/releases/$CLI_VERSION/bin"
cat > "$root/opt/muster/releases/$CLI_VERSION/bin/muster" <<'EOF'
#!/bin/sh
printf '%s\n' wrong-version
EOF
chmod 0755 "$root/opt/muster/releases/$CLI_VERSION/bin/muster"
ln -s "releases/$CLI_VERSION" "$root/opt/muster/current"
MUSTER_ROOT="$root" MUSTER_CLI_SOURCE="$CLI_SOURCE" MUSTER_CLI_VERSION="$CLI_VERSION" \
  "$REPO_ROOT/example/dvd-ingester/bin/muster-bootstrap.sh" ensure >/dev/null
test "$("$root/opt/muster/current/bin/muster" version)" = "$CLI_VERSION"
test "$(readlink "$root/usr/local/bin/muster")" = ../../../opt/muster/current/bin/muster
rm -rf "$root"

# A failed doctor still publishes and returns the new unhealthy observation;
# it never falls back to an older healthy result.
root=$(mktemp -d)
install_project "$root" dvd-ingester "$CLI_SOURCE"
rm -f "$root/etc/dvd-ingester/dvd-ingester.env"
if doctor_output=$("$root/usr/local/bin/muster" --root "$root" doctor dvd-ingester --json 2>"$root/doctor.stderr"); then
  echo "doctor with missing required configuration should have failed" >&2
  exit 1
fi
printf '%s' "$doctor_output" | grep -q '"kind": "doctor"'
printf '%s' "$doctor_output" | grep -q '"status": "unhealthy"'
grep -q '"health":"unhealthy"' "$root/run/muster/dvd-ingester/observations/doctor.json"
rm -rf "$root"

# Missing live Bluetooth configuration must be evidence, not a set -u abort.
fixture=$(mktemp -d)
if MUSTER_CONFIG_FILE="$fixture/missing.env" MUSTER_DOCTOR_OUTPUT="$fixture/doctor.json" \
  "$REPO_ROOT/example/bt-audio-gateway/bin/doctor.sh" --runtime --json >"$fixture/stdout" 2>"$fixture/stderr"; then
  echo "Bluetooth doctor with missing config should have failed" >&2
  exit 1
fi
grep -q 'muster.observation/v1' "$fixture/doctor.json"
grep -q 'missing config' "$fixture/stdout"
rm -rf "$fixture"

# Observation helpers must report persistence failure instead of claiming a
# successful doctor whose evidence could not be refreshed.
(
  # shellcheck disable=SC1091
  . "$REPO_ROOT/example/bt-audio-gateway/bin/muster-observation.sh"
  blocker=$(mktemp)
  muster_observation_begin test doctor "$blocker/doctor.json" 1
  muster_check healthy sample "sample check"
  if muster_observation_emit >/dev/null 2>&1; then
    echo "observation persistence failure should have been reported" >&2
    exit 1
  fi
  rm -f "$blocker"
)

# Committed locks are deterministic projections of their authoring manifests.
for project in bt-audio-gateway dvd-ingester; do
  regenerated=$(mktemp)
  "$CLI_SOURCE" compile "$REPO_ROOT/example/$project/muster.yaml" "$regenerated" >/dev/null
  cmp "$REPO_ROOT/example/$project/muster.lock.json" "$regenerated"
  rm -f "$regenerated"
done

printf '%s\n' "ok: shared Muster lifecycle"
