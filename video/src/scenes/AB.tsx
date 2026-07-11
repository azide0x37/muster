import React from "react";
import { spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";
import { Kicker, Stencil, T, TermPanel, fadeUp } from "../ui";

const A_LINES: React.ReactNode[] = [
  <><T>$ ssh pi</T></>,
  <><T>$ history | grep -i install</T> <T c="cmt"># nothing</T></>,
  <><T>$ crontab -l</T> <T c="cmt"># 11 lines · 3 authors · 0 comments</T></>,
  <><T>$ ps aux | grep rip</T> <T c="cmt"># something is running</T></>,
  <><T c="bad"># …leave it alone. it might be load-bearing.</T></>,
];

const B_LINES: React.ReactNode[] = [
  <><T>$ cat README.md</T> <T c="cmt"># self-certification, with evidence</T></>,
  <><T>$ cat muster.yaml</T> <T c="cmt"># patterns: T2R4 · T2R5 · T2R6</T></>,
  <><T>$ systemctl status 'dvd-*'</T> <T c="cmt"># every unit named and owned</T></>,
  <><T>$ sudo bin/doctor.sh</T></>,
  <><T c="ok">doctor: 12/12 checks pass</T></>,
];

const Verdict: React.FC<{ color: string; text: string; at: number }> = ({ color, text, at }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = frame < at ? 0 : spring({ frame: frame - at, fps, config: { damping: 26, stiffness: 160 } });
  return (
    <div
      style={{
        borderTop: `1px solid ${C.lineSoft}`,
        padding: "14px 20px",
        fontFamily: F.mono,
        fontSize: 18,
        color,
        opacity: s,
        transform: `translateX(${(1 - s) * -20}px)`,
      }}
    >
      ▸ {text}
    </div>
  );
};

export const ABScene: React.FC = () => {
  const frame = useCurrentFrame();
  return (
    <div style={{ position: "absolute", inset: 0, padding: "130px 110px" }}>
      <div style={fadeUp(frame, 0)}>
        <Kicker idx="§04" text="A/B inspection · same idea, two futures" />
      </div>
      <div style={{ display: "flex", alignItems: "baseline", gap: 40 }}>
        <Stencil size={92} style={{ marginTop: 18, ...fadeUp(frame, 6) }}>
          Six months later
        </Stencil>
        <span style={{ ...fadeUp(frame, 14), fontFamily: F.mono, fontSize: 16, letterSpacing: "0.14em", color: C.faint, textTransform: "uppercase" }}>
          A: “make me a script” · B: “produce the operational harness”
        </span>
      </div>

      <div style={{ display: "flex", gap: 26, marginTop: 44 }}>
        <div style={{ flex: 1, ...fadeUp(frame, 24) }}>
          <TermPanel title="A · the script" right="the dig begins" accent={C.bad} minHeight={330}>
            {A_LINES.map((l, i) => (
              <div key={i} style={{ ...fadeUp(frame, 56 + i * 26, 10, 8), fontSize: 19, lineHeight: 2 }}>
                {l}
              </div>
            ))}
          </TermPanel>
          <div style={{ border: `1px solid ${C.line}`, borderTop: "none", background: C.termBg }}>
            <Verdict color={C.bad} text="Congratulations: it’s archaeology." at={230} />
          </div>
        </div>

        <div style={{ flex: 1, ...fadeUp(frame, 36) }}>
          <TermPanel title="B · the muster repo" right="the repo explains itself" accent={C.good} minHeight={330}>
            {B_LINES.map((l, i) => (
              <div key={i} style={{ ...fadeUp(frame, 72 + i * 26, 10, 8), fontSize: 19, lineHeight: 2 }}>
                {l}
              </div>
            ))}
          </TermPanel>
          <div style={{ border: `1px solid ${C.line}`, borderTop: "none", background: C.termBg }}>
            <Verdict color={C.good} text="Five minutes of reading and you own it again." at={252} />
          </div>
        </div>
      </div>
    </div>
  );
};
