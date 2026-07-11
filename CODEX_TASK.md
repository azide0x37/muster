# Codex Task: Apply The Muster Framework

Use this task prompt in a fresh GitHub repository when you want Codex to turn a prose Linux appliance/service architecture into a Muster-compliant repo.

## Goal

Implement a Muster service repo: an installable, updateable, auditable Linux service appliance built from the architecture description supplied by the user.

Muster favors systemd components first. Use typed Python with `uv` only when shell becomes structurally worse for complex state, structured parsing, network APIs, JSON manipulation, or orchestration.

Before inventing a bespoke service shape, identify matching atoms or composed patterns in the [Muster Pattern Library](https://github.com/azide0x37/muster-pattern-library). Use MPL default branch identifiers in docs and `muster.yaml`. For example, device-event ingest appliances should start from `T2R4.device-triggered-conveyor`.

## Required Repo Shape

Create or update:

- `README.md`
- `AGENTS.md`
- `MUSTER.md`
- `RELEASE.md`
- `SECURITY.md`
- `muster.yaml`
- `muster.lock.json` generated deterministically from `muster.yaml`
- `Makefile`
- `bin/install.sh`
- `bin/uninstall.sh`
- `bin/update.sh`
- `bin/doctor.sh`
- `bin/muster-bootstrap.sh`
- `bin/muster-observation.sh`
- `bin/release-transaction.sh`
- `bin/render-units.sh`
- `systemd/*.service`
- `systemd/*.timer`
- `etc/*.env.example`
- `src/*`
- `tests/*`

## Architecture Rules

1. Identify the closest MPL atoms and composed pattern before naming project-specific pieces.
2. Identify the network leg, local plumbing leg, unreliable local leg, and lifecycle owner.
3. Keep network transport and local transport separate.
4. Put config under `/etc/<project>/`.
5. Put installed code under `/opt/<project>/releases/<version>/`.
6. Point `/opt/<project>/current` to the active release.
7. Make systemd units call `/opt/<project>/current/bin/...`.
8. Add health checks through `doctor.sh`.
9. Add update polling through a systemd timer.
10. Make the installer idempotent.
11. Make the updater verify SHA256 and roll back if `doctor.sh` fails.
12. Describe every owned unit, config surface, release pointer, runtime fact, doctor, and MPL claim as globally addressable components in schema 2.
13. Emit structured doctor evidence at `/run/muster/<project>/observations/doctor.json`.
14. Give every component non-empty, implementation-specific `what` and `why` text.

## Installer Requirements

`bin/install.sh` must:

- run from a checkout and through `curl | sh`
- require root unless using a staged root for tests
- install package dependencies when supported
- create `/etc/<project>/`
- preserve existing config
- install into `/opt/<project>/releases/<version>/`
- validate version strings, archive members, embedded version, lock, and project
  identity before publishing an immutable release directory
- serialize same-project lifecycle changes and never rewrite a valid existing
  version in place
- atomically update `/opt/<project>/current`
- install systemd units
- run `systemctl daemon-reload`
- enable required services and timers
- support staged idempotence tests without touching the host
- ensure the shared `/opt/muster` core and managed `/usr/local/bin/muster` link
- atomically register only this implementation under `/etc/muster/implementations.d/`
- validate the normalized graph before enabling services
- refuse to overwrite an unrelated `muster` command

## Updater Requirements

`bin/update.sh` must:

- read `/etc/<project>/<project>.env`
- skip cleanly when `AUTOUPDATE=0`
- fetch release metadata
- compare installed and available versions
- download the artifact
- verify SHA256
- unpack into a new release directory
- stage and validate the new immutable release before switching
- switch `/opt/<project>/current`, registration, managed units/rules, and
  rollback state as one transaction
- restart affected services
- run `doctor.sh`
- roll back and restart services if the health check fails
- restore the prior implementation registration when validation or health fails

## Documentation Requirements

`README.md` must include:

- service purpose
- text architecture diagram
- MPL pattern mapping when the library has a matching atom or composed pattern
- install command
- manual install steps
- config reference
- systemd unit list
- update and rollback behavior
- troubleshooting commands
- integration notes for adjacent systems
- Muster self-certification table
- shared inspector registration and `muster` command behavior
- object IDs, component graph, doctor evidence, and pattern-tree inspection

## Completion Bar

Do not declare completion until:

- `make test` passes, or unsupported local tools are documented
- `make package` builds release artifacts
- systemd units verify when `systemd-analyze` is available
- installer idempotence is tested
- README self-certification is current
- known limitations are listed
- MPL mapping is current when applicable
- the committed lock matches `muster compile muster.yaml`
- first/second install order and shared-core uninstall ownership are tested
