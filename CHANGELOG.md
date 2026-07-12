# Changelog

All notable changes to the shared Muster core are documented here.

## 0.1.2 - 2026-07-11

### Changed

- Rebuilt the terminal inspector around implementation cards and operational
  triage.
- Added healthy-subtree folding with automatic expansion of unhealthy paths.
- Added a live cross-implementation filter with match highlighting and lineage.
- Added viewport scrolling, proportional scrollbars, page keys, and mouse-wheel
  support to detail and help views.
- Added navigable tables for observation checks and metadata.
- Prioritized verdicts, health causes, and current evidence while omitting
  undeclared placeholder sections.

## 0.1.1 - 2026-07-11

### Changed

- Polished the full-screen inspector with clearer visual hierarchy, live status
  chrome, contextual key hints, and health-aware color.
- Added Bubbles-powered keys and activity indication plus Harmonica-smoothed
  navigation and inspector transitions.
- Added the `MUSTER_REDUCE_MOTION=1` accessibility control.
- Presented doctor confirmation as a centered modal, including safe rendering
  at the minimum supported terminal size.
- Updated the Charm TUI dependency set and the minimum Go version to 1.25.

## 0.1.0 - 2026-07-10

### Added

- Shared host inspector, object graph, CLI, and full-screen TUI.
- Schema 2 component manifests and deterministic release locks.
- Recursive health, typed graph edges, evidence observations, and literate
  object explanations.
- Globally addressable components, actions, and observations with preserved
  declared-versus-effective health causality.
- Atomic doctor evidence that remains visible after failed checks and becomes
  unknown when stale.
- Serialized, immutable implementation release transactions with atomic
  current/registration/unit rollback.
- Checksum-verified, independently versioned core bootstrap and registration.
- Cross-compiled Linux release packages and staged multi-service lifecycle
  verification.
