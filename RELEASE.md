# Muster 0.1.1

Muster 0.1.1 makes the shared host inspector feel at home in the terminal while
preserving its single, addressable runtime object graph.

## Changed

- Refined the full-screen inspector with a colorful wordmark, clearer panel
  titles, health-aware counts, a segmented status bar, and contextual key hints.
- Added spring-smoothed navigation, an animated activity indicator, and a
  breathing fleet-health signal using Bubbles and Harmonica.
- Added `MUSTER_REDUCE_MOTION=1` for an entirely still console.
- Made doctor confirmation an unmistakable modal that remains within the
  smallest supported terminal.
- Updated Bubble Tea, Bubbles, Lip Gloss, and the supporting Charm libraries.

## Operational contract

- Installing the first Muster implementation installs the shared inspector.
- Later implementations register without replacing or downgrading that core.
- An implementation uninstaller removes only its own registration.
- The inspector is read-only during initial rendering; doctor execution is an
  explicit advertised action that declares and enforces its root requirement.
- Failed doctors still publish and refresh structured evidence; stale evidence
  is rendered as unknown rather than reusing an old green result.
- Recursive health exports retain both declared and effective values so
  `muster explain` can show the actual component and graph path responsible.
- Core releases remain checksum-verified, immutable, and independently
  versioned from implementation releases.
