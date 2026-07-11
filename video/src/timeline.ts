// Single source of truth for pacing. After you record the narration,
// re-time scenes here (frames = seconds × FPS) — nothing else moves.
export const FPS = 30;
export const W = 1920;
export const H = 1080;

/* Pacing tuned to an ~117 wpm deadpan read: shorter setup scenes, longer
   holds where a stamp needs to land (rail rollback, end card). */
const DUR = {
  hero: 240, //     0:00–0:08  cold open
  problem: 315, //  0:08–0:18  server archaeology
  shape: 345, //    0:18–0:30  the load-bearing shape
  rail: 690, //     0:30–0:53  the update rail (good ship, then bad)
  patterns: 375, // 0:53–1:05  pattern library
  ab: 345, //       1:05–1:17  six months later, A/B
  outro: 390, //    1:17–1:30  contract + end card, long confident hold
} as const;

export type SceneKey = keyof typeof DUR;

export const SCENES = (() => {
  let at = 0;
  const out = {} as Record<SceneKey, { from: number; dur: number }>;
  (Object.keys(DUR) as SceneKey[]).forEach((k) => {
    out[k] = { from: at, dur: DUR[k] };
    at += DUR[k];
  });
  return out;
})();

export const TOTAL_FRAMES = Object.values(DUR).reduce((a, b) => a + b, 0);

export const SECTION_LABEL: Record<SceneKey, string> = {
  hero: "§00 · MUSTER",
  problem: "§01 · SERVER ARCHAEOLOGY",
  shape: "§02 · THE SHAPE",
  rail: "§02 · THE UPDATE RAIL",
  patterns: "§03 · PATTERN LIBRARY",
  ab: "§04 · A/B INSPECTION",
  outro: "§06 · THE CONTRACT",
};

// Flip to true once public/vo.wav exists (see SCRIPT.md).
export const WITH_VO = false;
export const VO_FILE = "vo.wav";
