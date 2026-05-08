#!/bin/sh
set -eu

OUT_DIR="${1:-dist/rendered-systemd}"
mkdir -p "$OUT_DIR"
cp systemd/*.service systemd/*.timer "$OUT_DIR/"
cp udev/*.rules "$OUT_DIR/"
printf 'Rendered dvd-ingester units and udev rules into %s\n' "$OUT_DIR"
