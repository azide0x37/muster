# Codex Task: Apply The Muster Framework

Use this task prompt in a fresh GitHub repository when you want Codex to turn a prose Linux appliance/service architecture into a Muster-compliant repo.

## Goal

Implement a Muster service repo: an installable, updateable, auditable Linux service appliance built from the architecture description supplied by the user.

Muster favors systemd components first. Use typed Python with `uv` only when shell becomes structurally worse for complex state, structured parsing, network APIs, JSON manipulation, or orchestration.

## Required Repo Shape

Create or update:

- `README.md`
- `AGENTS.md`
- `MUSTER.md`
- `RELEASE.md`
- `SECURITY.md`
- `muster.yaml`
- `Makefile`
- `bin/install.sh`
- `bin/uninstall.sh`
- `bin/update.sh`
- `bin/doctor.sh`
- `bin/render-units.sh`
- `systemd/*.service`
- `systemd/*.timer`
- `etc/*.env.example`
- `src/*`
- `tests/*`

## Architecture Rules

1. Identify the network leg, local plumbing leg, unreliable local leg, and lifecycle owner.
2. Keep network transport and local transport separate.
3. Put config under `/etc/<project>/`.
4. Put installed code under `/opt/<project>/releases/<version>/`.
5. Point `/opt/<project>/current` to the active release.
6. Make systemd units call `/opt/<project>/current/bin/...`.
7. Add health checks through `doctor.sh`.
8. Add update polling through a systemd timer.
9. Make the installer idempotent.
10. Make the updater verify SHA256 and roll back if `doctor.sh` fails.

## Installer Requirements

`bin/install.sh` must:

- run from a checkout and through `curl | sh`
- require root unless using a staged root for tests
- install package dependencies when supported
- create `/etc/<project>/`
- preserve existing config
- install into `/opt/<project>/releases/<version>/`
- atomically update `/opt/<project>/current`
- install systemd units
- run `systemctl daemon-reload`
- enable required services and timers
- support staged idempotence tests without touching the host

## Updater Requirements

`bin/update.sh` must:

- read `/etc/<project>/<project>.env`
- skip cleanly when `AUTOUPDATE=0`
- fetch release metadata
- compare installed and available versions
- download the artifact
- verify SHA256
- unpack into a new release directory
- switch `/opt/<project>/current`
- restart affected services
- run `doctor.sh`
- roll back and restart services if the health check fails

## Documentation Requirements

`README.md` must include:

- service purpose
- text architecture diagram
- install command
- manual install steps
- config reference
- systemd unit list
- update and rollback behavior
- troubleshooting commands
- integration notes for adjacent systems
- Muster self-certification table

## Completion Bar

Do not declare completion until:

- `make test` passes, or unsupported local tools are documented
- `make package` builds release artifacts
- systemd units verify when `systemd-analyze` is available
- installer idempotence is tested
- README self-certification is current
- known limitations are listed
