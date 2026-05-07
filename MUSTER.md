# Muster

Muster is a repository scaffold framework for small Linux service appliances generated from prose architecture descriptions. A Muster repo must produce an installable, updateable, auditable service bundle, not just example commands.

## Contract

1. systemd owns service lifecycle.
2. systemd timers own scheduled checks and update polling.
3. Configuration lives under `/etc/<project>/`.
4. Runtime code lives under `/opt/<project>/releases/<version>/`.
5. `/opt/<project>/current` points to the active release.
6. systemd units call scripts through `/opt/<project>/current/bin/...`.
7. The installer is idempotent.
8. The updater fetches release metadata, verifies SHA256, installs a new version, runs `doctor.sh`, and rolls back on failure.
9. The uninstaller stops and disables units and removes installed code. It preserves config unless called with `--purge`.
10. Shell is acceptable for simple glue around system tools.
11. Python is allowed only when shell becomes structurally worse for complex state, parsing, APIs, JSON, or orchestration.
12. If Python is used, the repo uses `uv`, `pyproject.toml`, full typing, and tests.
13. `README.md` includes a self-certification table with evidence.
14. Codex may not mark work complete unless tests, release packaging, unit verification where available, and certification docs are current.

## Examples

Concrete Muster implementations live under `example/<name>/`.

The included `example/bt-audio-gateway` project implements:

`Snapserver -> Raspberry Pi snapclient -> PipeWire/Pulse sink -> Bluetooth speaker`

Snapcast is the network transport. PipeWire is the local audio graph. Bluetooth is only the local last hop. systemd owns restart, routing, health checks, and update polling.

The included `example/dvd-ingester` project implements:

`udev -> systemd one-shot service -> rip script -> local work dir -> atomic NAS publish -> eject`

udev only triggers systemd. systemd owns the long-running rip. The script fingerprints before work, records success locally, writes final files atomically, and leaves Plex, Jellyfin, NAS, or desktop systems to handle later transcoding.
