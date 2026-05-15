# dvd-ingester

`dvd-ingester` is a Muster example repo for a Raspberry Pi OS or Debian box with
a USB optical drive and a mounted media destination.

It implements the Muster Pattern Library
`T2R4.device-triggered-conveyor` pattern: a device event triggers one bounded
ingest job, the job proves cold storage, waits for hot-storage capacity, stages
local work, and hands completed output to a timer-driven hot/cold conveyor.
In this implementation, udev only asks systemd to start the bounded service.

## Architecture

```text
DVD media becomes ready
  -> udev rule adds SYSTEMD_WANTS=dvd-rip@%k.service
  -> systemd runs /opt/dvd-ingester/current/bin/dvd-rip-one /dev/%I --apply
  -> dvd-rip-one proves DEST_DIR and waits for HOT_DIR capacity
  -> completed rip moves to HOT_DIR/<run-id>
  -> dvd-publish-one.timer drains HOT_DIR to DEST_DIR
  -> publish writes DEST_DIR/.incoming-<run-id> and atomically renames final output
```

## MPL Pattern Mapping

| dvd-ingester boundary | MPL vocabulary | Evidence |
| --- | --- | --- |
| Optical drive event only requests systemd | `R2.device-binding`, `C1.service-capsule` | `udev/90-dvd-ingester.rules`, `systemd/dvd-rip@.service` |
| Destination is proven before high-volume writes | `R5.capability-mount`, `C4.lazy-resource-gate` | `src/dvd-rip-one`, `src/dvd-publish-one` |
| Local hot work drains to mounted cold storage | `T2C1.hot-cold-nas-conveyor` | `HOT_DIR`, `.ingest-complete`, `src/dvd-publish-one` |
| Repeated drain, doctor, and update checks | `C2.persistent-tick`, `T2C3.scheduled-herald` | `systemd/*.timer` |
| Degraded and failed states remain inspectable | `C5.failure-ratchet` | JSON state files under `STATE_DIR` |
| Install, update, uninstall, package lifecycle | `C6.lifecycle-capsule` | `bin/*.sh`, `Makefile`, `dist/manifest.json` |

## Install

From a checkout:

```sh
sudo ./bin/install.sh
```

From a published release:

```sh
curl -fsSL https://github.com/azide0x37/dvd-ingester/releases/latest/download/install.sh | sudo sh
```

For staged verification without touching the host:

```sh
MUSTER_ROOT="$(mktemp -d)" MUSTER_SKIP_PACKAGES=1 ./bin/install.sh
```

## Configuration

The installer creates `/etc/dvd-ingester/dvd-ingester.env` from
`etc/dvd-ingester.env.example` and preserves the file on later installs.

| Setting | Default | Purpose |
| --- | --- | --- |
| `DEST_DIR` | `/mnt/media/dvd-ingester` | Mounted cold destination for completed rips |
| `HOT_DIR` | `/var/cache/dvd-ingester/hot` | Local handoff directory drained by the publish timer |
| `WORK_DIR` | `/var/lib/dvd-ingester/work` | Temporary rip workspace |
| `STATE_DIR` | `/run/dvd-ingester` | Runtime JSON state and locks |
| `MIN_HOT_FREE_BYTES` | `10737418240` | Required hot-storage free space before a rip starts |
| `CAPACITY_TIMEOUT_SECONDS` | `900` | Maximum wait for hot capacity |
| `RIP_COMMAND` | empty | Optional override command for real ripping |
| `EJECT_AFTER_RIP` | `1` | Eject after successful hot handoff |
| `ALLOW_UNMOUNTED_DEST` | `0` | Permit local, unmounted `DEST_DIR` only when deliberately set |
| `AUTOUPDATE` | `1` | Enable update timer work |
| `UPDATE_MANIFEST_URL` | release manifest URL | Manifest used by `bin/update.sh` |

`RIP_COMMAND` receives `DEVICE`, `RUN_DIR`, `RUN_ID`, and
`DEVICE_FINGERPRINT` in the environment. If it is empty, apply mode tries
`dvdbackup` first, then `makemkvcon`. Mock mode writes a small payload for
tests.

## systemd Units

| Unit | Purpose |
| --- | --- |
| `dvd-rip@.service` | One bounded rip job for `/dev/%I` |
| `dvd-publish-one.service` | One hot-to-cold publish drain pass |
| `dvd-publish-one.timer` | Periodic publish drain |
| `dvd-ingester-doctor.service` | Health check |
| `dvd-ingester-doctor.timer` | Periodic health check |
| `dvd-ingester-update.service` | Release manifest update check |
| `dvd-ingester-update.timer` | Periodic update polling |

## Operations

Run a doctor check:

```sh
/opt/dvd-ingester/current/bin/doctor.sh
```

Drain hot storage manually:

```sh
sudo systemctl start dvd-publish-one.service
```

Inspect the latest runtime states:

```sh
sudo ls -l /run/dvd-ingester
sudo cat /run/dvd-ingester/rip.json
sudo cat /run/dvd-ingester/publish.json
```

Watch logs:

```sh
journalctl -u 'dvd-rip@*' -u dvd-publish-one.service -f
```

## Update And Rollback

`bin/update.sh` reads `/etc/dvd-ingester/dvd-ingester.env`, exits cleanly when
`AUTOUPDATE=0`, downloads `manifest.json`, verifies the artifact SHA256,
unpacks the new release under `/opt/dvd-ingester/releases/<version>`, switches
`/opt/dvd-ingester/current`, restarts timers, and runs `doctor.sh`.

If the doctor check fails, the updater restores the previous `current` symlink
and restarts the timers again.

## Adjacent Systems

`dvd-ingester` stops at publishing DVD output. Plex, Jellyfin, HandBrake,
library managers, or desktop import tools should watch `DEST_DIR` after the
atomic final directory appears. They should not read `.incoming-*` directories.

## Tests

```sh
make test
make package
```

The test suite uses mock mode for the conveyor flow and staged roots for
installer idempotence. It does not require a DVD drive or a mounted NAS.

## Known Limits

- Title selection is delegated to `RIP_COMMAND` or the installed ripper tool.
- Apply mode expects the operator to configure legal ripping tools for their
  jurisdiction and media.
- The publish drain copies to a destination-side temporary directory before the
  final rename. It is atomic for readers of final output, but interrupted
  copies may leave `.incoming-*` directories for inspection.

## Muster Self-Certification

| Requirement | Status | Evidence |
| --- | --- | --- |
| systemd owns lifecycle | PASS | `systemd/dvd-rip@.service`, publish, doctor, and update units |
| timer-based update polling exists | PASS | `systemd/dvd-ingester-update.timer` |
| timer-based drain/status work exists | PASS | `systemd/dvd-publish-one.timer`, `systemd/dvd-ingester-doctor.timer` |
| config under `/etc/dvd-ingester` | PASS | `bin/install.sh`, `etc/dvd-ingester.env.example` |
| code under `/opt/dvd-ingester/releases/<version>` | PASS | `bin/install.sh` |
| current symlink managed atomically | PASS | `bin/install.sh`, `bin/update.sh` |
| idempotent installer exists | PASS | `tests/test_installer_idempotent.sh` |
| rollback path exists | PASS | `bin/update.sh` restores previous `current` on failed doctor |
| doctor check exists | PASS | `bin/doctor.sh` |
| release artifacts buildable | PASS | `make package` |
| tests runnable | PASS | `make test` |
| systemd units verify when available | PASS | `tests/test_units.sh` runs `systemd-analyze verify` when installed |
| installer preserves config | PASS | staged idempotence test appends and rechecks config |
| generated instructions avoid unmanaged files | PASS | only `/etc/dvd-ingester/dvd-ingester.env` is operator-edited |
| MPL pattern mapping documented | PASS | README, `MUSTER.md`, and `muster.yaml` name `T2R4.device-triggered-conveyor` |
