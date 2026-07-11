import React from "react";
import { useCurrentFrame } from "remotion";
import { C, F } from "../theme";
import { Kicker, Stencil, fadeUp } from "../ui";

const LAYERS = [
  ["rip.sh", "2019 · author: you, allegedly"],
  ["rip_v2_FINAL.sh", "2020 · calls rip.sh, sometimes"],
  ["fix_bluetooth.sh", "2021 · cron @reboot"],
  ["backup.sh.old", "2022 · does not back up"],
  ["crontab -e", "undated · author unknown"],
  ["/usr/local/bin/helper", "load-bearing · do not touch"],
] as const;

const BEATS = [
  "a quick script becomes a service.",
  "the service becomes infrastructure.",
  "infrastructure becomes the thing holding up the shed.",
] as const;

export const ProblemScene: React.FC = () => {
  const frame = useCurrentFrame();
  return (
    <div style={{ position: "absolute", inset: 0, padding: "150px 110px" }}>
      <div style={fadeUp(frame, 0)}>
        <Kicker idx="§01" text="The problem" />
      </div>
      <Stencil size={118} style={{ marginTop: 22, ...fadeUp(frame, 8) }}>
        Server
        <br />
        archaeology
      </Stencil>

      <div style={{ display: "flex", gap: 90, marginTop: 60, alignItems: "flex-start" }}>
        <div style={{ flex: "0 0 760px" }}>
          {LAYERS.map(([name, era], i) => {
            const deep = i === LAYERS.length - 1;
            const hot = deep && frame > 200;
            return (
              <div
                key={name}
                style={{
                  ...fadeUp(frame, 56 + i * 13, 12, 18),
                  border: `1px solid ${C.line}`,
                  borderLeft: `4px solid ${hot ? C.bad : deep ? C.bad : C.steelDim}`,
                  background: hot ? C.badSoft : C.bg2,
                  padding: "16px 22px",
                  marginTop: -1,
                  fontFamily: F.mono,
                  fontSize: 19,
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 20,
                }}
              >
                <span style={{ color: C.fg }}>{name}</span>
                <span
                  style={{
                    color: hot ? C.bad : C.faint,
                    fontSize: 14,
                    letterSpacing: "0.14em",
                    textTransform: "uppercase",
                    alignSelf: "center",
                  }}
                >
                  {era}
                </span>
              </div>
            );
          })}
          <div
            style={{
              ...fadeUp(frame, 240),
              marginTop: 18,
              fontFamily: F.mono,
              fontSize: 15,
              color: C.faint,
              letterSpacing: "0.06em",
            }}
          >
            FIG. 01 — excavation of /usr/local/bin. Every layer was “just a quick script.”
          </div>
        </div>

        <div style={{ paddingTop: 26 }}>
          {BEATS.map((b, i) => (
            <div
              key={b}
              style={{
                ...fadeUp(frame, 80 + i * 52, 16),
                fontFamily: F.mono,
                fontSize: 33,
                lineHeight: 1.5,
                color: i === 2 ? C.accent : C.fg,
                maxWidth: 720,
                marginTop: i ? 26 : 0,
              }}
            >
              {b}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
