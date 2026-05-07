When running python, use UV. Always pull latest from remote at the start of your action phase.

# Muster Example Contract

This example implements the Muster pattern for `dvd-ingester`. Future Codex runs may not declare it complete unless:

- `make test` passes, or unsupported local tooling is explicitly documented.
- `systemd-analyze verify` has passed when available.
- udev rules are syntax-present and only trigger systemd; they must not run long work directly.
- `README.md` self-certification is current.
- release artifact generation through `make package` has been tested.
- installer idempotence is tested through a staged root.

Use systemd for lifecycle and timers. Use typed Python through `uv` only if the DVD title-selection policy or metadata handling becomes too complex for maintainable shell.
