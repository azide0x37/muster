# MUSTER — Explainer Narration Script

**Runtime:** 90 seconds · 2,700 frames @ 30 fps · ~175 words · ~117 wpm
**Voice direction:** dry, deadpan, military-documentary — but *calm*, not
clipped. Decide the emotion and hold it: quiet confidence. You are reading an
inspection report that happens to be funny; do not perform the jokes — land
them flat and leave silence after them. The pacing budget deliberately
under-fills every scene: when in doubt, pause longer. Read paths as words:
`/usr/local/bin` → "user-local-bin", `/etc` → "et-see", `/opt` → "opt".

---

## The script

### §00 · COLD OPEN — 0:00–0:08
*On screen: particles muster into the chevron; MUSTER settles in (no bounce).
Let the first phrase start about half a second in.*

> This is Muster — a repo scaffold framework for small Linux service
> appliances. Built once, forgotten on purpose — and still standing
> inspection.

### §01 · THE PROBLEM — 0:08–0:18.5
*On screen: the /usr/local/bin excavation settles layer by layer; three
escalation beats on the right. Pause a full beat after "infrastructure."*

> Here's the trap. A script becomes a service. The service becomes
> infrastructure. Six months later, something in user-local-bin is
> load-bearing.

### §02 · THE SHAPE — 0:18.5–0:30
*On screen: install layout tree types out; the orange sweep hits the
`current` line at ~0:26 — say "symlink" near it, close is fine.*

> Every Muster tool gets one boring shape, owned by systemd. Config in
> et-see. Releases in opt — immutable. And current is a symlink: the only
> moving part.

### §02 · THE UPDATE RAIL — 0:30–0:53
*On screen: gates light with the five verbs — pace them evenly, don't chase.
Good ship certifies (~0:39, PASSES MUSTER stamp — leave a beat of silence).
"Six days later" divider ~0:42. The doctor fails 1.0.5, the symlink flips
back, ROLLED BACK stamps at ~0:50 — then say nothing until the scene ends.*

> Updates ride a rail. Fetch. Verify. Stage. Flip. Restart. Then the doctor
> decides. A good release gets certified. A bad one fails inspection — the
> symlink flips back, and the build is kept for autopsy. Nobody gets paged.

### §03 · THE PATTERN LIBRARY — 0:53–1:05.5
*On screen: T2R4 unfolds into its nine subpatterns; counter runs to 35.*

> Every appliance names its shape in the Muster Pattern Library —
> thirty-five operational patterns, from single-purpose atoms to whole
> machines. Compose them; don't reinvent them.

### §04 · SIX MONTHS LATER — 1:05.5–1:17
*On screen: A/B terminals type in parallel; verdicts slide in ~1:13.*

> Six months later, the script is a dig site. The Muster repo reads its own
> paperwork, proves its own health, and hands you back the keys.

### §06 · OUTRO — 1:17–1:30
*On screen: four contract laws stamp at a measured cadence; crossfade to the
end card at ~1:23. Finish speaking by ~1:26 — the URL holds in silence for
the last four seconds. Do not read the URL.*

> Small tools deserve boring discipline. Muster makes it repeatable.
> Stand your services for inspection — and pass muster.

---

## Pacing philosophy (why it's cut this way)

Tuned against Apple's fluid-interface principles (Emil Kowalski's
apple-design skill): entrances decelerate instead of moving linearly;
springs are critically damped except where the motion carries momentum —
the rubber stamps and the thrown particles keep one confident overshoot,
nothing else bounces. Each scene has one focal movement at a time, and the
two stamp moments plus the end card get long holds. The read should match:
~117 wpm, downward inflection, silence where the picture is doing the work.

## Recording notes

- **Format:** WAV, 48 kHz, mono, -16 to -19 LUFS integrated. Leave 0.5 s of
  room tone at the head and tail.
- **Takes:** one take per section. The rail section is worth three takes —
  the five gate verbs want even, unhurried spacing, and "Nobody gets paged."
  wants to be almost thrown away.
- **Timing slack:** every scene under-fills its slot by 1–3 s. Run long
  rather than rushing; if a section runs *very* long, stretch that scene
  (below) instead of tightening the read.

## Wiring the narration in

1. Drop the file at `video/public/vo.wav`.
2. In [src/timeline.ts](src/timeline.ts), set `WITH_VO = true`.
3. Preview against picture: `npm run dev` (Remotion Studio, scrubbable).
4. Re-time if needed: scene durations live in the `DUR` map in
   [src/timeline.ts](src/timeline.ts) — frames = seconds × 30. Everything
   downstream (sequence starts, the status-bar section label) recomputes
   from that one map. In-scene beats are authored relative to scene start,
   so stretch scenes from the tail.
5. Render: `npm run render` → `out/muster-explainer.mp4`
   (or `make video` from the repo root).

## Renders

- Full video: `npx remotion render MusterExplainer out/muster-explainer.mp4`
- One scene: `npx remotion render Scene-rail out/rail.mp4`
- A still: `npx remotion still MusterExplainer out/frame.png --frame=1600`
- 4K master: add `--scale=2`
