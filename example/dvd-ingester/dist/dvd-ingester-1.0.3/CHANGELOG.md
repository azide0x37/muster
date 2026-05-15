# Changelog

This file is generated from `RELEASE.md`. Update release notes first, then run `make changelog`.

## 1.0.3

- Log captured `doctor.sh` stdout and stderr before rolling back a failed
  release update, so `journalctl -t dvd-ingester-update` shows the actual
  health-check failure instead of only the generic rollback message.
- Added a regression test that exercises failed update rollback and verifies
  the doctor output is preserved in updater logs.

## 1.0.2

- Avoid recursive NAS size scans in the Home Assistant MQTT bridge so slow NAS
  reads do not make the one-minute telemetry refresh time out and leave stale
  retained MQTT state in Home Assistant.
- Detect loaded optical media from udev `ID_CDROM_MEDIA=1` before falling back
  to filesystem probing, making the disk state sensor more accurate while a
  newly inserted disc is being claimed by udev/systemd.
- Report conveyance as `waiting_for_extraction` while an extraction is active
  so bridge snapshots do not mix a new rip state with an older publish handoff.

## 1.0.1

- Removed the rip service's device lifetime binding so ejecting a completed
  disc does not cause systemd to terminate the post-rip hot handoff.
- Made the hot handoff publish-safe by copying completed rips into hidden
  `.incoming-*` hot directories and writing `.ingest-complete` only after the
  handoff copy is complete.
- Clean up interrupted hidden hot handoff directories while preserving the
  completed work source until the visible hot handoff succeeds.

## 1.0.0

- Added Home Assistant MQTT discovery and state publishing through
  `T2R6.home-assistant-mqtt-bridge`.
- Added scoped controls for restart and enable/disable without stopping active
  `dvd-rip@*.service` jobs.
- Added separate `/etc/dvd-ingester/dvd-ingester.mqtt.env` defaults so broker
  credentials are not mixed into the general appliance config.
- Added mock tests for discovery payloads, state aggregation, and rejected
  command payloads.
- Restored disc-title output directory names in the hot/cold conveyor, using
  `<disc-title>_<fingerprint>` instead of device/timestamp run names.
- Eject completed discs before the slower hot/cold handoff so operators can
  swap media without waiting for archive movement.
- Updated rip and handoff state files at the start of extraction so Home
  Assistant does not show stale `ready_for_cold_publish` while a new disc is
  actively ripping.
- Added Home Assistant MQTT folder index sensors for work, hot, and completed
  title directories, with counts in sensor state and bounded directory lists in
  attributes.
- Added diagnostic Home Assistant entities for capability state, hot-capacity
  state, publish state, publish counts, and installed version.
- Added Home Assistant MQTT disk state plus extraction and conveyance progress
  sensors. Extraction progress compares current run bytes with disc size;
  conveyance progress compares active incoming publish bytes with hot source
  size.

## 0.4.0

- Reimplemented the example from scratch around
  `T2R4.device-triggered-conveyor`.
- Added hot-storage backpressure waiting before rip work begins.
- Added named capability, capacity, rip, handoff, and publish state files.
- Added timer-driven publish drain with atomic final rename at the cold
  destination.
- Added a mock conveyor flow test before legacy test comparison.
- Preserved the release target as `azide0x37/dvd-ingester`.
