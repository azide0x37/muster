# Muster Compliance: dvd-ingester

This example applies the root Muster contract to a lawful DVD ingest appliance.

Architecture:

`udev -> systemd one-shot service -> rip script -> local work dir -> atomic NAS publish -> eject`

The udev rule must only request a systemd unit with `SYSTEMD_WANTS`. It must not perform ripping directly.

Configuration lives in `/etc/dvd-ingester/dvd-ingester.env`. Installed code lives in `/opt/dvd-ingester/releases/<version>`, with `/opt/dvd-ingester/current` pointing at the active release.

## MPL Mapping

`dvd-ingester` is documented as a project-specific implementation of [`T2R4.device-triggered-conveyor`](https://github.com/azide0x37/muster-pattern-library/blob/main/patterns/t2/rare/T2R4.device-triggered-conveyor/README.md), using MPL default branch identifiers.

- `R2.device-binding`: `udev/90-dvd-ingester.rules` requests `dvd-rip@%k.service` through `SYSTEMD_WANTS`.
- `R5.capability-mount`: `REQUIRE_DEST_MOUNT`, `DEST_DIR`, and `findmnt` prevent accidental writes to an unmounted local path.
- `T2C1.hot-cold-nas-conveyor`: local `WORK_DIR` staging, `dvd-publish-one`, `.part` outputs, and final rename implement hot-to-cold publishing.
- `C2.persistent-tick` and `T2C3.scheduled-herald`: doctor and update timers own repeated health and release checks.
- `C5.failure-ratchet`: `.rip-*` markers and per-disc logs leave failed work inspectable.

The docs-first migration is intentionally non-runtime. Future runtime work should first add MPL-style state names and handoff metadata around the existing scripts, then evaluate deeper reuse of MPL helper scripts and units.

Known gaps are explicit: low work capacity currently fails immediately instead of waiting for a drain, destination capability proof does not write a named capability state file, and async publishing is not yet a generic MPL hot/cold handoff contract.
