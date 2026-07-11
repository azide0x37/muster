# Security

## Trust boundary

The shared `muster` process runs as the invoking operator. It reads root-owned
registrations, immutable release locks, bounded runtime observations, and
systemd properties. Initial rendering performs no network requests and does
not source implementation environment files.

Implementation manifests and locks are privileged configuration. An attacker
who can modify files under `/etc/muster` or `/opt/<project>/current` already
controls the appliance's installed service surface.

## Actions

The object model can advertise actions, but the 0.1 core executes only the
bounded `doctor.run` action. Commands are stored as argument arrays and are
passed directly to `exec`; the core does not evaluate shell fragments. Doctor
execution is explicit, confirmed in the TUI, and never triggered during initial
rendering or refresh. Bundled doctor actions declare `requires_root`; the core enforces that
boundary before execution, while all existing evidence remains readable from
an unprivileged inspector. A doctor that cannot atomically persist its evidence
fails rather than presenting an unchanged observation as a successful run.

## Installation

The bootstrap verifies the platform-specific core artifact against a 64-digit
SHA256 from its release manifest, rejects unsafe archive members, publishes an
immutable release, and atomically switches `/opt/muster/current`. It refuses
to replace an unrelated `/usr/local/bin/muster` file or foreign symlink.

Implementation uninstallers never remove the shared core. This prevents one
service from damaging the inspection surface of another service.

## Sensitive data

The inspector reports configuration-file presence and permissions, not file
contents. Observation producers must avoid secrets in summaries, metadata,
checks, and artifacts. MQTT credentials remain in the implementation's
root-managed `0600` configuration.
