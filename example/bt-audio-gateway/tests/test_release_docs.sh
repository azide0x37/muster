#!/bin/sh
set -eu

VERSION=$(cat VERSION)
./bin/generate-changelog.sh --check
grep -q "## $VERSION" RELEASE.md
grep -q "## $VERSION" CHANGELOG.md
