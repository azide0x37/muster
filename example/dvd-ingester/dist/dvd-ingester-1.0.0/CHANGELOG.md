# Changelog

This file is generated from `RELEASE.md`. Update release notes first, then run `make changelog`.

## 1.0.0

- Added Home Assistant MQTT discovery and state publishing through
  `T2R6.home-assistant-mqtt-bridge`.
- Added scoped controls for restart and enable/disable without stopping active
  `dvd-rip@*.service` jobs.
- Added separate `/etc/dvd-ingester/dvd-ingester.mqtt.env` defaults so broker
  credentials are not mixed into the general appliance config.
- Added mock tests for discovery payloads, state aggregation, and rejected
  command payloads.

## 0.4.0

- Reimplemented the example from scratch around
  `T2R4.device-triggered-conveyor`.
- Added hot-storage backpressure waiting before rip work begins.
- Added named capability, capacity, rip, handoff, and publish state files.
- Added timer-driven publish drain with atomic final rename at the cold
  destination.
- Added a mock conveyor flow test before legacy test comparison.
- Preserved the release target as `azide0x37/dvd-ingester`.
