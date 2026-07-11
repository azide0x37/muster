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
