# dvd-ingester Muster Notes

`dvd-ingester` is an `impl muster` example for a DVD ingest appliance. It is a
fresh concrete implementation of the Muster Pattern Library
`T2R4.device-triggered-conveyor` pattern.

Architecture:

```text
DVD drive readiness
  -> udev SYSTEMD_WANTS
  -> dvd-rip@.service
  -> dvd-rip-one
  -> hot local handoff
  -> dvd-publish-one timer drain
  -> mounted cold destination
```

## MPL Mapping

Primary pattern:

`T2R4.device-triggered-conveyor`

Subpattern mapping:

| Project boundary | MPL vocabulary | Implementation |
| --- | --- | --- |
| udev starts only a systemd job | `R2.device-binding`, `C1.service-capsule` | `udev/90-dvd-ingester.rules`, `systemd/dvd-rip@.service` |
| destination is proven before high-volume work | `R5.capability-mount`, `C4.lazy-resource-gate` | `src/dvd-rip-one`, `src/dvd-publish-one` capability checks |
| hot local work drains to cold storage | `T2C1.hot-cold-nas-conveyor` | `.ingest-complete` hot handoff and atomic publish rename |
| repeated drains, health checks, and updates | `C2.persistent-tick`, `T2C3.scheduled-herald` | publish, doctor, and update timers |
| failed and degraded states stay inspectable | `C5.failure-ratchet` | JSON state files and `.ingest-failed` markers |
| installer, updater, uninstaller, package | `C6.lifecycle-capsule` | `bin/*.sh`, `Makefile`, release manifest |

## Stable Contract Notes

- The udev rule does not run long work.
- The rip job waits for hot-storage capacity instead of immediately failing
  while publish work is still draining.
- Real apply mode refuses to use an unmounted destination when `findmnt` is
  available, unless `ALLOW_UNMOUNTED_DEST=1` is set deliberately.
- Mock mode is the test boundary. It exercises capability proof, backpressure,
  handoff state, publish drain, installer idempotence, and package generation
  without touching host services.
