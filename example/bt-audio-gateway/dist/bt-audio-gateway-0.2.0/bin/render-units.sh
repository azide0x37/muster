#!/bin/sh
set -eu

PROJECT="bt-audio-gateway"
OUT_DIR="${1:-dist/rendered-systemd}"

mkdir -p "$OUT_DIR"
cp systemd/*.service systemd/*.timer "$OUT_DIR/"
printf 'Rendered %s systemd units into %s\n' "$PROJECT" "$OUT_DIR"
