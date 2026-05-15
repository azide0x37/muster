# Security

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
- Updates use a release manifest and SHA256 verification before switching the
  `/opt/dvd-ingester/current` symlink.
- MQTT command payloads are not shell commands. They map only to explicit
  restart and enable/disable controls, and disable keeps the bridge available
  for re-enable.
