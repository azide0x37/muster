<p align="center">
  <img src="assets/brand/muster-logo-wordmark.png" alt="Muster" width="720">
</p>

# Muster

Muster is a repo scaffold framework for small Linux service appliances.

It turns a prose architecture into a boring, installable, updateable, health-checkable service repository. The goal is not to make tiny tools fancy. The goal is to stop tiny tools from becoming future server archaeology.

Muster is for the kind of project that starts as:

> I just need a Raspberry Pi to watch for X, run Y, publish Z, and restart itself when Bluetooth flakes out again.

That sort of project is easy to prototype and weirdly easy to trust too soon. A shell script becomes a service. A service becomes infrastructure. Infrastructure becomes the thing you forgot was holding up the shed.

Muster gives those projects a minimum operational skeleton:

- systemd units for lifecycle
- systemd timers for checks and update polling
- idempotent installers
- versioned installs under `/opt/<project>/releases/<version>/`
- config under `/etc/<project>/`
- health checks through `doctor.sh`
- packaging through `make package`
- rollback-aware updates
- README self-certification
- agent instructions that make Codex prove the repo is not merely plausible

It is deliberately boring.

No Kubernetes cosplay. No snowflake Pi setup. No README full of sacred hand-entered commands that only worked once.

Muster is a way to use agentic coding without letting the agent leave you with a pile of charming little security problems under `/usr/local/bin`.

## The Problem

Agentic coding makes it very easy to create useful software faster than we create discipline around it.

That is the trap.

The fun part is the idea. The fun part is the protocol dance. The fun part is making the DVD drive rip on insert or the Bluetooth speaker behave like a network audio endpoint.

The unfun part is the part that matters six months later:

- install layout
- idempotence
- logs
- service ownership
- config preservation
- health checks
- rollback
- tests
- update hygiene
- security notes
- documentation that is not just vibes in Markdown

Muster exists to offload that boring work to the agent, but in a constrained way. Not "make me a script." More like:

> Here is the architecture. Produce the operational harness. Pass muster.

That constraint is the point.

Agentic coding should make small tools more productionizable, not more haunted.

## The Muster Pattern

A Muster repo takes a small service architecture and forces it into a production-shaped repository.

For example:

```text
Home Assistant / Music Assistant
  -> Snapcast stream
  -> Raspberry Pi snapclient
  -> PipeWire/Pulse sink
  -> Bluetooth speaker
```

Muster asks the operational questions before the script escapes the workbench:

- What owns lifecycle?
- What owns updates?
- Where does configuration live?
- What happens if install runs twice?
- What happens if update fails?
- How do we prove the service is healthy?
- How does the next person understand this without asking the original author?

The answer is a repo that installs cleanly, explains itself, packages itself, and gives future-you a fighting chance.

## Muster Pattern Library

Muster implementations should be described in terms of the [Muster Pattern Library](https://github.com/azide0x37/muster-pattern-library) when a matching atom or composed pattern exists. This repo owns the service-repo contract; MPL owns the smaller vocabulary for reusable solution shapes.

Use MPL as the first pass vocabulary before inventing a project-specific structure:

- common atoms describe service capsules, timers, lazy resources, failure markers, device bindings, and capabilities
- composed patterns describe repeatable appliance shapes such as hot/cold NAS conveyors and device-triggered conveyors
- project repos adapt those atoms to real services, paths, packages, installers, and user-facing docs

For device ingest appliances, the current default pattern is [`T2R4.device-triggered-conveyor`](https://github.com/azide0x37/muster-pattern-library/blob/main/patterns/t2/rare/T2R4.device-triggered-conveyor/README.md). It composes device binding, capability proof, hot-storage backpressure, scheduled drain/status work, and inspectable failure state.

## What It Produces

Concrete Muster implementations live under `example/<name>/`. A generated implementation should look like this:

```text
<project>/
  README.md
  AGENTS.md
  MUSTER.md
  RELEASE.md
  SECURITY.md
  muster.yaml
  Makefile
  bin/
    install.sh
    uninstall.sh
    update.sh
    doctor.sh
    render-units.sh
  etc/
    <project>.env.example
  src/
  systemd/
    *.service
    *.timer
  tests/
```

The exact service code changes by project. The operational skeleton does not.

## Current Examples

- `example/bt-audio-gateway`: Snapcast to PipeWire/Pulse to Bluetooth speaker.
- `example/dvd-ingester`: DVD insert to local ingest to atomic NAS publish.

Each example is a self-contained `impl muster` project with its own installer, updater, doctor, tests, release artifacts, and README self-certification.

## Brand Assets

Brand assets live in `assets/brand/`:

- `muster-logo-wordmark.png`: primary README/header asset.
- `muster-logo.png`: square icon mark.
- `muster-wordmark.png`: horizontal wordmark.

## Use Muster In A Fresh Repo

1. Initialize the repository.
2. Copy `AGENTS.md`, `MUSTER.md`, and `CODEX_TASK.md`.
3. Paste the project architecture into Codex.
4. Identify the closest MPL atoms and composed pattern.
5. Tell Codex to apply the Muster framework using those atoms.
6. Require `make test` and `make package` before declaring the repo complete.

The implementation should produce systemd units, idempotent installers, staged tests, release artifacts, health checks, autoupdate timers, rollback-aware updates, and README self-certification.

## Contract

The formal contract lives in [`MUSTER.md`](MUSTER.md). The short version:

- systemd owns service lifecycle.
- systemd timers own scheduled checks and update polling.
- config lives under `/etc/<project>/`.
- runtime code lives under `/opt/<project>/releases/<version>/`.
- `/opt/<project>/current` points to the active release.
- installers are idempotent.
- updates verify SHA256 and roll back on failed health checks.
- matching MPL atoms are documented in `muster.yaml`, `MUSTER.md`, or the README.
- Python is used only when it is clearly superior to shell, and then through `uv`.
- the README self-certifies compliance.

The README is the front door. `MUSTER.md` is the law book.

## Test Everything

```sh
make test
```

The root `Makefile` runs each example's test suite.

## Package Everything

```sh
make package
```

Each example owns its generated `dist/` directory.

## Philosophy

Muster is not a general app framework.

It is not a replacement for Ansible, Nix, Docker, or Debian packaging.

It is not trying to solve enterprise deployment.

It is for the neglected middle zone: small Linux appliances, Raspberry Pi services, home-lab helpers, office automations, data-ingest boxes, and weird little tools that are too useful to leave as loose scripts.

Small tools deserve boring operational discipline. Muster makes that discipline repeatable.
