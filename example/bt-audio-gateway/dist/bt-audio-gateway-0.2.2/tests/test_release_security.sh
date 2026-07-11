#!/bin/sh
set -eu

ROOT=$(mktemp -d)
trap 'rm -rf "$ROOT"' EXIT INT TERM

PROJECT=bt-audio-gateway
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
TMP_DIR="$ROOT/work"
mkdir -p "$TMP_DIR" "$RELEASES_DIR"

# shellcheck disable=SC1091
. ./bin/release-transaction.sh

expect_failure() {
  label="$1"
  shift
  if ("$@") >/dev/null 2>&1; then
    echo "expected failure: $label" >&2
    exit 1
  fi
}

expect_failure "traversal version" release_require_version ../1.2.3
expect_failure "incomplete version" release_require_version 1.2
release_require_version 1.2.3
release_require_version 1.2.3-rc.1

expect_failure "short SHA" release_require_sha deadbeef
expect_failure "non-hex SHA" release_require_sha zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz
release_require_sha 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

release_dir_valid . "$(cat VERSION)"
mkdir -p "$ROOT/invalid/bin"
cp VERSION muster.yaml muster.lock.json bin/muster-bootstrap.sh bin/doctor.sh "$ROOT/invalid/"
mv "$ROOT/invalid/muster-bootstrap.sh" "$ROOT/invalid/bin/muster-bootstrap.sh"
mv "$ROOT/invalid/doctor.sh" "$ROOT/invalid/bin/doctor.sh"
printf '%s\n' 9.9.9 > "$ROOT/invalid/VERSION"
if release_dir_valid "$ROOT/invalid" 9.9.9; then
  echo "release validation accepted a VERSION/lock mismatch" >&2
  exit 1
fi

archive_version=9.9.9
prefix="$PROJECT-$archive_version"
mkdir -p "$ROOT/archive/$prefix"
printf '%s\n' safe > "$ROOT/archive/$prefix/file"
tar -czf "$ROOT/safe.tar.gz" -C "$ROOT/archive" "$prefix"
release_validate_archive "$ROOT/safe.tar.gz" "$archive_version"

ln -s file "$ROOT/archive/$prefix/link"
tar -czf "$ROOT/link.tar.gz" -C "$ROOT/archive" "$prefix"
expect_failure "archive symlink" release_validate_archive "$ROOT/link.tar.gz" "$archive_version"

mkdir -p "$ROOT/outside/not-$prefix"
printf '%s\n' unsafe > "$ROOT/outside/not-$prefix/file"
tar -czf "$ROOT/outside.tar.gz" -C "$ROOT/outside" "not-$prefix"
expect_failure "archive outside project/version root" release_validate_archive "$ROOT/outside.tar.gz" "$archive_version"

LOCK_PATH="$ROOT/var/lock/muster/$PROJECT.release.lock"
mkdir -p "$LOCK_PATH"
dead_pid=999999
while kill -0 "$dead_pid" 2>/dev/null; do dead_pid=$((dead_pid + 1)); done
{
  printf 'project=%s\n' "$PROJECT"
  printf 'pid=%s\n' "$dead_pid"
  printf 'started=1\n'
  printf 'token=stale\n'
} > "$LOCK_PATH/owner"
release_acquire_lock
grep -q "^pid=$$\$" "$LOCK_PATH/owner"
grep -q '^started=[0-9][0-9]*$' "$LOCK_PATH/owner"
release_unlock
test ! -e "$LOCK_PATH"

mkdir -p "$LOCK_PATH"
now=$(date +%s)
{
  printf 'project=%s\n' "$PROJECT"
  printf 'pid=%s\n' "$$"
  printf 'started=%s\n' "$now"
  printf 'token=live\n'
} > "$LOCK_PATH/owner"
if MUSTER_ROOT="$ROOT" MUSTER_RELEASE_LOCK_TIMEOUT_SECONDS=1 ./bin/uninstall.sh >/dev/null 2>&1; then
  echo "uninstall bypassed a live project transaction lock" >&2
  exit 1
fi
grep -q '^token=live$' "$LOCK_PATH/owner"
rm -rf "$LOCK_PATH"
