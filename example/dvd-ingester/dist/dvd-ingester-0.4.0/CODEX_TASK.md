# Codex Task: dvd-ingester

Rebuild `dvd-ingester` as a Muster service repo implementing
`T2R4.device-triggered-conveyor`.

The appliance binds optical drive readiness to one bounded systemd job. The job
records the event, proves the cold destination capability, waits for hot local
capacity, stages the rip locally, marks inspectable state, and leaves completed
work for a timer-driven publish drain.

Required properties:

- udev only requests `dvd-rip@%k.service`.
- systemd owns rip, publish, doctor, and update lifecycle.
- config lives under `/etc/dvd-ingester/dvd-ingester.env`.
- installed code lives under `/opt/dvd-ingester/releases/<version>/`.
- `/opt/dvd-ingester/current` points at the active release.
- hot work lives under `/var/cache/dvd-ingester/hot`.
- state lives under `/run/dvd-ingester`.
- cold publish writes to the destination through an atomic final rename.
- failures leave state files and marker files for inspection.
- staged install and mock conveyor tests must pass before comparing to legacy
  tests.
