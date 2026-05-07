# Codex Task: Implement bt-audio-gateway With Muster

Build a Muster-compliant Linux appliance that bridges Snapcast audio to a Bluetooth speaker through PipeWire/Pulse.

The architecture is:

`Home Assistant / Music Assistant -> Snapserver -> Raspberry Pi snapclient -> PipeWire/Pulse sink -> Bluetooth speaker`

Important constraints:

- Snapcast is the network transport.
- Bluetooth is only the local unreliable last hop.
- PipeWire/Pulse is the audio plumbing.
- systemd owns lifecycle, restart, health checks, and update polling.
- Configuration lives in `/etc/bt-audio-gateway/bt-audio-gateway.env`.
- Installed code lives under `/opt/bt-audio-gateway/releases/<version>`.
- `/opt/bt-audio-gateway/current` points to the active release.
- The installer must be idempotent and preserve existing config.
- The updater must verify SHA256 and roll back when `doctor.sh` fails.
- Completion requires `make test`, `make package`, installer idempotence, and current README self-certification.
