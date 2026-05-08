# Security

This repo installs root-owned systemd units and scripts that manage Bluetooth, PipeWire/Pulse routing, and Snapcast client playback. Treat installer and updater changes as privileged-code changes.

## Reporting

Open a private security advisory or contact the repository owner before publishing issues that allow arbitrary command execution, unsafe update substitution, config disclosure, or privilege escalation.

## Update Safety

`bin/update.sh` must verify release artifact SHA256 before switching `/opt/bt-audio-gateway/current`. It must run `bin/doctor.sh` after switching and roll back to the previous release when the health check fails.
