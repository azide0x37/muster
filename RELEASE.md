# Muster 0.1.2

Muster 0.1.2 reorganizes the shared host inspector around operational triage
while preserving the same normalized, addressable object graph.

## Changed

- Replaced the flat implementation list with implementation cards that expose
  health, version, selection, and visible-object counts directly in the frame.
- Folded fully healthy subtrees by default while automatically opening paths to
  degraded, unhealthy, or unknown objects.
- Added a live `/` filter across implementation cards with lineage-preserving
  matches and highlighted query text.
- Added viewport-backed scrolling, mouse-wheel support, page navigation, and
  proportional scrollbars to overview, inspect, and help panes.
- Rendered observation checks and metadata as aligned, keyboard-navigable
  tables.
- Reworked overview and inspect views to lead with verdicts, health causes, and
  fresh evidence before literate context.
- Removed redundant or undeclared placeholder sections so simple components
  remain concise and factual.

## Operational contract

- The object model, CLI commands, registration schema, and observation format
  remain compatible with 0.1.x implementations.
- Installing the first Muster implementation installs the shared inspector.
- Later implementations register without replacing or downgrading that core.
- Core releases remain checksum-verified, immutable, and independently
  versioned from implementation releases.
