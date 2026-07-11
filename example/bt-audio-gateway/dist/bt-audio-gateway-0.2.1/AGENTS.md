When running python, use UV. Always pull latest from remote at the start of your action phase.

# Muster Example Contract

This example implements the Muster pattern for `bt-audio-gateway`. Future Codex runs may not declare it complete unless:

- `make test` passes, or each failure is explicitly documented with the unsupported local dependency or environment constraint.
- systemd units verify with `systemd-analyze verify` when that command is available.
- `README.md` contains an up-to-date Muster self-certification table.
- release artifact generation through `make package` has been tested.
- installer idempotence has been considered and tested through a staged root or equivalent.
- generated instructions do not tell users to edit unmanaged files outside the repo without explaining how the installer owns or preserves them.

Prefer systemd units and timers for lifecycle, health, and update polling. Use typed Python only when it is clearly superior to shell for structured state, parsing, network APIs, JSON manipulation, or long-running orchestration. If Python is used, use `uv`, `pyproject.toml`, full typing, and tests.
