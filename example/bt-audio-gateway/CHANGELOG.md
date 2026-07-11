# Changelog

This file is generated from `RELEASE.md`. Update release notes first, then run `make changelog`.

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

