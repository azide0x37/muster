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

for project in bt-audio-gateway dvd-ingester; do
  project_root="$REPO_ROOT/example/$project"
  version=$(cat "$project_root/VERSION")
  dist="$project_root/dist"
  archive="$dist/$project-$version.tar.gz"
  installer="$dist/install.sh"
  expected=$(cat "$archive.sha256")
  actual=$(sha256_file "$archive")

  test "$actual" = "$expected"
  grep -q '"project": "'"$project"'"' "$dist/manifest.json"
  grep -q '"version": "'"$version"'"' "$dist/manifest.json"
  grep -q '"sha256": "'"$actual"'"' "$dist/manifest.json"
  sh -n "$installer"
  grep -q '^MUSTER_STANDALONE=1$' "$installer"
  grep -q '^release_validate_archive()' "$installer"
  if grep -q 'release-transaction.sh"' "$installer"; then
    echo "$installer still depends on an adjacent release helper" >&2
    exit 1
  fi
  if tar -tzf "$archive" | grep -Eq '(^/|(^|/)\.\.(/|$))'; then
    echo "unsafe member in $archive" >&2
    exit 1
  fi
  if tar -tvzf "$archive" | awk 'substr($0, 1, 1) != "-" && substr($0, 1, 1) != "d" { exit 1 }'; then
    :
  else
    echo "linked or special member in $archive" >&2
    exit 1
  fi

  fixture=$(mktemp -d)
  root="$fixture/root"
  cat > "$fixture/manifest.json" <<EOF
{
  "project": "$project",
  "version": "$version",
  "artifact_url": "file://$archive",
  "sha256": "$actual"
}
EOF
  MUSTER_ROOT="$root" \
    MUSTER_SKIP_PACKAGES=1 \
    MUSTER_CLI_SOURCE="$CLI_SOURCE" \
    MUSTER_CLI_VERSION="$CLI_VERSION" \
    INSTALL_MANIFEST_URL="file://$fixture/manifest.json" \
    "$installer" >/dev/null

  test "$(readlink "$root/opt/$project/current")" = "releases/$version"
  test -f "$root/etc/muster/implementations.d/$project.json"
  "$root/usr/local/bin/muster" --root "$root" validate >/dev/null
  MUSTER_ROOT="$root" "$root/opt/$project/current/bin/uninstall.sh" >/dev/null
  test ! -e "$root/etc/muster/implementations.d/$project.json"
  test -x "$root/opt/muster/current/bin/muster"
  rm -rf "$fixture"
done

printf '%s\n' "ok: example release packages and standalone installers"
