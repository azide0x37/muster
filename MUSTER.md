# Muster

Muster is a repository scaffold framework for small Linux service appliances generated from prose architecture descriptions. A Muster repo must produce an installable, updateable, auditable service bundle, not just example commands.

Muster coordinates with the [Muster Pattern Library](https://github.com/azide0x37/muster-pattern-library). MPL supplies reusable atoms and composed patterns; this repo supplies the concrete service repository contract. When MPL has a matching atom, a Muster implementation should document that mapping before creating bespoke project vocabulary.

## Contract

1. systemd owns service lifecycle.
2. systemd timers own scheduled checks and update polling.
3. Configuration lives under `/etc/<project>/`.
4. Runtime code lives under `/opt/<project>/releases/<version>/`.
5. `/opt/<project>/current` points to the active release.
6. systemd units call scripts through `/opt/<project>/current/bin/...`.
7. The installer is idempotent.
8. The updater fetches release metadata, verifies SHA256, installs a new version, runs `doctor.sh`, and rolls back on failure.
9. The uninstaller stops and disables units and removes installed code. It preserves config unless called with `--purge`.
10. Shell is acceptable for simple glue around system tools.
11. Python is allowed only when shell becomes structurally worse for complex state, parsing, APIs, JSON, or orchestration.
12. If Python is used, the repo uses `uv`, `pyproject.toml`, full typing, and tests.
13. `README.md` includes a self-certification table with evidence.
14. `muster.yaml`, `MUSTER.md`, or `README.md` identifies relevant MPL atoms and composed patterns when they exist.
15. Codex may not mark work complete unless tests, release packaging, unit verification where available, and certification docs are current.
16. The first implementation installed on a host bootstraps the independently versioned Muster core under `/opt/muster/releases/<version>/`.
17. `/opt/muster/current` selects the active core and `/usr/local/bin/muster` is a managed relative symlink to that executable.
18. Each implementation owns exactly one registration under `/etc/muster/implementations.d/`; it never owns or removes the shared core.
19. `muster.yaml` schema 2 declares the implementation component graph and `muster.lock.json` freezes its deterministic release projection.
20. Every implementation and component has a globally addressable ID. Presentation layers consume the normalized graph rather than application-specific parsers.
21. Component health is recursive and uses `healthy`, `degraded`, `unhealthy`, and `unknown`; status must never be communicated through color alone.
22. Doctor is structured evidence using `muster.observation/v1`, not a privileged special case in the UI.
23. Initial inspection and refresh are read-only, execute no implementation commands, source no environment files, and perform no network requests.
24. Actions are explicit argument arrays. The initial core may execute only an advertised `doctor.run` action and never evaluates manifest shell fragments.
25. A declared installed lock is mandatory and its manifest digest must verify; installed inspection must not silently reinterpret an unlocked manifest.
26. Action IDs and observation IDs share the global object namespace with components. Declared and recursively effective health remain separately inspectable.
27. Doctor actions that write root-owned evidence declare and enforce `requires_root`; failed or stale evidence must never be presented as a current healthy run.
28. Release versions, archives, locks, and project identity are validated before immutable publication; current, registration, and managed unit changes are serialized and rolled back as one transaction.

## Host Inspector Contract

The shared host surface is:

```text
/opt/muster/releases/<core-version>/bin/muster
/opt/muster/current -> releases/<core-version>
/usr/local/bin/muster -> ../../../opt/muster/current/bin/muster
/etc/muster/implementations.d/<project>.json
```

An implementation installer must ensure a viable core before registration,
publish its active release, atomically write its own registry locator, and run
`muster validate`. A later implementation must return without consulting the
network when a viable core already exists. It must not downgrade or silently
replace that core.

An implementation updater must restore both its previous `current` target and
its previous registration when graph validation or doctor evidence rejects the
new release. An uninstaller removes only its own registration; even the last
implementation leaves the shared core installed for explicit operator removal.

## Runtime Object Model

Every implementation projects into one generic graph:

- components expose identity, kind, health, summary, metadata, actions,
  children, purpose, responsibilities, and failure modes;
- typed edges are `depends_on`, `implements`, `owns`, `produces`, `consumes`,
  `observes`, and `configures`;
- health flows recursively through child and dependency relationships;
- observations contain a producer, component, time, duration, checks, and
  artifacts;
- every object can be addressed by `muster inspect`, exported as JSON, and
  explained through the same graph used by the TUI.

Pattern trees belong to the installed release. Locks record the exact MPL
commit and resolved closure; the inspector must not substitute a newer website
or marketing catalog for historical installed truth.

Doctor observations live at a stable path such as
`/run/muster/<project>/observations/doctor.json`, are written atomically, and
separate health from operational phase. Stale evidence becomes `unknown`.

## MPL Pattern Vocabulary

Use MPL default branch names as the live source of truth. Do not freeze generated repos to an older MPL identifier when the library has renamed the pattern.

For device-triggered ingest appliances, use [`T2R4.device-triggered-conveyor`](https://github.com/azide0x37/muster-pattern-library/blob/main/patterns/t2/rare/T2R4.device-triggered-conveyor/README.md). The core atom chain is:

- `R2.device-binding`: udev hands device events to systemd.
- `R5.capability-mount`: a destination capability is proven before high-volume writes.
- `T2C1.hot-cold-nas-conveyor`: hot local work moves to cold NAS or network storage with atomic handoff.
- `C2.persistent-tick` and `T2C3.scheduled-herald`: timers own repeated checks, drains, and status work.
- `C5.failure-ratchet`: failures leave inspectable state instead of disappearing into logs.

## Examples

Concrete Muster implementations live under `example/<name>/`.

The included `example/bt-audio-gateway` project implements:

`Snapserver -> Raspberry Pi snapclient -> PipeWire/Pulse sink -> Bluetooth speaker`

Snapcast is the network transport. PipeWire is the local audio graph. Bluetooth is only the local last hop. systemd owns restart, routing, health checks, and update polling.

The included `example/dvd-ingester` project implements:

`udev -> systemd one-shot service -> rip script -> local work dir -> atomic NAS publish -> eject`

This is a concrete `T2R4.device-triggered-conveyor` instance. udev only triggers systemd. systemd owns the long-running rip. The script fingerprints before work, records success locally, writes final files atomically, and leaves Plex, Jellyfin, NAS, or desktop systems to handle later transcoding.
