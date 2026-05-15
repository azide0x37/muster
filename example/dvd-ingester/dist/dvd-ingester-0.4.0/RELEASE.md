# dvd-ingester Release Notes

## 0.4.0

- Reimplemented the example from scratch around
  `T2R4.device-triggered-conveyor`.
- Added hot-storage backpressure waiting before rip work begins.
- Added named capability, capacity, rip, handoff, and publish state files.
- Added timer-driven publish drain with atomic final rename at the cold
  destination.
- Added a mock conveyor flow test before legacy test comparison.
- Preserved the release target as `azide0x37/dvd-ingester`.
