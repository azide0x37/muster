import React from "react";
import {
  AbsoluteFill,
  Easing,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { C, F } from "./theme";
import { ensureStencil } from "./load-font";

/* ---------- helpers ---------- */

export const typed = (text: string, frame: number, start: number, cps = 40) =>
  frame <= start
    ? ""
    : text.slice(0, Math.max(0, Math.floor(((frame - start) / 30) * cps)));

/* Entrances decelerate (ease-out) — never linear. Apple-style: short travel,
   settled quickly, below the strobe threshold. */
export const fadeUp = (
  frame: number,
  start: number,
  dur = 12,
  dist = 14
): React.CSSProperties => {
  const p = interpolate(frame, [start, start + dur], [0, 1], {
    easing: Easing.out(Easing.cubic),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  return { opacity: p, transform: `translateY(${(1 - p) * dist}px)` };
};

/* Critically damped by default (damping 2·√k — no overshoot). Content that
   didn't arrive with momentum doesn't bounce. */
export const pop = (
  frame: number,
  fps: number,
  start: number,
  damping = 22
): number =>
  frame < start
    ? 0
    : spring({ frame: frame - start, fps, config: { damping, stiffness: 120 } });

/* ---------- backdrop ---------- */

const NOISE_URI =
  "url(\"data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")";

export const Shell: React.FC<{ children: React.ReactNode; section?: string }> = ({
  children,
  section,
}) => {
  ensureStencil();
  return (
    <AbsoluteFill style={{ background: C.bg, fontFamily: F.sans, color: C.fg }}>
      <AbsoluteFill
        style={{
          backgroundImage: `linear-gradient(to right, ${C.gridLine} 1px, transparent 1px), linear-gradient(to bottom, ${C.gridLine} 1px, transparent 1px)`,
          backgroundSize: "56px 56px",
          opacity: 0.5,
        }}
      />
      {children}
      <AbsoluteFill style={{ backgroundImage: NOISE_URI, opacity: 0.02, pointerEvents: "none" }} />
      {section ? <Chrome section={section} /> : null}
    </AbsoluteFill>
  );
};

/* ---------- brand ---------- */

export const ChevronMark: React.FC<{ size?: number }> = ({ size = 30 }) => (
  <svg width={size} height={size} viewBox="0 0 100 100">
    <path
      fill={C.steel}
      fillRule="evenodd"
      d="M8 6 L50 40 L92 6 L92 52 L50 92 L8 52 Z M20 32 L50 57 L80 32 L80 46 L50 78 L20 46 Z"
    />
    <path fill={C.accent} d="M34 57 L45 66 L71 36 L76 44 L46 77 L30 65 Z" />
  </svg>
);

export const Chrome: React.FC<{ section: string }> = ({ section }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = Math.floor(frame / fps);
  const pad = (n: number) => String(n).padStart(2, "0");
  const led = 0.55 + 0.45 * Math.abs(Math.sin(frame / 22));
  const mono: React.CSSProperties = {
    fontFamily: F.mono,
    fontSize: 15,
    letterSpacing: "0.16em",
    textTransform: "uppercase",
    color: C.muted,
  };
  return (
    <>
      <div
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          right: 0,
          height: 62,
          display: "flex",
          alignItems: "center",
          gap: 16,
          padding: "0 36px",
          borderBottom: `1px solid ${C.lineSoft}`,
          background: "oklch(0.145 0.01 255 / 0.82)",
        }}
      >
        <ChevronMark size={28} />
        <span style={{ fontFamily: F.stencil, fontSize: 26, letterSpacing: "0.08em", color: C.fg }}>
          MUSTER
        </span>
        <span style={{ flex: 1 }} />
        <span style={{ ...mono, color: C.faint }}>FIELD MANUAL · 90 SEC</span>
      </div>
      <div
        style={{
          position: "absolute",
          bottom: 0,
          left: 0,
          right: 0,
          height: 44,
          display: "flex",
          alignItems: "center",
          gap: 26,
          padding: "0 36px",
          borderTop: `1px solid ${C.lineSoft}`,
          background: "oklch(0.145 0.01 255 / 0.86)",
          ...mono,
          fontSize: 13,
        }}
      >
        <span style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span
            style={{
              width: 9,
              height: 9,
              borderRadius: "50%",
              background: C.good,
              boxShadow: `0 0 10px ${C.good}`,
              opacity: led,
            }}
          />
          muster-explainer.service
          <span style={{ color: C.faint }}>active (running)</span>
        </span>
        <span style={{ color: C.faint }}>
          up 00:{pad(Math.floor(s / 60))}:{pad(s % 60)}
        </span>
        <span style={{ flex: 1 }} />
        <span style={{ color: C.accent }}>{section}</span>
        <span style={{ color: C.faint }}>1 take · 0 external requests</span>
      </div>
    </>
  );
};

/* ---------- type ---------- */

export const Kicker: React.FC<{ idx: string; text: string; style?: React.CSSProperties }> = ({
  idx,
  text,
  style,
}) => (
  <div
    style={{
      fontFamily: F.mono,
      fontSize: 17,
      letterSpacing: "0.3em",
      textTransform: "uppercase",
      color: C.accent,
      display: "flex",
      alignItems: "center",
      gap: 18,
      ...style,
    }}
  >
    <span style={{ color: C.muted }}>{idx}</span>
    <span>{text}</span>
    <span
      style={{
        height: 1,
        width: 340,
        background: `linear-gradient(to right, ${C.accent}, transparent)`,
        opacity: 0.5,
      }}
    />
  </div>
);

export const Stencil: React.FC<{
  children: React.ReactNode;
  size?: number;
  color?: string;
  style?: React.CSSProperties;
}> = ({ children, size = 96, color = C.fg, style }) => (
  <div
    style={{
      fontFamily: F.stencil,
      fontSize: size,
      lineHeight: 0.95,
      textTransform: "uppercase",
      letterSpacing: "0",
      color,
      ...style,
    }}
  >
    {children}
  </div>
);

/* ---------- terminal ---------- */

export const TermPanel: React.FC<{
  title: string;
  right?: string;
  width?: number | string;
  minHeight?: number;
  children: React.ReactNode;
  accent?: string;
  style?: React.CSSProperties;
}> = ({ title, right, width, minHeight, children, accent, style }) => (
  <div
    style={{
      border: `1px solid ${C.line}`,
      borderTop: accent ? `3px solid ${accent}` : `1px solid ${C.line}`,
      background: C.termBg,
      width,
      ...style,
    }}
  >
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 18px",
        borderBottom: `1px solid ${C.lineSoft}`,
        fontFamily: F.mono,
        fontSize: 13,
        letterSpacing: "0.2em",
        textTransform: "uppercase",
        color: C.muted,
      }}
    >
      <span style={{ display: "flex", gap: 6 }}>
        {[0, 1, 2].map((i) => (
          <span key={i} style={{ width: 9, height: 9, borderRadius: "50%", background: C.line }} />
        ))}
      </span>
      {title}
      {right ? <span style={{ marginLeft: "auto", color: C.faint }}>{right}</span> : null}
    </div>
    <div
      style={{
        padding: "16px 20px",
        fontFamily: F.mono,
        fontSize: 17,
        lineHeight: 1.75,
        minHeight,
      }}
    >
      {children}
    </div>
  </div>
);

export const T: React.FC<{ c?: "ok" | "dim" | "acc" | "bad" | "fg" | "cmt"; children: React.ReactNode }> = ({
  c = "fg",
  children,
}) => {
  const map = { ok: C.good, dim: C.faint, acc: C.accent, bad: C.bad, fg: C.fg, cmt: C.faint };
  return <span style={{ color: map[c] }}>{children}</span>;
};

/* ---------- instruments ---------- */

export const Gate: React.FC<{ label: string; state: "" | "live" | "pass" | "fail" }> = ({
  label,
  state,
}) => {
  const color = state === "pass" ? C.good : state === "fail" ? C.bad : state === "live" ? C.fg : C.faint;
  const border = state === "pass" ? C.good : state === "fail" ? C.bad : state === "live" ? C.accent : C.lineSoft;
  const dot = state === "pass" ? C.good : state === "fail" ? C.bad : state === "live" ? C.accent : C.line;
  return (
    <span
      style={{
        fontFamily: F.mono,
        fontSize: 13,
        letterSpacing: "0.16em",
        textTransform: "uppercase",
        border: `1px solid ${border}`,
        color,
        padding: "7px 14px",
        display: "inline-flex",
        alignItems: "center",
        gap: 9,
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: "50%",
          background: dot,
          boxShadow: state === "live" ? `0 0 9px ${C.accent}` : undefined,
        }}
      />
      {label}
    </span>
  );
};

export const Stamp: React.FC<{ text: string; color?: string; at: number; fontSize?: number }> = ({
  text,
  color = C.good,
  at,
  fontSize = 44,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  if (frame < at) return null;
  // A stamp arrives with momentum — one confident overshoot, no jiggle.
  const s = spring({ frame: frame - at, fps, config: { damping: 18, stiffness: 200 } });
  const scale = interpolate(s, [0, 1], [1.7, 1]);
  return (
    <div
      style={{
        fontFamily: F.stencil,
        fontSize,
        letterSpacing: "0.08em",
        textTransform: "uppercase",
        border: `4px solid ${color}`,
        color,
        padding: "8px 26px 5px",
        transform: `rotate(-8deg) scale(${scale})`,
        opacity: Math.min(s * 1.4, 0.94),
        display: "inline-block",
      }}
    >
      {text}
    </div>
  );
};
