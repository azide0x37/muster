# Muster

Muster is a repo scaffold framework for small Linux service appliances. It turns a prose architecture into a repository that can install, update, health-check, and package a service without hiding operational behavior in a README.

The framework lives at the repository root. Concrete implementations live under `example/<name>/`.

Current examples:

- `example/bt-audio-gateway`: Snapcast to PipeWire/Pulse to Bluetooth speaker.
- `example/dvd-ingester`: DVD insert to local ingest to atomic NAS publish.

## Framework Files

- `MUSTER.md`: the contract every implementation must satisfy.
- `CODEX_TASK.md`: the task prompt to give Codex when applying Muster to a new repo.
- `AGENTS.md`: agent rules for working in this repository.
- `Makefile`: runs checks across all examples.

## Use Muster In A Fresh Repo

1. Initialize the repository.
2. Copy `AGENTS.md`, `MUSTER.md`, and `CODEX_TASK.md`.
3. Paste the project architecture into Codex.
4. Tell Codex to apply the Muster framework.
5. Require `make test` and `make package` before declaring the repo complete.

The implementation should produce systemd units, idempotent installers, staged tests, release artifacts, health checks, autoupdate timers, and README self-certification.

## Test All Examples

```sh
make test
```

## Package All Examples

```sh
make package
```

Each example owns its generated `dist/` directory.
