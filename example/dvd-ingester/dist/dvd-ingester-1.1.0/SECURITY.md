# Security

The installer owns only the `dvd-ingester` registration under
`/etc/muster/implementations.d`. It never removes or overwrites the shared
`/opt/muster` core. The bootstrap refuses a foreign `/usr/local/bin/muster`
entry and verifies downloaded core artifacts before publication.

The inspector reads configuration metadata, not environment-file contents.
Its only executable action is the explicitly declared argument-vector
`doctor.run`; initial rendering and refresh do not execute ingest commands or
perform network requests.

`dvd-ingester` runs privileged local automation around removable optical media
and high-volume filesystem writes.

## Boundaries

- udev only asks systemd to start `dvd-rip@%k.service`.
- Ripping and publishing run from `/opt/dvd-ingester/current/bin`.
- Configuration is preserved under `/etc/dvd-ingester/`.
- MQTT broker settings live in `/etc/dvd-ingester/dvd-ingester.mqtt.env`,
  installed with `0600` permissions.
- Runtime state is written under `/run/dvd-ingester`.
- Hot local payloads are written under `/var/cache/dvd-ingester/hot`.
- Apply mode proves that the cold destination is writable and, when `findmnt`
  is available, mounted.

## Notes

- Keep the optical-drive udev rule narrow.
- Treat all disc content as untrusted input.
- Do not set `ALLOW_UNMOUNTED_DEST=1` unless the configured destination is
  intentionally local storage.
- Prefer a project-specific `RIP_COMMAND` when the default `dvdbackup` or
  `makemkvcon` behavior is not sufficient.
- Install and update require the exact manifest project, a path-safe semantic
  version, and a 64-character lowercase SHA256. Archives may contain only
  regular files and directories beneath the expected project/version root;
  links and traversal are rejected before private staging extraction.
- A project-scoped lock records owner PID and start time for bounded dead-owner
  recovery. Valid version directories are reused without mutation.
- Current, registration, managed systemd units, and the udev rule form one
  rollback-aware transaction and are restored atomically when graph validation
  or the post-switch doctor fails.
- MQTT command payloads are not shell commands. They map only to explicit
  restart and enable/disable controls, and disable keeps the bridge available
  for re-enable.
