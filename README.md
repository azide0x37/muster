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
  muster.lock.json
  Makefile
  bin/
    install.sh
    uninstall.sh
    update.sh
    doctor.sh
    muster-bootstrap.sh
    muster-observation.sh
    release-transaction.sh
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

## Muster On A Box

Installing the first Muster implementation installs an independently versioned
host inspector and places one managed command in PATH:

```text
/usr/local/bin/muster -> ../../../opt/muster/current/bin/muster
/opt/muster/current -> releases/<core-version>
/etc/muster/implementations.d/<project>.json
```

The 0.1 core is install-once and shared: later implementation installs do not
downgrade or silently replace it, and uninstalling the last implementation
leaves it intact. Automatic core upgrade/removal is intentionally deferred to
an explicit operator-management surface rather than being owned by whichever
service happened to install next.

Run `muster` in a terminal to open the full-screen inspector. It presents each
implementation as a health-aware card, folds fully healthy subtrees, and opens
the paths that need attention. `/` filters across cards while preserving match
lineage; detail panes use viewport scrolling, proportional scrollbars, and
navigable evidence and metadata tables. Overview and inspect views lead with
verdicts, health causes, and current observations before literate context. The
same screen still exposes systemd components, runtime state, exact installed
pattern trees, and explicit doctor confirmation. Set `MUSTER_REDUCE_MOTION=1`
to disable spring scrolling, transitions, and blinking while retaining the same
information and controls.

The same object graph remains available without a TTY:

```sh
muster list
muster status dvd-ingester
muster inspect component:dvd-ingester:state:publish
muster explain component:dvd-ingester:unit:dvd-publish-one.service
muster inspect action:dvd-ingester:doctor.run
sudo muster doctor dvd-ingester
muster export --json
```

All inspection commands accept `--json`. `--root <path>` inspects a staged
installation without chrooting, which is also how the lifecycle tests prove
multi-service ownership.

### One Object Model

The TUI is the first renderer for Muster's reference runtime model—not a
collection of application-specific pages. Every implementation projects into
globally addressable components with:

- identity, kind, summary, metadata, actions, and children
- declared and recursively effective `healthy`, `degraded`, `unhealthy`, or
  `unknown` health, with causal paths preserved
- typed `depends_on`, `implements`, `owns`, `produces`, `consumes`, `observes`,
  and `configures` graph edges
- timestamped observations, checks, and artifacts
- literate purpose, responsibilities, and failure modes

`muster.yaml` schema 2 is the human-authored declaration.
`muster.lock.json` is produced with `muster compile muster.yaml` and freezes the
deterministic release graph plus adapter declarations. A tiny registration
points at the active release, so switching `/opt/<project>/current` also
switches the inspected object model.

Doctor is ordinary evidence in that graph. Implementations emit the common
`muster.observation/v1` envelope under
`/run/muster/<project>/observations/doctor.json`; the inspector never needs to
learn application-specific doctor output.

The initial view is read-only and performs no network requests. Running a
doctor is an explicit, root-required action because it updates root-owned
evidence under `/run/muster`; ordinary inspection remains unprivileged. Future
web, API, Grafana, or editor views can consume the same exported graph rather
than reimplementing discovery.

## Website

The project site lives at [`docs/index.html`](docs/index.html): one static file, no build step, zero runtime requests. It explains the concept, walks the systemd lifecycle with a live model of the rollback-aware update rail, diagrams the pattern library from the real MPL manifests, and inspects script-vs-muster A/B scenarios.

Serve it locally with `uv run python -m http.server -d docs`, or point GitHub Pages at `main` / `docs/`. The page ends with its own self-certification table, as is the law.

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
- the first installed implementation bootstraps the shared inspector and every implementation owns only its registration.
- static structure compiles into a deterministic lock while live adapters produce generic observations.
- Python is used only when it is clearly superior to shell, and then through `uv`.
- the README self-certifies compliance.

The README is the front door. `MUSTER.md` is the law book.
[`docs/OBJECT_MODEL.md`](docs/OBJECT_MODEL.md) is the runtime ontology shared by
the CLI, TUI, JSON export, and future presentation layers.

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

## Muster Self-Certification

| Requirement | Status | Evidence |
|---|---|---|
| systemd owns implementation lifecycle | PASS | Example `systemd/*.service` and `*.timer` units |
| shared command has independent versioned ownership | PASS | `/opt/muster/releases`, `current`, and managed PATH link in `muster-bootstrap.sh` |
| first service installs the command | PASS | Both example installers call `muster-bootstrap.sh ensure` |
| implementations register independently | PASS | `/etc/muster/implementations.d/*.json` staged lifecycle tests |
| uninstall cannot remove another inspection surface | PASS | Both install orders and uninstall orders in `tests/test_shared_muster_lifecycle.sh` |
| one normalized component graph backs CLI and TUI | PASS | `internal/model`, `internal/inspector`, `internal/cli`, and `internal/tui` |
| recursive health and typed relationships exist | PASS | Model fixed-point and explanation tests |
| doctor emits structured evidence | PASS | `muster.observation/v1` helpers and example doctor tests |
| pattern trees are release-specific | PASS | Schema 2 manifests and deterministic locks with verified MPL commits |
| installer and shared-core idempotence are tested | PASS | Example idempotence and concurrent cross-service bootstrap tests |
| project releases publish immutably and roll back transactionally | PASS | `release-transaction.sh`, project lifecycle regression tests |
| core artifacts are checksummed and safe to unpack | PASS | Platform manifests, archive verification, and bad-checksum staged test |
| test suite passes | PASS | `make test` |
| release artifacts build | PASS | `make package` and `tests/test_core_packages.sh` |
