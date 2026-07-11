#!/bin/sh
# Project-local release transaction primitives shared by install.sh and update.sh.
# This file is intentionally identical across Muster examples.

RELEASE_LOCK=""
RELEASE_LOCK_TOKEN=""
RELEASE_STAGE=""
HAD_CURRENT=0
PREVIOUS_TARGET=""
HAD_REGISTRATION=0

release_die() {
  printf '%s: %s\n' "$PROJECT" "$*" >&2 || true
  exit 1
}

release_json_value() {
  key="$1"
  file="$2"
  sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n 1
}

release_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

release_version_valid() {
  printf '%s\n' "$1" | awk '
    /^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?(\+[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$/ { found = 1 }
    END { exit found ? 0 : 1 }
  '
}

release_require_version() {
  release_version_valid "$1" || release_die "invalid release version: $1"
}

release_sha_valid() {
  [ "${#1}" -eq 64 ] || return 1
  case "$1" in
    *[!0-9a-f]*) return 1 ;;
    *) return 0 ;;
  esac
}

release_require_sha() {
  release_sha_valid "$1" || release_die "invalid SHA256: expected 64 lowercase hexadecimal characters"
}

release_yaml_project() {
  awk '
    $0 == "project:" { in_project = 1; next }
    in_project && /^  name:[[:space:]]+/ {
      sub(/^  name:[[:space:]]+/, "")
      print
      exit
    }
    in_project && /^[^[:space:]]/ { exit }
  ' "$1"
}

release_dir_valid() {
  directory="$1"
  expected_version="$2"
  [ -d "$directory" ] && [ ! -L "$directory" ] || return 1
  for required in VERSION muster.yaml muster.lock.json bin/muster-bootstrap.sh bin/doctor.sh; do
    [ -f "$directory/$required" ] && [ ! -L "$directory/$required" ] || return 1
  done
  [ "$(cat "$directory/VERSION")" = "$expected_version" ] || return 1
  [ "$(release_yaml_project "$directory/muster.yaml")" = "$PROJECT" ] || return 1
  [ "$(release_json_value schema "$directory/muster.lock.json")" = "muster.lock/v1" ] || return 1
  locked_version=$(release_json_value version "$directory/muster.lock.json")
  [ "$locked_version" = "$expected_version" ] || return 1
  locked_digest=$(release_json_value manifest_sha256 "$directory/muster.lock.json")
  release_sha_valid "$locked_digest" || return 1
  [ "$(release_sha256 "$directory/muster.yaml")" = "$locked_digest" ] || return 1
  grep -q "\"id\"[[:space:]]*:[[:space:]]*\"implementation:$PROJECT\"" "$directory/muster.lock.json" || return 1
}

release_require_dir() {
  release_dir_valid "$1" "$2" || release_die "release $2 does not contain a valid $PROJECT VERSION, manifest, and lock"
}

release_acquire_lock() {
  lock_timeout="${MUSTER_RELEASE_LOCK_TIMEOUT_SECONDS:-120}"
  case "$lock_timeout" in ''|*[!0-9]*|0) release_die "invalid release lock timeout: $lock_timeout" ;; esac
  max_attempts=$((lock_timeout * 10))
  lock_base="$ROOT/var/lock/muster"
  mkdir -p "$lock_base"
  RELEASE_LOCK="$lock_base/$PROJECT.release.lock"
  attempts=0
  while ! mkdir "$RELEASE_LOCK" 2>/dev/null; do
    if [ -L "$RELEASE_LOCK" ]; then
      release_die "refusing symlink release lock $RELEASE_LOCK"
    fi
    owner_pid=""
    owner_project=""
    owner_started=""
    if [ -f "$RELEASE_LOCK/owner" ] && [ ! -L "$RELEASE_LOCK/owner" ]; then
      owner_pid=$(sed -n 's/^pid=//p' "$RELEASE_LOCK/owner" | head -n 1)
      owner_project=$(sed -n 's/^project=//p' "$RELEASE_LOCK/owner" | head -n 1)
      owner_started=$(sed -n 's/^started=//p' "$RELEASE_LOCK/owner" | head -n 1)
    fi
    owner_metadata_valid=1
    case "$owner_pid" in ''|*[!0-9]*) owner_metadata_valid=0 ;; esac
    case "$owner_started" in ''|*[!0-9]*) owner_metadata_valid=0 ;; esac
    if [ "${#owner_pid}" -gt 10 ] || [ "${#owner_started}" -gt 10 ]; then
      owner_metadata_valid=0
    fi
    if [ "$owner_metadata_valid" = "1" ]; then
      now=$(date +%s)
      age=$((now - owner_started))
      if [ "$owner_project" = "$PROJECT" ] && [ "$owner_pid" -gt 1 ] && [ "$age" -ge 5 ] && ! kill -0 "$owner_pid" 2>/dev/null; then
          stale="$lock_base/$PROJECT.release.lock.stale.$$"
          if [ ! -e "$stale" ] && [ ! -L "$stale" ] && mv "$RELEASE_LOCK" "$stale" 2>/dev/null; then
            rm -rf "$stale"
            continue
          fi
      fi
    fi
    attempts=$((attempts + 1))
    if [ "$attempts" -ge "$max_attempts" ]; then
      release_die "timed out waiting for the $PROJECT release lock"
    fi
    sleep 0.1
  done
  chmod 0700 "$RELEASE_LOCK"
  started=$(date +%s)
  RELEASE_LOCK_TOKEN="$$:$started"
  {
    printf 'project=%s\n' "$PROJECT"
    printf 'pid=%s\n' "$$"
    printf 'started=%s\n' "$started"
    printf 'token=%s\n' "$RELEASE_LOCK_TOKEN"
  } > "$RELEASE_LOCK/owner.new"
  chmod 0600 "$RELEASE_LOCK/owner.new"
  mv -f "$RELEASE_LOCK/owner.new" "$RELEASE_LOCK/owner"
}

release_unlock() {
  if [ -n "$RELEASE_LOCK" ]; then
    owner_token=""
    if [ -f "$RELEASE_LOCK/owner" ] && [ ! -L "$RELEASE_LOCK/owner" ]; then
      owner_token=$(sed -n 's/^token=//p' "$RELEASE_LOCK/owner" | head -n 1)
    fi
    if [ -n "$RELEASE_LOCK_TOKEN" ] && [ "$owner_token" = "$RELEASE_LOCK_TOKEN" ]; then
      rm -f "$RELEASE_LOCK/owner"
      rmdir "$RELEASE_LOCK" 2>/dev/null || true
    fi
    RELEASE_LOCK=""
    RELEASE_LOCK_TOKEN=""
  fi
}

release_new_stage() {
  version="$1"
  mkdir -p "$RELEASES_DIR"
  old_umask=$(umask)
  umask 077
  RELEASE_STAGE=$(mktemp -d "$RELEASES_DIR/.${version}.stage.XXXXXX") || {
    umask "$old_umask"
    release_die "could not create private release staging"
  }
  umask "$old_umask"
}

release_validate_archive() {
  archive="$1"
  version="$2"
  members="$TMP_DIR/archive.members"
  listing="$TMP_DIR/archive.listing"
  prefix="$PROJECT-$version"

  tar -tzf "$archive" > "$members" || release_die "cannot list release archive"
  [ -s "$members" ] || release_die "release archive is empty"
  while IFS= read -r member || [ -n "$member" ]; do
    case "$member" in
      ''|/*|../*|*/../*|*/..|*\\*) release_die "unsafe archive member: $member" ;;
    esac
    case "$member" in
      "$prefix"|"$prefix/"|"$prefix/"*) ;;
      *) release_die "archive member is outside $prefix: $member" ;;
    esac
  done < "$members"

  tar -tvzf "$archive" > "$listing" || release_die "cannot inspect release archive entries"
  if awk 'substr($0, 1, 1) != "-" && substr($0, 1, 1) != "d" { exit 1 }' "$listing"; then
    :
  else
    release_die "release archive contains a link or non-regular entry"
  fi
}

release_extract_archive() {
  archive="$1"
  version="$2"
  release_validate_archive "$archive" "$version"
  release_new_stage "$version"
  if ! tar -xzf "$archive" -C "$RELEASE_STAGE" --strip-components=1; then
    release_die "could not extract release archive"
  fi
}

release_publish_stage() {
  version="$1"
  destination="$RELEASES_DIR/$version"
  project_release_valid "$RELEASE_STAGE" "$version" || release_die "staged $PROJECT release $version is incomplete"

  if [ -e "$destination" ] || [ -L "$destination" ]; then
    project_release_valid "$destination" "$version" || release_die "refusing to replace invalid existing release $destination"
    rm -rf "$RELEASE_STAGE"
    RELEASE_STAGE=""
    return 0
  fi

  find "$RELEASE_STAGE" -type d -exec chmod 0755 {} \;
  find "$RELEASE_STAGE" -type f -exec chmod a-w {} \;
  mv "$RELEASE_STAGE" "$destination" || release_die "could not atomically publish release $version"
  RELEASE_STAGE=""
  project_release_valid "$destination" "$version" || release_die "published release $version failed validation"
}

release_snapshot_state() {
  mkdir -p "$TMP_DIR"
  if [ -L "$CURRENT_LINK" ]; then
    PREVIOUS_TARGET=$(readlink "$CURRENT_LINK")
    case "$PREVIOUS_TARGET" in
      releases/*)
        previous_version=${PREVIOUS_TARGET#releases/}
        release_require_version "$previous_version"
        [ "$PREVIOUS_TARGET" = "releases/$previous_version" ] || release_die "invalid current release target: $PREVIOUS_TARGET"
        ;;
      *) release_die "refusing unmanaged current target: $PREVIOUS_TARGET" ;;
    esac
    HAD_CURRENT=1
  elif [ -e "$CURRENT_LINK" ]; then
    release_die "refusing to replace non-symlink current path $CURRENT_LINK"
  fi

  if [ -L "$REGISTRATION" ]; then
    release_die "refusing symlink registration $REGISTRATION"
  elif [ -f "$REGISTRATION" ]; then
    cp -p "$REGISTRATION" "$TMP_DIR/registration.previous"
    HAD_REGISTRATION=1
  elif [ -e "$REGISTRATION" ]; then
    release_die "refusing non-file registration $REGISTRATION"
  fi
}

release_atomic_link() {
  target="$1"
  destination="$2"
  temporary="$(dirname "$destination")/.${PROJECT}.link.$$"
  rm -f "$temporary"
  ln -s "$target" "$temporary" || release_die "could not create temporary release link"
  if mv -f -T "$temporary" "$destination" 2>/dev/null; then
    return 0
  fi
  if mv -f -h "$temporary" "$destination" 2>/dev/null; then
    return 0
  fi
  rm -f "$temporary"
  release_die "could not atomically switch $destination"
}

release_atomic_remove() {
  destination="$1"
  [ -e "$destination" ] || [ -L "$destination" ] || return 0
  discarded="$(dirname "$destination")/.${PROJECT}.discarded.$$"
  rm -f "$discarded"
  mv "$destination" "$discarded" || return 1
  rm -f "$discarded"
}

release_switch_current() {
  version="$1"
  release_atomic_link "releases/$version" "$CURRENT_LINK"
}

release_restore_state() {
  if [ "$HAD_CURRENT" = "1" ]; then
    release_atomic_link "$PREVIOUS_TARGET" "$CURRENT_LINK"
  else
    release_atomic_remove "$CURRENT_LINK" || true
  fi

  if [ "$HAD_REGISTRATION" = "1" ]; then
    mkdir -p "$(dirname "$REGISTRATION")"
    temporary="$(dirname "$REGISTRATION")/.${PROJECT}.registration.$$"
    rm -f "$temporary"
    cp -p "$TMP_DIR/registration.previous" "$temporary"
    mv -f "$temporary" "$REGISTRATION"
  else
    release_atomic_remove "$REGISTRATION" || true
  fi
}

managed_filename_valid() {
  case "$1" in
    ''|*[!A-Za-z0-9@_.:-]*) return 1 ;;
    *) return 0 ;;
  esac
}

managed_snapshot() {
  name="$1"
  destination_dir="$2"
  new_dir="$3"
  old_dir="$4"
  state="$TMP_DIR/managed-$name"
  mkdir -p "$state/present" "$state/absent"
  : > "$state/names.unsorted"

  for source_dir in "$old_dir" "$new_dir"; do
    [ -d "$source_dir" ] || continue
    for source in "$source_dir"/*; do
      [ -e "$source" ] || [ -L "$source" ] || continue
      [ -f "$source" ] && [ ! -L "$source" ] || release_die "managed source must be a regular file: $source"
      filename=$(basename "$source")
      managed_filename_valid "$filename" || release_die "invalid managed filename: $filename"
      printf '%s\n' "$filename" >> "$state/names.unsorted"
    done
  done
  sort -u "$state/names.unsorted" > "$state/names"

  mkdir -p "$destination_dir"
  while IFS= read -r filename || [ -n "$filename" ]; do
    [ -n "$filename" ] || continue
    destination="$destination_dir/$filename"
    if [ -L "$destination" ]; then
      release_die "refusing managed-file symlink $destination"
    elif [ -f "$destination" ]; then
      cp -p "$destination" "$state/present/$filename"
    elif [ -e "$destination" ]; then
      release_die "refusing non-file managed path $destination"
    else
      : > "$state/absent/$filename"
    fi
  done < "$state/names"
}

managed_atomic_copy() {
  source="$1"
  destination="$2"
  mode="${3:-}"
  temporary="$(dirname "$destination")/.$(basename "$destination").new.$$"
  rm -f "$temporary"
  cp -p "$source" "$temporary"
  if [ -n "$mode" ]; then
    chmod "$mode" "$temporary"
  fi
  mv -f "$temporary" "$destination"
}

managed_apply() {
  name="$1"
  destination_dir="$2"
  new_dir="$3"
  state="$TMP_DIR/managed-$name"
  while IFS= read -r filename || [ -n "$filename" ]; do
    [ -n "$filename" ] || continue
    source="$new_dir/$filename"
    destination="$destination_dir/$filename"
    if [ -f "$source" ] && [ ! -L "$source" ]; then
      if [ -f "$destination" ] && cmp -s "$source" "$destination"; then
        continue
      fi
      managed_atomic_copy "$source" "$destination" 0644
    else
      release_atomic_remove "$destination" || release_die "could not remove obsolete managed file $destination"
    fi
  done < "$state/names"
}

managed_restore() {
  name="$1"
  destination_dir="$2"
  state="$TMP_DIR/managed-$name"
  [ -f "$state/names" ] || return 0
  while IFS= read -r filename || [ -n "$filename" ]; do
    [ -n "$filename" ] || continue
    destination="$destination_dir/$filename"
    if [ -f "$state/present/$filename" ]; then
      managed_atomic_copy "$state/present/$filename" "$destination" || true
    else
      release_atomic_remove "$destination" || true
    fi
  done < "$state/names"
}
MUSTER_STANDALONE=1
set -eu

PROJECT="bt-audio-gateway"
ROOT="${MUSTER_ROOT:-}"
CONFIG_DIR="$ROOT/etc/$PROJECT"
CONFIG_FILE="$CONFIG_DIR/$PROJECT.env"
INSTALL_DIR="$ROOT/opt/$PROJECT"
CURRENT_LINK="$INSTALL_DIR/current"
RELEASES_DIR="$INSTALL_DIR/releases"
SYSTEMD_DIR="$ROOT/etc/systemd/system"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SRC_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DEFAULT_MANIFEST_URL="https://github.com/azide0x37/bt-audio-gateway/releases/latest/download/manifest.json"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_DIR=""
TMP_CREATED=0
REQUESTED_VERSION="${MUSTER_VERSION:-}"
VERSION=""
RELEASE_DIR=""
REGISTRATION="$ROOT/etc/muster/implementations.d/$PROJECT.json"
TRANSACTION_ACTIVE=0
AUDIO_USER_VALUE=""
MIGRATED_AUDIO_USER_FROM=""


log() {
  printf '%s\n' "$*" || true
}

create_private_tmp() {
  old_umask=$(umask)
  umask 077
  TMP_DIR=$(mktemp -d "$TMP_PARENT/$PROJECT-install.XXXXXX") || {
    umask "$old_umask"
    release_die "could not create private install workspace"
  }
  umask "$old_umask"
  TMP_CREATED=1
}

reload_managers() {
  if [ -z "$ROOT" ] && command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
  fi
}

rollback_transaction() {
  [ "$TRANSACTION_ACTIVE" = "1" ] || return 0
  TRANSACTION_ACTIVE=0
  managed_restore systemd "$SYSTEMD_DIR"
  release_restore_state
  reload_managers
}

cleanup() {
  status=$?
  trap - EXIT INT TERM
  rollback_transaction || true
  if [ -n "$RELEASE_STAGE" ]; then
    rm -rf "$RELEASE_STAGE"
    RELEASE_STAGE=""
  fi
  if [ "$TMP_CREATED" = "1" ] && [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
    TMP_CREATED=0
  fi
  release_unlock
  exit "$status"
}
trap cleanup EXIT INT TERM

need_root() {
  if [ -z "$ROOT" ] && [ "$(id -u)" -ne 0 ]; then
    release_die "install.sh must run as root; use sudo or set MUSTER_ROOT for a staged install"
  fi
}

install_packages() {
  if [ "${MUSTER_SKIP_PACKAGES:-0}" = "1" ]; then
    log "Skipping package install because MUSTER_SKIP_PACKAGES=1"
    return 0
  fi
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update
    apt-get install -y \
      bluez \
      pipewire pipewire-pulse wireplumber libspa-0.2-bluetooth \
      pulseaudio-utils \
      snapclient \
      curl ca-certificates
  else
    log "apt-get not found; skipping package install"
  fi
}

project_release_valid() {
  directory="$1"
  version="$2"
  release_dir_valid "$directory" "$version" || return 1
  for required in \
    bin/release-transaction.sh bin/bt-audio-watch bin/bt-audio-route \
    systemd/bt-audio-watch.service systemd/bt-audio-doctor.timer \
    etc/bt-audio-gateway.env.example; do
    [ -f "$directory/$required" ] && [ ! -L "$directory/$required" ] || return 1
  done
  [ -x "$directory/bin/muster-bootstrap.sh" ] && [ -x "$directory/bin/doctor.sh" ] && \
    [ -x "$directory/bin/bt-audio-watch" ] && [ -x "$directory/bin/bt-audio-route" ]
}

stage_checkout() {
  release_new_stage "$VERSION"
  mkdir -p "$RELEASE_STAGE/bin" "$RELEASE_STAGE/systemd" "$RELEASE_STAGE/etc" "$RELEASE_STAGE/doc"
  cp "$SRC_ROOT"/bin/*.sh "$RELEASE_STAGE/bin/"
  cp "$SRC_ROOT/src/bt-audio-watch" "$SRC_ROOT/src/bt-audio-route" "$RELEASE_STAGE/bin/"
  cp "$SRC_ROOT"/systemd/* "$RELEASE_STAGE/systemd/"
  cp "$SRC_ROOT"/etc/* "$RELEASE_STAGE/etc/"
  cp "$SRC_ROOT/muster.yaml" "$SRC_ROOT/muster.lock.json" "$SRC_ROOT/VERSION" "$RELEASE_STAGE/"
  for document in README.md MUSTER.md RELEASE.md SECURITY.md CHANGELOG.md; do
    [ -f "$SRC_ROOT/$document" ] && cp "$SRC_ROOT/$document" "$RELEASE_STAGE/doc/"
  done
  chmod 0755 "$RELEASE_STAGE"/bin/*.sh "$RELEASE_STAGE/bin/bt-audio-watch" "$RELEASE_STAGE/bin/bt-audio-route"
}

prepare_release() {
  if [ "${MUSTER_STANDALONE:-0}" != "1" ] && [ -f "$SRC_ROOT/muster.yaml" ] && [ -f "$SRC_ROOT/src/bt-audio-watch" ]; then
    embedded_version=$(cat "$SRC_ROOT/VERSION")
    release_require_version "$embedded_version"
    if [ -n "$REQUESTED_VERSION" ] && [ "$REQUESTED_VERSION" != "$embedded_version" ]; then
      release_die "requested version $REQUESTED_VERSION does not match checkout VERSION $embedded_version"
    fi
    VERSION="$embedded_version"
    stage_checkout
  else
    manifest_url="${INSTALL_MANIFEST_URL:-$DEFAULT_MANIFEST_URL}"
    case "$manifest_url" in ''|*'<'*|*'>'*) release_die "INSTALL_MANIFEST_URL is not configured" ;; esac
    curl -fsSL "$manifest_url" -o "$TMP_DIR/manifest.json" || release_die "could not fetch release manifest"
    manifest_project=$(release_json_value project "$TMP_DIR/manifest.json")
    VERSION=$(release_json_value version "$TMP_DIR/manifest.json")
    artifact_url=$(release_json_value artifact_url "$TMP_DIR/manifest.json")
    artifact_sha=$(release_json_value sha256 "$TMP_DIR/manifest.json")
    [ "$manifest_project" = "$PROJECT" ] || release_die "release manifest project $manifest_project does not match $PROJECT"
    release_require_version "$VERSION"
    release_require_sha "$artifact_sha"
    [ -n "$artifact_url" ] || release_die "release manifest has no artifact_url"
    if [ -n "$REQUESTED_VERSION" ] && [ "$REQUESTED_VERSION" != "$VERSION" ]; then
      release_die "requested version $REQUESTED_VERSION does not match manifest version $VERSION"
    fi
    curl -fsSL "$artifact_url" -o "$TMP_DIR/artifact.tar.gz" || release_die "could not fetch release artifact"
    [ "$(release_sha256 "$TMP_DIR/artifact.tar.gz")" = "$artifact_sha" ] || release_die "downloaded artifact SHA256 mismatch"
    release_extract_archive "$TMP_DIR/artifact.tar.gz" "$VERSION"
  fi

  project_release_valid "$RELEASE_STAGE" "$VERSION" || release_die "staged release $VERSION failed project validation"
  release_publish_stage "$VERSION"
  RELEASE_DIR="$RELEASES_DIR/$VERSION"
}

install_config() {
  mkdir -p "$CONFIG_DIR"
  if [ -f "$CONFIG_FILE" ]; then
    log "Preserving existing $CONFIG_FILE"
    return 0
  fi
  cp "$RELEASE_DIR/etc/$PROJECT.env.example" "$CONFIG_FILE"
  chmod 0644 "$CONFIG_FILE"
  log "Installed example config at $CONFIG_FILE"
}

audio_user_exists() {
  candidate="$1"
  [ -n "$candidate" ] && [ "$candidate" != "root" ] && id -u "$candidate" >/dev/null 2>&1
}

replacement_audio_user() {
  if [ -n "${MUSTER_AUDIO_USER:-}" ]; then
    printf '%s\n' "$MUSTER_AUDIO_USER"
    return 0
  fi
  if audio_user_exists "${SUDO_USER:-}"; then
    printf '%s\n' "$SUDO_USER"
    return 0
  fi
  if [ -n "$ROOT" ]; then
    staged_user=$(id -un 2>/dev/null || true)
    if audio_user_exists "$staged_user"; then
      printf '%s\n' "$staged_user"
      return 0
    fi
  fi
  return 1
}

write_audio_user() {
  replacement="$1"
  case "$replacement" in
    ''|*[!A-Za-z0-9_.-]*) release_die "invalid audio user name: $replacement" ;;
  esac
  audio_user_exists "$replacement" || release_die "audio user does not exist or is not allowed: $replacement"
  config_stage=$(mktemp "$CONFIG_DIR/.audio-user.XXXXXX") || release_die "could not stage audio user configuration"
  if ! awk -v user="$replacement" '
    BEGIN { replaced = 0 }
    /^AUDIO_USER=/ { print "AUDIO_USER=" user; replaced = 1; next }
    { print }
    END { if (!replaced) print "AUDIO_USER=" user }
  ' "$CONFIG_FILE" > "$config_stage"; then
    rm -f "$config_stage"
    release_die "could not write audio user configuration"
  fi
  chmod 0644 "$config_stage"
  mv -f "$config_stage" "$CONFIG_FILE"
}

resolve_audio_user() {
  [ -f "$CONFIG_FILE" ] || release_die "missing configuration: $CONFIG_FILE"
  # shellcheck disable=SC1090
  . "$CONFIG_FILE"
  configured="${AUDIO_USER:-}"

  if [ -n "${MUSTER_AUDIO_USER:-}" ]; then
    replacement=$(replacement_audio_user) || release_die "MUSTER_AUDIO_USER must name an existing non-root user"
    audio_user_exists "$replacement" || release_die "MUSTER_AUDIO_USER must name an existing non-root user"
    if [ "$configured" != "$replacement" ]; then
      MIGRATED_AUDIO_USER_FROM="$configured"
      write_audio_user "$replacement"
      log "Configured audio user $replacement from MUSTER_AUDIO_USER"
    fi
    AUDIO_USER_VALUE="$replacement"
    return 0
  fi

  if audio_user_exists "$configured"; then
    AUDIO_USER_VALUE="$configured"
    return 0
  fi

  replacement=$(replacement_audio_user) || release_die "audio user $configured does not exist; rerun with MUSTER_AUDIO_USER=<existing-user>"
  if [ "$configured" != "pi" ]; then
    release_die "audio user $configured does not exist; refusing to rewrite an intentional setting without MUSTER_AUDIO_USER"
  fi
  MIGRATED_AUDIO_USER_FROM="$configured"
  write_audio_user "$replacement"
  AUDIO_USER_VALUE="$replacement"
  log "Migrated legacy audio user $configured to $replacement"
}

validate_registration() {
  if [ -n "$ROOT" ]; then
    "$ROOT/usr/local/bin/muster" --root "$ROOT" validate
  else
    /usr/local/bin/muster validate
  fi
}

activate_release() {
  old_release="$TMP_DIR/no-old"
  if [ "$HAD_CURRENT" = "1" ]; then
    old_release="$INSTALL_DIR/$PREVIOUS_TARGET"
  fi
  managed_snapshot systemd "$SYSTEMD_DIR" "$RELEASE_DIR/systemd" "$old_release/systemd"
  TRANSACTION_ACTIVE=1
  managed_apply systemd "$SYSTEMD_DIR" "$RELEASE_DIR/systemd"
  release_switch_current "$VERSION"
  if ! "$RELEASE_DIR/bin/muster-bootstrap.sh" register "$PROJECT"; then
    rollback_transaction
    release_die "Muster registration failed; restored the prior release transaction"
  fi
  if ! validate_registration; then
    rollback_transaction
    release_die "Muster graph validation failed; restored the prior release transaction"
  fi
  TRANSACTION_ACTIVE=0
}

enable_audio_user() {
  [ -z "$ROOT" ] || return 0
  uid_num=$(id -u "$AUDIO_USER_VALUE")
  loginctl enable-linger "$AUDIO_USER_VALUE" || release_die "could not enable linger for audio user $AUDIO_USER_VALUE"
  if ! sudo -u "$AUDIO_USER_VALUE" XDG_RUNTIME_DIR="/run/user/$uid_num" \
    systemctl --user enable --now pipewire pipewire-pulse wireplumber; then
    release_die "could not enable PipeWire services for audio user $AUDIO_USER_VALUE"
  fi
}

enable_systemd() {
  [ -z "$ROOT" ] || return 0
  command -v systemctl >/dev/null 2>&1 || return 0
  systemctl daemon-reload
  systemctl enable --now bt-audio-watch.service
  systemctl enable --now bt-audio-doctor.timer bt-audio-update.timer
  if [ -n "$MIGRATED_AUDIO_USER_FROM" ] && [ "$MIGRATED_AUDIO_USER_FROM" != "$AUDIO_USER_VALUE" ]; then
    systemctl disable --now "snapclient-bt@$MIGRATED_AUDIO_USER_FROM.service" >/dev/null 2>&1 || true
  fi
  if ! systemctl enable --now "snapclient-bt@$AUDIO_USER_VALUE.service"; then
    release_die "snapclient-bt@$AUDIO_USER_VALUE.service failed to start; inspect it with systemctl status and journalctl"
  fi
}

need_root
create_private_tmp
release_acquire_lock
prepare_release
install_packages
install_config
resolve_audio_user
"$RELEASE_DIR/bin/muster-bootstrap.sh" ensure
release_snapshot_state
activate_release
enable_audio_user
enable_systemd
log "$PROJECT $VERSION installed"
