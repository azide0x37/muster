import React from "react";
import { interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import { PATTERNS } from "../patterns";
import { C, F } from "../theme";
import { Kicker, Stencil, fadeUp, pop } from "../ui";
import { RARITY } from "../theme";

const byId = Object.fromEntries(PATTERNS.map((p) => [p.id, p]));
const SEL = "T2R4.device-triggered-conveyor";
const sel = byId[SEL];

const T2_ROW = [SEL, "T2C1.hot-cold-nas-conveyor", "T2C3.scheduled-herald"];
const T1_ROW = ["C1.service-capsule", "C2.persistent-tick", "C4.lazy-resource-gate", "C5.failure-ratchet", "C6.lifecycle-capsule", "R2.device-binding", "R5.capability-mount"];

const GW = 1700;
const Y2 = 104;
const Y1 = 356;

const posOf = (id: string): { x: number; y: number } => {
  const i2 = T2_ROW.indexOf(id);
  if (i2 >= 0) return { x: GW / 2 + (i2 - 1) * 280, y: Y2 };
  const i1 = T1_ROW.indexOf(id);
  return { x: GW / 2 + (i1 - 3) * 225, y: Y1 };
};

const short = (id: string) => id.split(".")[0];

type Edge = { a: string; b: string; hot: boolean; at: number };
const EDGES: Edge[] = [
  ...sel.subs
    .filter((s) => T2_ROW.includes(s) || T1_ROW.includes(s))
    .map((s, i) => ({ a: SEL, b: s, hot: true, at: 70 + i * 5 })),
  ...(["T2C1.hot-cold-nas-conveyor", "T2C3.scheduled-herald"] as const).flatMap((t2, k) =>
    byId[t2].subs
      .filter((s) => T1_ROW.includes(s))
      .map((s, i) => ({ a: t2, b: s, hot: false, at: 140 + k * 14 + i * 5 }))
  ),
];

export const PatternsScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const nodeAt = (id: string) =>
    id === SEL ? 40 : T2_ROW.includes(id) ? 52 + T2_ROW.indexOf(id) * 8 : 88 + T1_ROW.indexOf(id) * 7;

  const mrlFill = (i: number) => frame >= 246 + i * 4;
  const count = Math.round(
    interpolate(frame, [290, 340], [0, PATTERNS.length], { extrapolateLeft: "clamp", extrapolateRight: "clamp" })
  );

  return (
    <div style={{ position: "absolute", inset: 0, padding: "130px 110px" }}>
      <div style={fadeUp(frame, 0)}>
        <Kicker idx="§03" text="The Muster Pattern Library" />
      </div>
      <Stencil size={92} style={{ marginTop: 18, ...fadeUp(frame, 6) }}>
        A vocabulary of operational shapes
      </Stencil>

      <div style={{ position: "relative", marginTop: 34, border: `1px solid ${C.line}`, background: C.bg2, ...fadeUp(frame, 16) }}>
        <svg viewBox={`0 0 ${GW} 452`} style={{ display: "block", width: "100%" }}>
          {/* row labels */}
          <text x={30} y={Y2 + 6} fill={C.faint} fontSize={15} fontFamily={F.mono} letterSpacing={3}>T2</text>
          <text x={30} y={Y1 + 6} fill={C.faint} fontSize={15} fontFamily={F.mono} letterSpacing={3}>T1</text>

          {EDGES.map((e, i) => {
            const A = posOf(e.a);
            const B = posOf(e.b);
            const sameRow = A.y === B.y;
            const d = sameRow
              ? `M ${A.x} ${A.y - 26} C ${A.x} ${A.y - 92}, ${B.x} ${B.y - 92}, ${B.x} ${B.y - 26}`
              : `M ${A.x} ${A.y + 26} C ${A.x} ${A.y + 110}, ${B.x} ${B.y - 110}, ${B.x} ${B.y - 26}`;
            const len = Math.hypot(B.x - A.x, B.y - A.y) * 1.3 + 60;
            const drawn = interpolate(frame, [e.at, e.at + 26], [len, 0], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            });
            return (
              <path
                key={i}
                d={d}
                fill="none"
                stroke={e.hot ? C.accent : C.line}
                strokeWidth={e.hot ? 2.4 : 1.4}
                opacity={e.hot ? 0.9 : 0.6}
                strokeDasharray={len}
                strokeDashoffset={drawn}
              />
            );
          })}

          {[...T2_ROW, ...T1_ROW].map((id) => {
            const p = byId[id];
            const P = posOf(id);
            const isSel = id === SEL;
            const s = pop(frame, fps, nodeAt(id));
            const w = isSel ? 150 : 110;
            return (
              <g key={id} transform={`translate(${P.x} ${P.y}) scale(${0.6 + 0.4 * s})`} opacity={s}>
                <rect
                  x={-w / 2}
                  y={-26}
                  width={w}
                  height={52}
                  fill={isSel ? C.accentSoft : C.bg}
                  stroke={isSel ? C.accent : RARITY[p.rarity]}
                  strokeWidth={isSel ? 3 : 1.8}
                />
                <text x={0} y={7} textAnchor="middle" fill={isSel ? C.accent : C.fg} fontSize={20} fontFamily={F.mono} letterSpacing={1.5}>
                  {short(id)}
                </text>
                <text x={0} y={50} textAnchor="middle" fill={C.muted} fontSize={12.5} fontFamily={F.mono} letterSpacing={0.8}>
                  {p.name.toLowerCase()}
                </text>
              </g>
            );
          })}
        </svg>

        {/* detail strip */}
        <div
          style={{
            ...fadeUp(frame, 206),
            display: "flex",
            gap: 40,
            alignItems: "center",
            borderTop: `1px solid ${C.lineSoft}`,
            background: C.bg,
            padding: "20px 30px",
          }}
        >
          <div>
            <div style={{ fontFamily: F.mono, fontSize: 14, letterSpacing: "0.2em", color: C.accent, textTransform: "uppercase" }}>{sel.id}</div>
            <div style={{ fontFamily: F.stencil, fontSize: 40, marginTop: 2 }}>{sel.name}</div>
          </div>
          <div style={{ flex: 1, color: C.muted, fontSize: 18, lineHeight: 1.5, maxWidth: 780 }}>{sel.summary}</div>
          <div style={{ display: "grid", gap: 8, justifyItems: "end", fontFamily: F.mono, fontSize: 13, letterSpacing: "0.14em", textTransform: "uppercase", color: C.muted }}>
            <span style={{ border: `1px solid ${C.rRare}`, color: C.rRare, padding: "3px 10px", letterSpacing: "0.2em" }}>rare</span>
            <span>tech II · stable · 9 subpatterns</span>
            <span style={{ display: "inline-flex", gap: 4 }}>
              mrl
              {Array.from({ length: 9 }, (_, i) => (
                <i key={i} style={{ width: 12, height: 17, background: mrlFill(i) && i < sel.mrl ? C.accent : C.lineSoft, display: "inline-block" }} />
              ))}
            </span>
          </div>
        </div>
      </div>

      <div style={{ display: "flex", gap: 14, marginTop: 20, ...fadeUp(frame, 292) }}>
        {[`${count} patterns`, "tech I → III", "common · rare · mythic", "generated from manifests"].map((t, i) => (
          <span
            key={i}
            style={{
              fontFamily: F.mono,
              fontSize: 15,
              letterSpacing: "0.18em",
              textTransform: "uppercase",
              border: `1px solid ${C.line}`,
              color: i === 0 ? C.accent : C.muted,
              padding: "9px 16px",
              background: C.bg2,
            }}
          >
            {t}
          </span>
        ))}
      </div>
    </div>
  );
};
