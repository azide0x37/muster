When running python, use UV. Always pull latest from remote at the start of your action phase.

# Muster Example Contract

This repository is a Muster example implementation of the
`T2R4.device-triggered-conveyor` Muster Pattern Library pattern.

Future Codex runs may not declare it complete unless:

- `make test` passes, or unsupported local tooling is explicitly documented.
- `make package` builds release artifacts.
- `systemd-analyze verify` passes when available.
- udev only requests systemd units and never runs long work directly.
- installer idempotence is tested through a staged root.
- `README.md` self-certification reflects the current implementation.

Use systemd for lifecycle and timers. Use typed Python through `uv` only if the
DVD title-selection policy or metadata handling becomes too complex for
maintainable shell.
