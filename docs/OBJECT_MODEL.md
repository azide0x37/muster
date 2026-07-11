# Muster Runtime Object Model

The Muster TUI, plain-text CLI, JSON export, and future remote views consume one
normalized object graph. None of those renderers parse application-specific
state or know what a DVD, Bluetooth speaker, camera, database, or battery is.

## Identity

Every object has a stable, globally addressable ID:

```text
implementation:dvd-ingester
component:dvd-ingester:unit:dvd-publish-one.service
component:dvd-ingester:doctor
pattern:dvd-ingester:T2R4.device-triggered-conveyor
action:dvd-ingester:doctor.run
```

IDs are durable API, not display labels. `muster inspect`, `muster explain`,
JSON exports, and all renderers address the same IDs.

Components expose effective recursive health in `health` and retain the
adapter's direct assertion in `declared_health`. Actions and observations are
addressable too: inspecting either returns that object, while explaining it
follows its target or observed component into the same dependency graph.

## Graph

An implementation owns a root component and an ordered set of components.
Every component exposes:

- kind, summary, health, metadata, actions, and child IDs;
- a literate `what` and `why`;
- responsibilities and failure modes.

Typed edges add non-hierarchical meaning:

| Edge | Meaning |
|---|---|
| `depends_on` | source cannot satisfy its contract without target |
| `implements` | source is evidence for a pattern or declared behavior |
| `owns` | lifecycle and registration ownership |
| `produces` | source emits target state or work |
| `consumes` | source reads or drains target state or work |
| `observes` | source produces evidence about target |
| `configures` | source supplies bounded configuration to target |

Trees remain the default terminal projection, but the in-memory object is a
graph. This lets `muster explain` traverse dependencies and health causes
without application-specific logic.

## Recursive Health

Health is one of:

- `healthy`: the declared contract is currently satisfied;
- `degraded`: useful work remains possible, but attention is warranted;
- `unhealthy`: the declared contract is not satisfied;
- `unknown`: current evidence is missing, stale, or inconclusive.

Children and health-propagating graph relationships are reduced to a fixed
point. A degraded child degrades its parent; an unhealthy required dependency
makes its dependents unhealthy. Cycles are supported and cannot recurse
forever. Explanations retain deterministic paths to the observations that
caused the effective status.

Operational phase is separate from health. `waiting`, `publishing`, `idle`, or
`connected` may be useful metadata, but they are not substitutes for health.

## Authoring, Locking, And Registration

There are three artifacts:

1. `muster.yaml` schema 2 is the human-authored declaration.
2. `muster.lock.json` is its deterministic normalized release projection.
3. `/etc/muster/implementations.d/<project>.json` is the machine-local locator
   for the active manifest and lock.

Compile a lock with:

```sh
muster compile muster.yaml
```

The lock records the source-manifest SHA256, exact MPL repository and commit,
resolved pattern components, typed edges, bounded adapter declarations, and
advertised actions. It contains no wall-clock generation time, so identical
inputs produce identical bytes.

The inspector verifies the manifest digest before using a lock. Registration
points through `/opt/<project>/current`, so an atomic release switch also
switches the declared graph.

## Adapters

Adapters only populate the model. Initial loading supports:

- `systemd.unit`: bounded `systemctl show` properties, with staged-root unit
  presence as the offline test boundary;
- `metadata.file`: presence, size, and permissions without reading secrets;
- `release.current`: the active release symlink;
- `observation.file`: a standardized timestamped observation envelope;
- `legacy.json`: a bounded transitional JSON reader with explicit field and
  health mappings;
- `pattern` and `static`: immutable declared structure, optionally with an
  explicit health assertion for partial or intentionally degraded coverage.

Initial rendering and refresh do not execute implementation commands, source
environment files, or make network requests.

## Evidence

Doctor is an ordinary observation attached to a doctor component:

```json
{
  "schema": "muster.observation/v1",
  "implementation": "dvd-ingester",
  "component": "doctor",
  "scope": "runtime",
  "health": "healthy",
  "status": "complete",
  "summary": "All required runtime checks passed",
  "observed_at": "2026-07-11T01:20:00Z",
  "duration_ms": 821,
  "valid_for_seconds": 7200,
  "checks": [
    {
      "id": "destination-mounted",
      "health": "healthy",
      "summary": "Cold destination is mounted and writable"
    }
  ],
  "artifacts": []
}
```

Observations are written atomically under
`/run/muster/<implementation>/observations/`. Stale evidence becomes unknown.
Check evidence and artifact metadata must not contain secrets.

## Actions

Actions are advertised objects, not arbitrary menu callbacks. Commands are
argument arrays and never shell fragments. The first core executes only
`doctor.run`, only after an explicit operator command or confirmed keypress. Future
mutating actions require their own authorization, confirmation, and audit
contract; advertising an action does not authorize its execution.

The bundled doctors write root-owned evidence under `/run/muster`, so their
actions declare `requires_root`. Run them with `sudo muster doctor <ID>` (or
launch the inspector with equivalent privilege); reading existing evidence is
always unprivileged.
