# dvd-ingester

Current release: 1.1.0

`dvd-ingester` is a Muster example repo for a Raspberry Pi OS or Debian box with
a USB optical drive and a mounted media destination.

It implements the Muster Pattern Library
`T2R4.device-triggered-conveyor` pattern: a device event triggers one bounded
ingest job, the job proves cold storage, waits for hot-storage capacity, stages
local work, and hands completed output to a timer-driven hot/cold conveyor.
In this implementation, udev only asks systemd to start the bounded service.
It also implements `T2R6.home-assistant-mqtt-bridge` as an optional local
Home Assistant MQTT discovery, telemetry, and scoped-control bridge.

## Architecture

```text
DVD media becomes ready
  -> udev rule adds SYSTEMD_WANTS=dvd-rip@%k.service
  -> systemd runs /opt/dvd-ingester/current/bin/dvd-rip-one /dev/%I --apply
  -> dvd-rip-one proves DEST_DIR and waits for HOT_DIR capacity
  -> completed rip moves to HOT_DIR/<disc-title>_<fingerprint>
  -> dvd-publish-one.timer drains HOT_DIR to DEST_DIR
  -> publish writes DEST_DIR/.incoming-<disc-title>_<fingerprint> and atomically renames final output
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
| Home Assistant discovery and controls | `T2R6.home-assistant-mqtt-bridge` | `src/dvd-ha-mqtt-bridge`, `src/dvd-control`, `systemd/dvd-ingester-ha-mqtt.*` |

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
`DEVICE_FINGERPRINT` in the environment. Output directories are named
`<disc-title>_<fingerprint>` using the DVD filesystem label and a stable
20-character identity hash. If `RIP_COMMAND` is empty, apply mode tries
`dvdbackup` first, then `makemkvcon`. Mock mode writes a small payload for
tests.

The installer also creates `/etc/dvd-ingester/dvd-ingester.mqtt.env` with mode
`0600`. MQTT is disabled by default:

| Setting | Default | Purpose |
| --- | --- | --- |
| `HA_MQTT_ENABLE` | `0` | Set to `1` to publish with `mosquitto_pub` when available |
| `MQTT_HOST` | `127.0.0.1` | MQTT broker host |
| `MQTT_PORT` | `1883` | MQTT broker port |
| `MQTT_USERNAME` | empty | Optional MQTT username |
| `MQTT_PASSWORD` | empty | Optional MQTT password |
| `MQTT_PUBLISH_TIMEOUT_SECONDS` | `5` | Per-message MQTT publish timeout when `timeout(1)` is available |
| `HA_DISCOVERY_PREFIX` | `homeassistant` | Home Assistant discovery prefix |
| `HA_TOPIC_PREFIX` | `muster/dvd-ingester` | State and command topic prefix |

When MQTT is disabled or no broker tools are installed, the bridge still writes
mockable payloads under `STATE_DIR/ha-mqtt-outbox`.
When `HA_MQTT_ENABLE=1`, `mosquitto_pub` must be installed and able to reach the
configured broker. Publish failures make `dvd-ingester-ha-mqtt.service` fail so
the journal shows a real delivery problem instead of a false success.

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
| `dvd-ingester-ha-mqtt.service` | Publish Home Assistant discovery/state and process scoped controls |
| `dvd-ingester-ha-mqtt.timer` | Periodic Home Assistant bridge refresh |

## Muster Inspector

The installer ensures the independently versioned shared Muster core, registers
this implementation, and validates its schema 2 object graph. If this is the
first Muster implementation on the server, `muster` becomes available in PATH.

```sh
muster status dvd-ingester
muster inspect component:dvd-ingester:state:publish
muster inspect component:dvd-ingester:patterns
muster explain component:dvd-ingester:unit:dvd-publish-one.service
sudo muster doctor dvd-ingester
```

The pattern view is the complete 13-node closure of T2R4 and T2R6 at the MPL
commit recorded in this release—not a live marketing catalog. Existing runtime
JSON is exposed through bounded transitional adapters while doctor results use
the common `muster.observation/v1` envelope.

## Operations

Run a doctor check:

```sh
sudo /opt/dvd-ingester/current/bin/doctor.sh --runtime
sudo /opt/dvd-ingester/current/bin/doctor.sh --runtime --json
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
sudo cat /run/dvd-ingester/ha-mqtt-state.json
```

Refresh Home Assistant state manually:

```sh
sudo systemctl start dvd-ingester-ha-mqtt.service
```

Disable new ingest without stopping the bridge:

```sh
sudo /opt/dvd-ingester/current/bin/dvd-control --apply disable
```

Enable new ingest again:

```sh
sudo /opt/dvd-ingester/current/bin/dvd-control --apply enable
```

Restart owned background services without stopping active rip jobs:

```sh
sudo /opt/dvd-ingester/current/bin/dvd-control --apply restart
```

Inspect the Home Assistant bridge status payload:

```sh
sudo /opt/dvd-ingester/current/bin/dvd-control --apply status
```

Watch logs:

```sh
journalctl -u 'dvd-rip@*' -u dvd-publish-one.service -f
```

## Home Assistant Entities

When `HA_MQTT_ENABLE=1`, the bridge publishes a Home Assistant MQTT device
discovery payload and appliance state. The entity set is intentionally scoped to
operator decisions for this appliance:

| Entity | Purpose |
| --- | --- |
| Availability | Shows whether the bridge can publish current state |
| Health status | Summarizes doctor, rip, publish, and maintenance state |
| Disk state | Reports whether the configured optical device is loaded, busy, missing, or has no media |
| Rip state | Shows active or latest extraction state |
| Extraction progress | Reports current rip bytes against the disc size from metadata as a percentage |
| Publish state | Shows conveyor handoff and cold-destination publish state |
| Conveyance progress | Reports active NAS publish bytes against the hot source size as a percentage |
| Capability and capacity state | Reports destination mount/write health and local hot-cache capacity pressure |
| Local storage | Reports local hot-cache used, free, and total capacity in GiB |
| Destination storage | Reports mounted destination used, free, and total capacity in GiB |
| Folder indexes | Counts work, hot, and completed title directories; bounded directory names are published as MQTT sensor attributes |
| Publish counts | Reports the latest publish drain's published and failed counts |
| Version | Reports the installed `dvd-ingester` version as a diagnostic sensor |
| Restart button | Restarts owned background services without stopping active rip jobs |
| Enabled switch | Blocks or restores new ingest while leaving the bridge online |

Folder index entities intentionally keep counts in the sensor state and put
directory names in `json_attributes_topic` payloads. This keeps Home Assistant
state history small while still exposing the current work queue, hot handoff
queue, and completed title folders for dashboards and automations. The default
attribute list limit is `50` entries and can be changed with
`HA_FOLDER_INDEX_LIMIT`.

## Update And Rollback

`bin/update.sh` reads `/etc/dvd-ingester/dvd-ingester.env` and exits cleanly
when `AUTOUPDATE=0`. Install and update operations share a project-scoped
transaction lock. They require an exact project, semantic version, and
lowercase SHA256; reject archive traversal, links, and unexpected top-level
paths; and validate embedded `VERSION`, `muster.yaml`, and `muster.lock.json`
in private staging. A valid version directory is never rewritten.

The current link, registry entry, managed systemd units, and optical-drive udev
rule switch as one rollback-aware transaction. Graph validation and
`doctor.sh` gate completion. Failure logs captured doctor output, atomically
restores every prior pointer and managed file, reloads managers, and restarts
the prior timers. Logging itself is best-effort.

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

## Release Documentation Cycle

Every push that changes behavior, config, controls, units, packaging, or release
assets must refresh the operator-facing docs before it is considered complete:

1. Update `README.md` so install, config, operations, Home Assistant entities,
   self-certification, and known limits match the implementation.
2. Update `RELEASE.md` with the release-facing change notes.
3. Run `make changelog` to regenerate `CHANGELOG.md` from `RELEASE.md`.
4. Run `make test` and `make package`; both verify that the README and
   generated changelog are current enough to ship.

## Known Limits

- Title selection is delegated to `RIP_COMMAND` or the installed ripper tool.
- Apply mode expects the operator to configure legal ripping tools for their
  jurisdiction and media.
- The publish drain copies to a destination-side temporary directory before the
  final rename. It is atomic for readers of final output, but interrupted
  copies may leave `.incoming-*` directories for inspection.
- MQTT command handling is deliberately narrow. Restart does not stop active
  `dvd-rip@*.service` jobs, and disable leaves the Home Assistant bridge alive
  so it can be re-enabled.

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
| structured doctor evidence exists | PASS | `/run/muster/dvd-ingester/observations/doctor.json` |
| shared Muster command is registered | PASS | `bin/muster-bootstrap.sh`, staged installer test |
| schema 2 object and exact pattern graph are locked | PASS | `muster.yaml`, `muster.lock.json` |
| uninstaller preserves shared core | PASS | root cross-service lifecycle test |
| release artifacts buildable | PASS | `make package` |
| tests runnable | PASS | `make test` |
| systemd units verify when available | PASS | `tests/test_units.sh` runs `systemd-analyze verify` when installed |
| installer preserves config | PASS | staged idempotence test appends and rechecks config |
| generated instructions avoid unmanaged files | PASS | only `/etc/dvd-ingester/dvd-ingester.env` is operator-edited |
| Home Assistant bridge exists | PASS | `T2R6.home-assistant-mqtt-bridge`, `dvd-ingester-ha-mqtt.service`, `tests/test_ha_mqtt_bridge.sh` |
| MPL pattern mapping documented | PASS | README, `MUSTER.md`, and `muster.yaml` name `T2R4.device-triggered-conveyor` and `T2R6.home-assistant-mqtt-bridge` |
