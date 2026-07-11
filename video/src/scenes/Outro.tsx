import React from "react";
import { interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";
import { ChevronMark, Kicker, Stencil, fadeUp } from "../ui";

const LAWS: Array<[string, string]> = [
  ["01", "systemd owns service lifecycle."],
  ["05", "Installers are idempotent — run twice, converge twice, exit 0."],
  ["06", "Updates verify SHA256, then earn their keep."],
  ["08", "The README self-certifies, with evidence."],
];

export const OutroScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const lawsOut = interpolate(frame, [168, 192], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  const cardIn = frame >= 186;
  const logoS = cardIn ? spring({ frame: frame - 190, fps, config: { damping: 20, stiffness: 90 } }) : 0;

  return (
    <div style={{ position: "absolute", inset: 0, padding: "130px 110px" }}>
      {/* phase 1: the inspection sheet */}
      <div style={{ opacity: lawsOut }}>
        <div style={fadeUp(frame, 0)}>
          <Kicker idx="§06" text="The contract · MUSTER.md is the law book" />
        </div>
        <Stencil size={92} style={{ marginTop: 18, ...fadeUp(frame, 6) }}>
          The inspection sheet
        </Stencil>
        <div style={{ marginTop: 40, borderTop: `1px solid ${C.line}`, maxWidth: 1300 }}>
          {LAWS.map(([n, txt], i) => {
            const at = 26 + i * 32;
            const stampAt = at + 16;
            const s = frame < stampAt ? 0 : spring({ frame: frame - stampAt, fps, config: { damping: 18, stiffness: 210 } });
            return (
              <div
                key={n}
                style={{
                  ...fadeUp(frame, at, 12, 14),
                  display: "flex",
                  alignItems: "baseline",
                  gap: 34,
                  padding: "22px 6px",
                  borderBottom: `1px solid ${C.line}`,
                }}
              >
                <span style={{ fontFamily: F.stencil, fontSize: 40, color: C.faint, width: 70 }}>{n}</span>
                <span style={{ fontSize: 25, color: C.fg, flex: 1 }}>{txt}</span>
                <span
                  style={{
                    fontFamily: F.mono,
                    fontSize: 13,
                    letterSpacing: "0.2em",
                    textTransform: "uppercase",
                    color: C.good,
                    border: `1.5px solid ${C.good}`,
                    padding: "4px 12px",
                    opacity: Math.min(s * 1.3, 0.92),
                    transform: `rotate(-4deg) scale(${interpolate(s, [0, 1], [1.5, 1])})`,
                    display: "inline-block",
                  }}
                >
                  stamped
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {/* phase 2: end card */}
      {cardIn ? (
        <div
          style={{
            position: "absolute",
            inset: 0,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: 26,
          }}
        >
          <div style={{ transform: `scale(${interpolate(logoS, [0, 1], [1.4, 1])})`, opacity: logoS }}>
            <ChevronMark size={150} />
          </div>
          <Stencil size={150} style={{ opacity: logoS }}>
            Muster
          </Stencil>
          <div
            style={{
              ...fadeUp(frame, 214),
              fontFamily: F.mono,
              fontSize: 20,
              letterSpacing: "0.28em",
              textTransform: "uppercase",
              color: C.muted,
            }}
          >
            small tools · standing inspection
          </div>
          <div
            style={{
              ...fadeUp(frame, 234),
              fontFamily: F.mono,
              fontSize: 22,
              color: C.accent,
              border: `1px solid ${C.accent}`,
              padding: "14px 28px",
              background: C.accentSoft,
              letterSpacing: "0.06em",
            }}
          >
            github.com/azide0x37/muster
          </div>
          <div
            style={{
              ...fadeUp(frame, 262),
              fontFamily: F.mono,
              fontSize: 14,
              letterSpacing: "0.2em",
              textTransform: "uppercase",
              color: C.faint,
            }}
          >
            + muster-pattern-library · 35 shapes, ready to compose
          </div>
        </div>
      ) : null}
    </div>
  );
};
