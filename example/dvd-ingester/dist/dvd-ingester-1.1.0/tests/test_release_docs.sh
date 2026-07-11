#!/bin/sh
set -eu

VERSION="$(cat VERSION)"

./bin/generate-changelog.sh --check

grep -q "Current release: ${VERSION}" README.md
grep -q "## ${VERSION}" RELEASE.md
grep -q "## ${VERSION}" CHANGELOG.md
grep -q "Release Documentation Cycle" README.md
grep -q "make changelog" README.md
