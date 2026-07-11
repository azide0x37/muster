import React from "react";
import { interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { C, F } from "../theme";
import { Gate, Kicker, Stamp, Stencil, T, TermPanel, fadeUp, pop } from "../ui";

type GateState = "" | "live" | "pass" | "fail";
type RelState = "hidden" | "staging" | "ok" | "condemned";

type State = {
  gates: Record<string, GateState>;
  log: Array<{ f: number; node: React.ReactNode }>;
  current: string;
  lastFlip: number;
  r104: RelState;
  r105: RelState;
};

const GATES = ["fetch", "sha256", "stage", "flip", "restart", "doctor"] as const;

type Ev = { f: number; log?: React.ReactNode; set?: Partial<Omit<State, "log" | "gates">>; gates?: Partial<Record<string, GateState>> };

const EVENTS: Ev[] = [
  { f: 30, log: <><T c="dim">dvd-ingester-update.timer</T> fired → <T c="acc">update.service</T> begins</>, gates: { fetch: "live" } },
  { f: 72, log: <>fetch: polling release manifest… found <T c="acc">1.0.4</T></>, gates: { fetch: "pass", sha256: "live" } },
  { f: 106, log: <>sha256: <T c="ok">9f3c1a…e77b verified</T></>, gates: { sha256: "pass", stage: "live" } },
  { f: 140, log: <>stage: unpacking to <T c="dim">/opt/dvd-ingester/releases/1.0.4</T></>, set: { r104: "staging" }, gates: { stage: "pass", flip: "live" } },
  { f: 178, log: <>flip: <T c="acc">current -&gt; releases/1.0.4</T></>, set: { current: "1.0.4", lastFlip: 178, r104: "ok" }, gates: { flip: "pass", restart: "live" } },
  { f: 210, log: <>systemctl restart <T c="dim">dvd-rip@.service dvd-publish-one.timer …</T> ok</>, gates: { restart: "pass", doctor: "live" } },
  { f: 240, log: <>doctor: unit shape <T c="ok">ok</T> · capability probe <T c="ok">ok</T> · publish path <T c="ok">ok</T> — <T c="ok">12/12 pass</T></>, gates: { doctor: "pass" } },
  { f: 266, log: <><T c="ok">update: 1.0.4 certified. previous release retained.</T></> },

  { f: 352, log: <><T c="dim">────────────── six days later ──────────────</T></>, gates: { fetch: "", sha256: "", stage: "", flip: "", restart: "", doctor: "" } },
  { f: 368, log: <>timer fired → polling… found <T c="acc">1.0.5</T></>, gates: { fetch: "live" } },
  { f: 392, log: <>sha256: <T c="ok">verified</T> <T c="dim">(the artifact is honest; the code is not)</T></>, gates: { fetch: "pass", sha256: "pass", stage: "live" } },
  { f: 420, log: <>stage: unpacking to <T c="dim">/opt/dvd-ingester/releases/1.0.5</T></>, set: { r105: "staging" }, gates: { stage: "pass", flip: "live" } },
  { f: 450, log: <>flip: <T c="acc">current -&gt; releases/1.0.5</T></>, set: { current: "1.0.5", lastFlip: 450, r105: "ok" }, gates: { flip: "pass", restart: "live" } },
  { f: 476, log: <>systemctl restart <T c="dim">…</T> ok</>, gates: { restart: "pass", doctor: "live" } },
  { f: 505, log: <>doctor: publish path writable <T c="bad">FAIL</T> — <T c="bad">11/12</T></>, gates: { doctor: "fail" } },
  { f: 532, log: <><T c="bad">update: doctor gate refused 1.0.5</T> — initiating rollback</> },
  { f: 558, log: <>rollback: <T c="acc">current -&gt; releases/1.0.4</T> · units restarted on 1.0.4</>, set: { current: "1.0.4", lastFlip: 558, r105: "condemned" } },
  { f: 584, log: <><T c="dim">marker: /var/lib/dvd-ingester/failed/1.0.5.json written</T></> },
  { f: 608, log: <><T c="ok">● dvd-rip@.service — active (running), 1.0.4.</T> <T c="dim">nobody was paged.</T></> },
];

const stateAt = (frame: number): State => {
  const s: State = {
    gates: Object.fromEntries(GATES.map((g) => [g, ""])) as Record<string, GateState>,
    log: [],
    current: "1.0.3",
    lastFlip: -999,
    r104: "hidden",
    r105: "hidden",
  };
  for (const ev of EVENTS) {
    if (ev.f > frame) break;
    if (ev.log) s.log.push({ f: ev.f, node: ev.log });
    if (ev.gates) Object.assign(s.gates, ev.gates);
    if (ev.set) Object.assign(s, ev.set);
  }
  return s;
};

const Release: React.FC<{
  v: string;
  meta: string;
  state: "ok" | "staging" | "condemned";
  current: boolean;
  appearAt?: number;
}> = ({ v, meta, state, current, appearAt = -1 }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const s = appearAt < 0 ? 1 : pop(frame, fps, appearAt);
  const border = current ? C.good : state === "staging" ? C.accent : state === "condemned" ? C.bad : C.line;
  return (
    <div
      style={{
        border: `1.5px solid ${border}`,
        background: C.bg,
        padding: "18px 22px",
        minWidth: 180,
        position: "relative",
        fontFamily: F.mono,
        opacity: s,
        transform: `translateY(${(1 - s) * 14}px)`,
      }}
    >
      {current ? (
        <span style={{ position: "absolute", top: -12, right: 12, background: C.good, color: C.bg, fontSize: 12, letterSpacing: "0.18em", padding: "2px 9px", fontWeight: 600 }}>
          CURRENT
        </span>
      ) : null}
      {state === "condemned" ? (
        <span style={{ position: "absolute", top: -12, right: 12, background: C.bad, color: C.bg, fontSize: 11, letterSpacing: "0.12em", padding: "2px 8px", fontWeight: 600 }}>
          RETAINED FOR AUTOPSY
        </span>
      ) : null}
      <div style={{ fontFamily: F.stencil, fontSize: 38 }}>{v}</div>
      <div style={{ color: C.faint, fontSize: 13, marginTop: 5, letterSpacing: "0.08em" }}>{meta}</div>
    </div>
  );
};

export const RailScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const st = stateAt(frame);

  const flipPulse = frame - st.lastFlip < 22 ? 1 - (frame - st.lastFlip) / 22 : 0;
  const logLines = st.log.slice(-11);

  return (
    <div style={{ position: "absolute", inset: 0, padding: "130px 110px" }}>
      <div style={fadeUp(frame, 0)}>
        <Kicker idx="§02" text="Live model of bin/update.sh" />
      </div>
      <div style={{ display: "flex", alignItems: "baseline", gap: 40 }}>
        <Stencil size={92} style={{ marginTop: 18, ...fadeUp(frame, 6) }}>
          The update rail
        </Stencil>
        <span style={{ ...fadeUp(frame, 14), fontFamily: F.mono, fontSize: 16, letterSpacing: "0.14em", color: C.faint, textTransform: "uppercase" }}>
          fetch → sha256 → stage → flip → doctor → keep|rollback
        </span>
      </div>

      <div style={{ display: "flex", gap: 0, marginTop: 40, border: `1px solid ${C.line}`, background: C.bg2, ...fadeUp(frame, 18) }}>
        {/* stage */}
        <div style={{ flex: "1 1 58%", padding: "34px 36px 30px", borderRight: `1px solid ${C.lineSoft}`, position: "relative" }}>
          <div style={{ display: "flex", gap: 18, flexWrap: "wrap" }}>
            <Release v="1.0.2" meta="sha256 ok · retained" state="ok" current={false} />
            <Release v="1.0.3" meta="sha256 ok · certified" state="ok" current={st.current === "1.0.3"} />
            {st.r104 !== "hidden" ? (
              <Release v="1.0.4" meta={frame < 240 ? "sha256 ok · staged" : "sha256 ok · certified"} state={st.r104 === "staging" ? "staging" : "ok"} current={st.current === "1.0.4"} appearAt={140} />
            ) : null}
            {st.r105 !== "hidden" ? (
              <Release v="1.0.5" meta={st.r105 === "condemned" ? "failed doctor · kept" : "sha256 ok · staged"} state={st.r105 === "condemned" ? "condemned" : "staging"} current={st.current === "1.0.5"} appearAt={420} />
            ) : null}
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: 18, margin: "40px 0 26px", fontFamily: F.mono, fontSize: 19 }}>
            <span style={{ color: C.steel }}>/opt/dvd-ingester/current</span>
            <span style={{ color: C.accent, fontSize: 24, transform: `translateX(${flipPulse * 8}px)` }}>→</span>
            <span
              style={{
                color: flipPulse > 0 ? C.accent : C.good,
                border: `1.5px dashed ${flipPulse > 0 ? C.accent : C.line}`,
                padding: "6px 14px",
              }}
            >
              releases/{st.current}
            </span>
          </div>

          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            {GATES.map((g) => (
              <Gate key={g} label={g} state={st.gates[g]} />
            ))}
          </div>

          <div style={{ position: "absolute", right: 34, bottom: 26 }}>
            {frame >= 268 && frame < 348 ? <Stamp text="Passes muster" at={268} /> : null}
            {frame >= 612 ? <Stamp text="Rolled back" color={C.bad} at={612} /> : null}
          </div>
        </div>

        {/* journal */}
        <div style={{ flex: "1 1 42%" }}>
          <TermPanel title="journalctl -fu dvd-ingester-update" minHeight={470} style={{ border: "none" }}>
            {logLines.length === 0 ? (
              <div style={{ color: C.faint }}># idle. timer armed.</div>
            ) : (
              logLines.map((l) => (
                <div key={l.f} style={{ ...fadeUp(frame, l.f, 8, 10), fontSize: 16.5 }}>
                  {l.node}
                </div>
              ))
            )}
          </TermPanel>
        </div>
      </div>
    </div>
  );
};
