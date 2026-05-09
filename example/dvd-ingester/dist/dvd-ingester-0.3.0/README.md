# Muster Example: dvd-ingester

A Muster reference service repo for a Raspberry Pi DVD ingest appliance.

Architecture:

```text
DVD inserted
  -> udev marks the device for systemd
  -> dvd-rip@.service runs one shot
  -> dvd-rip-one fingerprints before work
  -> local work directory receives rip output
  -> final file or archive is published atomically to NAS
  -> success record is appended locally
  -> tray ejects
```

This appliance is for lawful, non-copy-protected discs or discs your tools can read without bypassing protection. The Pi is an ingest appliance, not a media workstation. Plex, Jellyfin, NAS jobs, or a desktop system should handle later transcoding when needed.

## Install

Preferred release install:

```sh
curl -fsSL https://github.com/azide0x37/dvd-ingester/releases/latest/download/install.sh | sudo sh
```

Development install from this example:

```sh
sudo ./bin/install.sh
```

Then edit:

```sh
sudoedit /etc/dvd-ingester/dvd-ingester.env
```

## Config

Config lives at `/etc/dvd-ingester/dvd-ingester.env`.

```sh
DEVICE_GLOB=sr[0-9]*
BASE_DIR=/var/lib/dvd-ingester
WORK_DIR=/var/lib/dvd-ingester/work
LOG_DIR=/var/lib/dvd-ingester/logs
RIPPED_DB=/var/lib/dvd-ingester/ripped.jsonl
DEST_DIR=/mnt/nas/DVD_Rips
RIP_MODE=movie
HANDBRAKE_PRESET='Fast 480p30'
MIN_TV_TITLE_SECONDS=1080
MEDIA_SETTLE_SECONDS=8
MEDIA_READY_TIMEOUT=60
MEDIA_READY_INTERVAL=2
EJECT_ON_SUCCESS=1
EJECT_ON_FAILURE=0
RSYNC_RETRIES=3
RSYNC_RETRY_SLEEP=30
REQUIRE_DEST_MOUNT=1
MIN_WORK_FREE_BYTES=10737418240
PUBLISH_ASYNC=1
PUBLISH_DURING_RIP=1
PUBLISH_SYNC_INTERVAL=60
AUTOUPDATE=1
UPDATE_CHANNEL=latest
UPDATE_MANIFEST_URL='https://github.com/azide0x37/dvd-ingester/releases/latest/download/manifest.json'
```

`RIP_MODE=movie` encodes the main feature to one MKV. `RIP_MODE=archive` uses `dvdbackup -M` locally and atomically publishes the DVD structure. `RIP_MODE=tv` is reserved for a tuned title-duration policy and intentionally fails until implemented for a known library.

## systemd And udev

- `udev/90-dvd-ingester.rules`: matches DVD media and asks systemd to start `dvd-rip@%k.service`.
- `dvd-rip@.service`: one-shot rip service for `/dev/%I`.
- `dvd-ingester-doctor.service`: health checks.
- `dvd-ingester-doctor.timer`: periodic health checks.
- `dvd-ingester-update.service`: release update check.
- `dvd-ingester-update.timer`: periodic autoupdate polling with jitter.

The udev rule does not run the rip. udev triggers systemd because udev kills long-running work.

## MPL Pattern Mapping

`dvd-ingester` is a concrete instance of [`T2R4.device-triggered-conveyor`](https://github.com/azide0x37/muster-pattern-library/blob/main/patterns/t2/rare/T2R4.device-triggered-conveyor/README.md) from the Muster Pattern Library. This example keeps its project-specific names and installer layout, but the design maps to MPL atoms as follows:

| dvd-ingester behavior | MPL atom | Current evidence |
|---|---|---|
| DVD media insertion asks systemd to run one bounded job | `R2.device-binding` and `T2R4.device-triggered-conveyor` | `udev/90-dvd-ingester.rules`, `systemd/dvd-rip@.service` |
| Destination NAS must be mounted before final writes | `R5.capability-mount` | `REQUIRE_DEST_MOUNT`, `DEST_DIR`, `findmnt` checks |
| Local work is staged before atomic NAS publish | `T2C1.hot-cold-nas-conveyor` | `WORK_DIR`, `MIN_WORK_FREE_BYTES`, `dvd-publish-one`, `.part` paths, final rename |
| Periodic health and update work is timer-owned | `C2.persistent-tick`, `T2C3.scheduled-herald` | `dvd-ingester-doctor.timer`, `dvd-ingester-update.timer` |
| Failed or partial work remains inspectable | `C5.failure-ratchet` | `.rip-in-progress`, `.rip-failed`, `.rip-complete`, per-disc logs |
| Successful disc identity is retained locally | dvd-ingester-specific ledger | `ripped.jsonl` |

### MPL Migration Plan

This documentation pass does not change runtime behavior, service names, installer behavior, or release semantics.

1. Declare atoms: keep this README, `MUSTER.md`, and `muster.yaml` aligned with the MPL mapping.
2. Adapter refactor later: wrap the existing DVD scripts with MPL-style states such as `capability_*`, `waiting_for_capacity`, `ready_for_cold_publish`, and explicit handoff metadata.
3. Full atom rebuild later: evaluate replacing bespoke boundaries with MPL helper scripts and units while preserving DVD fingerprinting, lawful-use scope, rip modes, eject behavior, and the existing update/install layout.

Current gaps against the production-beta MPL contract:

- Low work capacity fails immediately through `MIN_WORK_FREE_BYTES`; it does not yet wait for a drain command or record `waiting_for_capacity`.
- Destination mount proof exists, but it does not yet write a named capability state file.
- Async publishing exists, but it is not yet a generic MPL hot/cold handoff contract shared with `T2C1.hot-cold-nas-conveyor`.

## Manual Test

```sh
sudo /opt/dvd-ingester/current/bin/dvd-rip-one /dev/sr0
```

Watch insert-triggered logs:

```sh
journalctl -fu 'dvd-rip@sr0.service'
```

Inspect device properties before tightening matches:

```sh
udevadm info --query=property --name=/dev/sr0
```

Inspect rip and probe logs:

```sh
sudo journalctl -u 'dvd-rip@sr0.service' -n 100 --no-pager
sudo ls -lt /var/lib/dvd-ingester/logs
sudo tail -n 80 /var/lib/dvd-ingester/logs/*.log
```

## Update And Rollback

`dvd-ingester-update.timer` runs `bin/update.sh`. Updates are skipped when `AUTOUPDATE=0`.

The updater fetches a release manifest, downloads the release archive, verifies SHA256, installs it into `/opt/dvd-ingester/releases/<version>`, switches `/opt/dvd-ingester/current`, reloads systemd/udev, and runs `doctor.sh`. If the health check fails, it switches back to the prior release.

## Development

```sh
make test
make package
```

Run a staged idempotence check without touching the host:

```sh
MUSTER_ROOT="$(mktemp -d)" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
MUSTER_ROOT="$MUSTER_ROOT" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
```

## Muster Self-Certification

| Requirement | Status | Evidence |
|---|---:|---|
| systemd owns lifecycle | PASS | `systemd/*.service` |
| udev does not run long work | PASS | `udev/90-dvd-ingester.rules` uses `SYSTEMD_WANTS` |
| timer-based update polling exists | PASS | `systemd/dvd-ingester-update.timer` |
| idempotent installer exists | PASS | `bin/install.sh` and `tests/test_installer_idempotent.sh` |
| config lives in `/etc/<project>/` | PASS | `/etc/dvd-ingester/dvd-ingester.env` |
| versioned install path exists | PASS | `/opt/dvd-ingester/releases/<version>` |
| rollback path exists | PASS | `bin/update.sh` |
| doctor check exists | PASS | `bin/doctor.sh` |
| release artifacts buildable | PASS | `make package` |
| tests runnable | PASS | `make test` |
| MPL pattern mapping documented | PASS | `T2R4.device-triggered-conveyor` section above |
