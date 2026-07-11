// TAPS theme, lifted verbatim from docs/index.html. The video commits to
// night ops — one look, broadcast-safe on dark.
export const C = {
  bg: "oklch(0.145 0.01 255)",
  bg2: "oklch(0.175 0.012 255)",
  bg3: "oklch(0.21 0.014 255)",
  fg: "oklch(0.94 0.005 95)",
  muted: "oklch(0.67 0.012 255)",
  faint: "oklch(0.45 0.012 255)",
  line: "oklch(0.28 0.014 255)",
  lineSoft: "oklch(0.23 0.012 255)",
  accent: "oklch(0.76 0.155 65)",
  accentSoft: "oklch(0.76 0.155 65 / 0.13)",
  steel: "oklch(0.78 0.012 255)",
  steelDim: "oklch(0.6 0.012 255)",
  good: "oklch(0.75 0.16 150)",
  goodSoft: "oklch(0.75 0.16 150 / 0.12)",
  bad: "oklch(0.64 0.2 27)",
  badSoft: "oklch(0.64 0.2 27 / 0.12)",
  warn: "oklch(0.8 0.13 85)",
  rCommon: "oklch(0.72 0.012 255)",
  rRare: "oklch(0.8 0.125 85)",
  rMythic: "oklch(0.66 0.195 35)",
  termBg: "oklch(0.12 0.008 255)",
  gridLine: "oklch(0.22 0.012 255)",
} as const;

export const F = {
  stencil: '"Muster Stencil", "Arial Narrow", sans-serif',
  mono: 'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, "DejaVu Sans Mono", monospace',
  sans: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
} as const;

export const RARITY: Record<string, string> = {
  common: C.rCommon,
  rare: C.rRare,
  mythic: C.rMythic,
};
