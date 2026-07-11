import React, { useEffect, useMemo, useRef } from "react";
import {
  interpolate,
  random,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "../theme";
import { Stencil, fadeUp, typed } from "../ui";

/* Chevron + check geometry, same polygons as the site's offscreen sampler,
   but computed analytically so every frame is a pure function. */
type Pt = { x: number; y: number; check: boolean };

const OUTER = [[8, 6], [50, 40], [92, 6], [92, 52], [50, 92], [8, 52]] as const;
const INNER = [[20, 32], [50, 57], [80, 32], [80, 46], [50, 78], [20, 46]] as const;
const CHECK = [[34, 57], [45, 66], [71, 36], [76, 44], [46, 77], [30, 65]] as const;

const inPoly = (px: number, py: number, poly: readonly (readonly number[])[]) => {
  let inside = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [xi, yi] = poly[i];
    const [xj, yj] = poly[j];
    if (yi > py !== yj > py && px < ((xj - xi) * (py - yi)) / (yj - yi) + xi) {
      inside = !inside;
    }
  }
  return inside;
};

const samplePoints = (): Pt[] => {
  const pts: Pt[] = [];
  const step = 1.55;
  for (let y = 2; y < 98; y += step) {
    for (let x = 2; x < 98; x += step) {
      const check = inPoly(x, y, CHECK);
      const band = !check && inPoly(x, y, OUTER) && !inPoly(x, y, INNER);
      if (check || band) pts.push({ x: x / 100, y: y / 100, check });
    }
  }
  return pts;
};

export const HeroScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const pts = useMemo(samplePoints, []);
  const order = useMemo(
    () =>
      pts
        .map((_, i) => ({ i, r: random(`ord${i}`) }))
        .sort((a, b) => a.r - b.r)
        .map((o) => o.i),
    [pts]
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.clearRect(0, 0, width, height);
    const size = Math.min(width, height) * 0.74;
    const cx = width * 0.71;
    const cy = height * 0.47;
    for (let k = 0; k < pts.length; k++) {
      const i = order[k];
      const p = pts[i];
      const delay = (k / pts.length) * 55 + random(`d${i}`) * 10;
      const s = spring({
        frame: Math.max(0, frame - delay),
        fps,
        config: { damping: 12, stiffness: 60, mass: 1 },
      });
      const sx = random(`sx${i}`) * width;
      const sy = random(`sy${i}`) * height;
      const hx = cx + (p.x - 0.5) * size + Math.sin(frame * 0.013 + i) * 2.4;
      const hy = cy + (p.y - 0.5) * size + Math.cos(frame * 0.011 + i * 1.7) * 2.4;
      const x = sx + (hx - sx) * s;
      const y = sy + (hy - sy) * s;
      const twk = 0.75 + 0.25 * Math.sin(frame * 0.03 * (0.4 + random(`tw${i}`) * 1.4) + i);
      ctx.globalAlpha = (p.check ? 0.95 : 0.62) * twk;
      ctx.fillStyle = p.check ? "#eb9a30" : "#aeb3bd";
      ctx.fillRect(x, y, 3.4, 3.4);
    }
    ctx.globalAlpha = 1;
  }, [frame, fps, width, height, pts, order]);

  const titleS = spring({ frame: Math.max(0, frame - 26), fps, config: { damping: 20, stiffness: 90 } });

  return (
    <>
      <canvas ref={canvasRef} width={width} height={height} style={{ position: "absolute", inset: 0 }} />
      <div style={{ position: "absolute", left: 110, top: 200, maxWidth: 980 }}>
        <div
          style={{
            ...fadeUp(frame, 8),
            display: "inline-flex",
            alignItems: "center",
            gap: 14,
            border: `1px solid ${C.line}`,
            padding: "10px 20px",
            fontFamily: F.mono,
            fontSize: 16,
            letterSpacing: "0.32em",
            textTransform: "uppercase",
            color: C.accent,
            background: "oklch(0.145 0.01 255 / 0.72)",
          }}
        >
          <span style={{ width: 9, height: 9, borderRadius: "50%", background: C.good, boxShadow: `0 0 9px ${C.good}` }} />
          REPO SCAFFOLD FRAMEWORK · FIELD MANUAL
        </div>
        <Stencil
          size={236}
          style={{
            marginTop: 26,
            opacity: titleS,
            transform: `scale(${interpolate(titleS, [0, 1], [1.07, 1])})`,
            transformOrigin: "left center",
          }}
        >
          Muster
        </Stencil>
        <div
          style={{
            marginTop: 20,
            fontFamily: F.mono,
            fontSize: 27,
            color: C.fg,
            background: "oklch(0.145 0.01 255 / 0.62)",
            width: "fit-content",
          }}
        >
          {typed("A repo scaffold framework for ", frame, 62, 46)}
          <span style={{ color: C.accent }}>
            {typed("small Linux service appliances.", frame, 82, 46)}
          </span>
        </div>
        <div
          style={{
            ...fadeUp(frame, 120),
            marginTop: 16,
            fontSize: 23,
            color: C.muted,
            maxWidth: 760,
            background: "oklch(0.145 0.01 255 / 0.62)",
          }}
        >
          Built once. Forgotten on purpose. Still standing inspection.
        </div>
        <div style={{ display: "flex", gap: 12, marginTop: 34, flexWrap: "wrap" }}>
          {["systemd-native", "idempotent installs", "sha256 + rollback", "doctor.sh health", "self-certifying"].map(
            (t, i) => (
              <span
                key={t}
                style={{
                  ...fadeUp(frame, 142 + i * 9, 12),
                  fontFamily: F.mono,
                  fontSize: 14,
                  letterSpacing: "0.2em",
                  textTransform: "uppercase",
                  color: C.muted,
                  border: `1px solid ${C.lineSoft}`,
                  padding: "8px 14px",
                  background: "oklch(0.145 0.01 255 / 0.7)",
                }}
              >
                {t} <span style={{ color: C.good }}>●</span>
              </span>
            )
          )}
        </div>
      </div>
    </>
  );
};
