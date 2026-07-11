#!/bin/sh
set -eu

ROOT="${MUSTER_ROOT:-}"
CORE_DIR="$ROOT/opt/muster"
CORE_RELEASES="$CORE_DIR/releases"
CORE_CURRENT="$CORE_DIR/current"
PATH_LINK="$ROOT/usr/local/bin/muster"
PATH_TARGET="../../../opt/muster/current/bin/muster"
REGISTRY_DIR="$ROOT/etc/muster/implementations.d"
DEFAULT_CORE_VERSION="0.1.0"
DEFAULT_MANIFEST_BASE="https://github.com/azide0x37/muster/releases/latest/download"
CORE_WORK=""
LOCK_KIND=""
LOCK_DIR=""
SELECTED_VERSION=""

log() {
  printf '%s\n' "$*"
}

cleanup_core_work() {
  if [ -n "${CORE_WORK:-}" ]; then
    rm -rf "$CORE_WORK"
    CORE_WORK=""
  fi
}

die() {
  cleanup_core_work
  printf 'muster-bootstrap: %s\n' "$*" >&2
  exit 1
}

json_value() {
  key="$1"
  sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$2" | head -n 1
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

platform_name() {
  if [ -n "${MUSTER_CORE_PLATFORM:-}" ]; then
    printf '%s\n' "$MUSTER_CORE_PLATFORM"
    return
  fi
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$os:$arch" in
    linux:x86_64|linux:amd64) printf '%s\n' linux-amd64 ;;
    linux:aarch64|linux:arm64) printf '%s\n' linux-arm64 ;;
    linux:armv7l|linux:armv7) printf '%s\n' linux-armv7 ;;
    *) die "unsupported core platform $os/$arch; set MUSTER_CORE_PLATFORM explicitly" ;;
  esac
}

check_path_link() {
  if [ -L "$PATH_LINK" ]; then
    target=$(readlink "$PATH_LINK")
    if [ "$target" = "$PATH_TARGET" ]; then
      return
    fi
    die "refusing to replace foreign symlink $PATH_LINK -> $target"
  fi
  if [ -e "$PATH_LINK" ]; then
    die "refusing to replace unrelated file $PATH_LINK"
  fi
}

ensure_path_link() {
  check_path_link
  mkdir -p "$(dirname "$PATH_LINK")"
  if [ -L "$PATH_LINK" ]; then
    return
  fi
  if ln -s "$PATH_TARGET" "$PATH_LINK" 2>/dev/null; then
    return
  fi
  if [ -L "$PATH_LINK" ] && [ "$(readlink "$PATH_LINK")" = "$PATH_TARGET" ]; then
    return
  fi
  die "could not publish managed PATH link $PATH_LINK"
}

version_is_safe() {
  case "$1" in
    ''|.|..|*[!a-zA-Z0-9._+-]*) return 1 ;;
    *) return 0 ;;
  esac
}

validate_version() {
  version_is_safe "$1" || die "invalid core version: $1"
}

release_is_viable() {
  release="$1"
  version="$2"
  [ -d "$release" ] && [ ! -L "$release" ] || return 1
  [ -f "$release/bin/muster" ] && [ ! -L "$release/bin/muster" ] && [ -x "$release/bin/muster" ] || return 1
  reported_version=$("$release/bin/muster" version 2>/dev/null) || return 1
  [ "$reported_version" = "$version" ]
}

core_is_viable() {
  [ -L "$CORE_CURRENT" ] || return 1
  target=$(readlink "$CORE_CURRENT")
  case "$target" in
    releases/*) version=${target#releases/} ;;
    *) return 1 ;;
  esac
  version_is_safe "$version" || return 1
  [ "$target" = "releases/$version" ] || return 1
  release_is_viable "$CORE_RELEASES/$version" "$version"
}

lock_timeout() {
  timeout="${MUSTER_BOOTSTRAP_LOCK_TIMEOUT_SECONDS:-300}"
  case "$timeout" in
    ''|*[!0-9]*) die "MUSTER_BOOTSTRAP_LOCK_TIMEOUT_SECONDS must be a non-negative integer" ;;
  esac
  printf '%s\n' "$timeout"
}

recover_stale_directory_lock() {
  owner_file="$LOCK_DIR/owner"
  [ -f "$owner_file" ] || return 1
  owner=$(cat "$owner_file" 2>/dev/null || true)
  case "$owner" in
    ''|*[!0-9]*) return 1 ;;
  esac
  if kill -0 "$owner" 2>/dev/null; then
    return 1
  fi
  [ "$(cat "$owner_file" 2>/dev/null || true)" = "$owner" ] || return 1
  rm -f "$owner_file"
  rmdir "$LOCK_DIR" 2>/dev/null
}

acquire_lock() {
  mkdir -p "$CORE_DIR"
  timeout=$(lock_timeout)
  if [ "${MUSTER_BOOTSTRAP_DISABLE_FLOCK:-0}" != "1" ] && command -v flock >/dev/null 2>&1; then
    LOCK_KIND=flock
    exec 9<"$CORE_DIR"
    if flock -w "$timeout" 9; then
      return
    fi
    exec 9<&-
    LOCK_KIND=""
    die "timed out waiting for shared core bootstrap lock after ${timeout}s"
  fi

  LOCK_KIND=directory
  LOCK_DIR="$CORE_DIR/.bootstrap.lock"
  elapsed=0
  while ! mkdir "$LOCK_DIR" 2>/dev/null; do
    if recover_stale_directory_lock; then
      continue
    fi
    if [ "$elapsed" -ge "$timeout" ]; then
      LOCK_KIND=""
      die "timed out waiting for shared core bootstrap lock after ${timeout}s"
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
  if ! printf '%s\n' "$$" > "$LOCK_DIR/owner"; then
    rmdir "$LOCK_DIR" 2>/dev/null || true
    LOCK_KIND=""
    die "could not record shared core bootstrap lock owner"
  fi
}

release_lock() {
  cleanup_core_work
  case "${LOCK_KIND:-}" in
    flock)
      flock -u 9 2>/dev/null || true
      exec 9<&-
      ;;
    directory)
      if [ -n "${LOCK_DIR:-}" ] && [ "$(cat "$LOCK_DIR/owner" 2>/dev/null || true)" = "$$" ]; then
        rm -f "$LOCK_DIR/owner"
        rmdir "$LOCK_DIR" 2>/dev/null || true
      fi
      ;;
  esac
  LOCK_KIND=""
  LOCK_DIR=""
}

safe_archive() {
  archive="$1"
  members=$(tar -tzf "$archive") || die "cannot list core archive"
  old_ifs=$IFS
  IFS='
'
  for member in $members; do
    case "$member" in
      /*|../*|*/../*|*/..) IFS=$old_ifs; die "unsafe core archive member: $member" ;;
    esac
  done
  IFS=$old_ifs

  entries=$(tar -tvzf "$archive") || die "cannot inspect core archive member types"
  old_ifs=$IFS
  IFS='
'
  for entry in $entries; do
    case "$entry" in
      -*|d*) ;;
      *) IFS=$old_ifs; die "unsafe core archive member type: $entry" ;;
    esac
  done
  IFS=$old_ifs
}

install_from_executable() {
  source_bin="$1"
  version="${MUSTER_CLI_VERSION:-$DEFAULT_CORE_VERSION}"
  validate_version "$version"
  release="$CORE_RELEASES/$version"
  if release_is_viable "$release" "$version"; then
    SELECTED_VERSION="$version"
    return
  fi

  [ -x "$source_bin" ] || die "MUSTER_CLI_SOURCE is not executable: $source_bin"
  staging="$CORE_RELEASES/.${version}.new.$$"
  rm -rf "$staging"
  mkdir -p "$staging/bin"
  cp "$source_bin" "$staging/bin/muster"
  chmod 0755 "$staging/bin/muster"
  printf '%s\n' "$version" > "$staging/VERSION"
  if ! release_is_viable "$staging" "$version"; then
    reported_version=$("$staging/bin/muster" version 2>/dev/null || true)
    rm -rf "$staging"
    die "core executable reports $reported_version, expected $version"
  fi
  publish_release "$version" "$staging"
  SELECTED_VERSION="$version"
}

install_from_manifest() {
  platform=$(platform_name)
  old_umask=$(umask)
  umask 077
  work=$(mktemp -d "${TMPDIR:-/tmp}/muster-core-bootstrap.XXXXXX") || {
    umask "$old_umask"
    die "could not create private core download directory"
  }
  umask "$old_umask"
  CORE_WORK="$work"
  manifest_url="${MUSTER_CORE_MANIFEST_URL:-$DEFAULT_MANIFEST_BASE/muster-$platform.json}"
  curl -fsSL "$manifest_url" -o "$work/manifest.json" || die "could not fetch core manifest $manifest_url"
  version=$(json_value version "$work/manifest.json")
  declared_platform=$(json_value platform "$work/manifest.json")
  artifact_url=$(json_value artifact_url "$work/manifest.json")
  expected_sha=$(json_value sha256 "$work/manifest.json")
  [ -n "$version" ] || die "core manifest has no version"
  validate_version "$version"
  [ "$declared_platform" = "$platform" ] || die "core manifest platform $declared_platform does not match $platform"
  [ -n "$artifact_url" ] || die "core manifest has no artifact_url"
  case "$expected_sha" in
    *[!0-9a-fA-F]*|'') die "core manifest sha256 is invalid" ;;
  esac
  [ "${#expected_sha}" -eq 64 ] || die "core manifest sha256 must contain 64 hexadecimal characters"
  curl -fsSL "$artifact_url" -o "$work/core.tar.gz" || die "could not fetch core artifact"
  actual_sha=$(sha256_file "$work/core.tar.gz")
  [ "$actual_sha" = "$expected_sha" ] || die "core artifact SHA256 mismatch"
  safe_archive "$work/core.tar.gz"

  release="$CORE_RELEASES/$version"
  if ! release_is_viable "$release" "$version"; then
    staging="$CORE_RELEASES/.${version}.new.$$"
    rm -rf "$staging"
    mkdir -p "$staging"
    tar -xzf "$work/core.tar.gz" -C "$staging" --strip-components=1
    if ! release_is_viable "$staging" "$version"; then
      reported_version=$("$staging/bin/muster" version 2>/dev/null || true)
      rm -rf "$staging"
      die "core artifact reports $reported_version, expected $version"
    fi
    cp "$work/manifest.json" "$staging/release-manifest.json"
    publish_release "$version" "$staging"
  fi
  cleanup_core_work
  SELECTED_VERSION="$version"
}

publish_release() {
  version="$1"
  staging="$2"
  release="$CORE_RELEASES/$version"
  previous="$CORE_RELEASES/.${version}.previous.$$"
  had_previous=0

  rm -rf "$previous"
  if [ -e "$release" ] || [ -L "$release" ]; then
    if ! mv "$release" "$previous"; then
      rm -rf "$staging"
      die "could not quarantine invalid Muster core $version"
    fi
    had_previous=1
  fi

  if mv "$staging" "$release" && release_is_viable "$release" "$version"; then
    rm -rf "$previous"
    return
  fi

  rm -rf "$release" "$staging"
  if [ "$had_previous" = "1" ]; then
    if ! mv "$previous" "$release"; then
      die "could not publish or restore Muster core $version"
    fi
  fi
  die "could not publish Muster core $version"
}

switch_core() {
  version="$1"
  mkdir -p "$CORE_RELEASES"
  if [ -e "$CORE_CURRENT" ] || [ -L "$CORE_CURRENT" ]; then
    [ -L "$CORE_CURRENT" ] || die "refusing to replace non-symlink $CORE_CURRENT"
    current_target=$(readlink "$CORE_CURRENT")
    case "$current_target" in
      releases/*)
        current_version=${current_target#releases/}
        version_is_safe "$current_version" && [ "$current_target" = "releases/$current_version" ] || \
          die "refusing to replace foreign core pointer $CORE_CURRENT -> $current_target"
        ;;
      *) die "refusing to replace foreign core pointer $CORE_CURRENT -> $current_target" ;;
    esac
  fi

  next="$CORE_DIR/.current.next.$$"
  rm -f "$next"
  ln -s "releases/$version" "$next"
  if [ ! -e "$CORE_CURRENT" ] && [ ! -L "$CORE_CURRENT" ]; then
    if mv -f "$next" "$CORE_CURRENT"; then
      return
    fi
  elif mv -Tf "$next" "$CORE_CURRENT" 2>/dev/null; then
    return
  elif [ -L "$next" ] && mv -fh "$next" "$CORE_CURRENT" 2>/dev/null; then
    return
  fi
  rm -f "$next"
  die "could not atomically select Muster core $version"
}

ensure_core() {
  if core_is_viable; then
    ensure_path_link
    return
  fi

  check_path_link
  acquire_lock
  trap 'release_lock' EXIT
  trap 'release_lock; exit 130' INT
  trap 'release_lock; exit 143' TERM
  if core_is_viable; then
    ensure_path_link
    release_lock
    trap - EXIT INT TERM
    return
  fi
  mkdir -p "$CORE_RELEASES"
  if [ -n "${MUSTER_CLI_SOURCE:-}" ]; then
    install_from_executable "$MUSTER_CLI_SOURCE"
  else
    install_from_manifest
  fi
  version="$SELECTED_VERSION"
  switch_core "$version"
  if ! core_is_viable; then
    die "published Muster core $version is not viable"
  fi
  ensure_path_link
  release_lock
  trap - EXIT INT TERM
  log "Installed Muster core $version"
}

validate_project() {
  project="$1"
  case "$project" in
    ''|*[!a-zA-Z0-9._-]*) die "invalid project name: $project" ;;
  esac
}

validate_registry_path() {
  project="$1"
  path="$2"
  label="$3"
  case "$path" in
    "/opt/$project/"*) ;;
    *) die "$label path must remain under /opt/$project" ;;
  esac
  case "$path" in
    *[!a-zA-Z0-9_./+-]*) die "$label path contains unsupported characters" ;;
    *//*|*/./*|*/../*|*/.|*/..) die "$label path must be canonical" ;;
  esac
}

register_project() {
  project="$1"
  manifest="${2:-/opt/$project/current/muster.yaml}"
  lock="${3:-/opt/$project/current/muster.lock.json}"
  validate_project "$project"
  validate_registry_path "$project" "$manifest" manifest
  validate_registry_path "$project" "$lock" lock
  mkdir -p "$REGISTRY_DIR"
  destination="$REGISTRY_DIR/$project.json"
  if [ -e "$destination" ] || [ -L "$destination" ]; then
    [ -f "$destination" ] && [ ! -L "$destination" ] || die "refusing to replace foreign registry entry $destination"
  fi
  old_umask=$(umask)
  umask 077
  temporary=$(mktemp "$REGISTRY_DIR/.${project}.json.new.XXXXXX") || {
    umask "$old_umask"
    die "could not create registry staging file"
  }
  umask "$old_umask"
  if ! {
    printf '{\n'
    printf '  "schema": 1,\n'
    printf '  "id": "implementation:%s",\n' "$(json_escape "$project")"
    printf '  "manifest": "%s",\n' "$(json_escape "$manifest")"
    printf '  "lock": "%s"\n' "$(json_escape "$lock")"
    printf '}\n'
  } > "$temporary"; then
    rm -f "$temporary"
    die "could not write registry staging file"
  fi
  if ! chmod 0644 "$temporary" || ! mv -f "$temporary" "$destination"; then
    rm -f "$temporary"
    die "could not publish registry entry $destination"
  fi
}

unregister_project() {
  project="$1"
  validate_project "$project"
  rm -f "$REGISTRY_DIR/$project.json"
}

case "${1:-}" in
  ensure)
    ensure_core
    ;;
  register)
    [ "$#" -ge 2 ] && [ "$#" -le 4 ] || die "usage: muster-bootstrap.sh register PROJECT [MANIFEST [LOCK]]"
    register_project "$2" "${3:-}" "${4:-}"
    ;;
  unregister)
    [ "$#" -eq 2 ] || die "usage: muster-bootstrap.sh unregister PROJECT"
    unregister_project "$2"
    ;;
  *)
    die "usage: muster-bootstrap.sh {ensure|register|unregister}"
    ;;
esac
