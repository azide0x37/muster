# Muster 0.1.0

The first Muster host inspector turns the framework's operational vocabulary
into one addressable runtime object graph.

## Added

- A shared `muster` executable built with Bubble Tea v2 and Lip Gloss v2.
- A responsive, keyboard-first TUI for implementations, components, recursive
  health, pattern trees, doctor evidence, and literate explanations.
- Scriptable `list`, `status`, `inspect`, `explain`, `doctor`, `export`,
  `validate`, `compile`, and `version` commands with JSON output.
- Schema 2 authoring manifests and deterministic `muster.lock.json` release
  projections.
- A shared-core bootstrap under `/opt/muster`, a managed PATH symlink, and
  independently owned implementation registrations.
- Structured `muster.observation/v1` doctor evidence.
- Linux amd64, arm64, and armv7 core artifacts with SHA256 manifests.

## Operational contract

- Installing the first Muster implementation installs the shared inspector.
- Later implementations register without replacing or downgrading that core.
- An implementation uninstaller removes only its own registration.
- The inspector is read-only during initial rendering; doctor execution is an
  explicit advertised action that declares and enforces its root requirement.
- Failed doctors still publish and refresh structured evidence; stale evidence
  is rendered as unknown rather than reusing an old green result.
- Recursive health exports retain both declared and effective values so
  `muster explain` can show the actual component and graph path responsible.
- Project releases are staged, identity/lock/archive validated, published
  immutably, and activated with registration plus managed files in one
  rollback-aware transaction.
