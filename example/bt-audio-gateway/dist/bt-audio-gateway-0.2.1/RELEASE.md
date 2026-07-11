# bt-audio-gateway Release Notes

## 0.2.1

- Derive the PipeWire/Pulse runtime path from the instantiated service user's
  actual UID. The prior `%U` specifier resolved to the system manager's UID
  (`0`), so snapclient probed `/run/user/0` even for a valid non-root user.
- Add a static regression check for the service-user runtime socket path.

## 0.2.0

- Register the implementation with the shared Muster host inspector and
  bootstrap the independently versioned `muster` command when this is the
  first implementation on a server.
- Add a schema 2 component graph, deterministic release lock, exact partial
  T2R1 pattern tree, and globally addressable service components.
- Emit structured `muster.observation/v1` doctor evidence and inspect the
  active watcher plus configured snapclient instance.
- Restore the previous registry state as well as the release symlink when an
  update fails graph validation or health inspection.
- Harden install and update as one serialized release transaction with private
  staging, strict project/version/SHA and archive validation, immutable version
  reuse, and atomic rollback of current, registration, and managed units.
- Resolve the fresh-install audio user from the invoking sudo account when the
  legacy `pi` default is unavailable, migrate that legacy default during a
  `0.1.4` upgrade, and fail truthfully when audio services cannot start.

## Release Process

Muster release artifacts are built into `dist/`.

Required artifacts:

- `bt-audio-gateway-<version>.tar.gz`
- `bt-audio-gateway-<version>.tar.gz.sha256`
- `install.sh`
- `manifest.json`

Build:

```sh
make package
```

The release manifest records the exact project, version, artifact name, SHA256,
and expected install/update entry points. Archives must contain only regular
files and directories beneath `bt-audio-gateway-<version>/`; the embedded
manifest digest lock must match before publication. The public install path
should prefer a tagged GitHub release asset:

```sh
curl -fsSL https://github.com/azide0x37/bt-audio-gateway/releases/latest/download/install.sh | sudo sh
```

The raw `main` installer may exist for development, but stable installs and autoupdates should consume immutable release artifacts.
