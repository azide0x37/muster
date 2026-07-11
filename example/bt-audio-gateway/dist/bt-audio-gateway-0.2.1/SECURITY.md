# Security

The installer owns only the `bt-audio-gateway` registration under
`/etc/muster/implementations.d`. It never removes or overwrites the shared
`/opt/muster` core. The bootstrap refuses a foreign `/usr/local/bin/muster`
entry and verifies downloaded core artifacts before publication.

The inspector reads configuration metadata, not environment-file contents.
Its only executable action is the explicitly declared argument-vector
`doctor.run`; initial rendering and refresh do not execute gateway commands or
perform network requests.

This repo installs root-owned systemd units and scripts that manage Bluetooth, PipeWire/Pulse routing, and Snapcast client playback. Treat installer and updater changes as privileged-code changes.

## Reporting

Open a private security advisory or contact the repository owner before publishing issues that allow arbitrary command execution, unsafe update substitution, config disclosure, or privilege escalation.

## Update Safety

Install and update validate the exact manifest project, a path-safe semantic
version, and a 64-character lowercase SHA256. Archives are inspected before
extraction and may contain only regular files and directories beneath the
expected project/version root; links and traversal are rejected. Private work
and same-filesystem staging directories are created with `mktemp`.

A project-scoped lock serializes lifecycle changes and records its owner PID
and start time for bounded dead-owner recovery. Valid version directories are
reused without mutation. Switching current, registration, and managed systemd
units is rollback-aware, and previous files are restored atomically when graph
validation or `doctor.sh` fails.

The installer accepts only an existing non-root audio account. It preserves a
valid configured account, requires `MUSTER_AUDIO_USER` before replacing an
intentional invalid value, and only auto-migrates the historical `pi` default
when the invoking sudo account is valid.
