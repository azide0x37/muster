import React from "react";
import { interpolate, useCurrentFrame } from "remotion";
import { C, F } from "../theme";
import { Kicker, Stencil, TermPanel, T, fadeUp } from "../ui";

const TREE: Array<{ segs: Array<[string, string]>; at: number }> = [
  { segs: [["cfg", "/etc/dvd-ingester/"], ["pad", "          "], ["note", "# config · survives updates"]], at: 60 },
  { segs: [["fg", "  dvd-ingester.env"]], at: 74 },
  { segs: [["d", "/opt/dvd-ingester/"]], at: 88 },
  { segs: [["d", "  releases/"]], at: 100 },
  { segs: [["fg", "    1.0.2/"], ["pad", "                  "], ["note", "# immutable"]], at: 112 },
  { segs: [["fg", "    1.0.3/"], ["pad", "                  "], ["note", "# immutable"]], at: 124 },
  { segs: [["hot", "  current -> releases/1.0.3"], ["pad", " "], ["note", "# the only moving part"]], at: 140 },
  { segs: [["d", "/var/lib/dvd-ingester/"], ["pad", "      "], ["note", "# state · failure markers"]], at: 158 },
];

const UNITS: Array<[string, string, boolean]> = [
  ["90-dvd-ingester.rules", "udev: matches the drive, asks systemd", false],
  ["dvd-rip@sr0.service", "one bounded rip job", true],
  ["dvd-publish-one.timer", "atomic publish drain", false],
  ["dvd-ingester-doctor.timer", "health inspection", false],
  ["dvd-ingester-update.timer", "update polling", false],
];

const COL: Record<string, string> = {
  cfg: C.good,
  d: C.steel,
  note: C.faint,
  hot: C.accent,
  fg: C.fg,
  pad: C.faint,
};

export const ShapeScene: React.FC = () => {
  const frame = useCurrentFrame();
  const sweep = interpolate(frame, [210, 240], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  return (
    <div style={{ position: "absolute", inset: 0, padding: "150px 110px" }}>
      <div style={fadeUp(frame, 0)}>
        <Kicker idx="§02" text="The shape · systemd owns lifecycle" />
      </div>
      <Stencil size={104} style={{ marginTop: 22, ...fadeUp(frame, 8) }}>
        One boring, load-bearing shape
      </Stencil>

      <div style={{ display: "flex", gap: 70, marginTop: 56, alignItems: "flex-start" }}>
        <div style={{ flex: "0 0 900px", ...fadeUp(frame, 40) }}>
          <TermPanel title="pi@shed — install layout" right="tree -L 2" minHeight={330}>
            {TREE.map((ln, i) => {
              const visible = frame >= ln.at;
              const isSymlink = i === 6;
              return (
                <div
                  key={i}
                  style={{
                    whiteSpace: "pre",
                    opacity: visible ? 1 : 0,
                    fontSize: 19,
                    position: "relative",
                  }}
                >
                  {isSymlink ? (
                    <span
                      style={{
                        position: "absolute",
                        inset: "-2px -8px",
                        background: C.accentSoft,
                        transform: `scaleX(${sweep})`,
                        transformOrigin: "left",
                      }}
                    />
                  ) : null}
                  <span style={{ position: "relative" }}>
                    {ln.segs.map(([c, t], j) => (
                      <span key={j} style={{ color: COL[c] }}>
                        {t}
                      </span>
                    ))}
                  </span>
                </div>
              );
            })}
          </TermPanel>
          <div style={{ display: "grid", gap: 10, marginTop: 18 }}>
            {UNITS.map(([nm, what, running], i) => (
              <div
                key={nm}
                style={{
                  ...fadeUp(frame, 130 + i * 12, 12, 16),
                  display: "flex",
                  alignItems: "center",
                  gap: 16,
                  border: `1px solid ${C.line}`,
                  background: C.bg2,
                  padding: "13px 18px",
                  fontFamily: F.mono,
                  fontSize: 17,
                }}
              >
                <span
                  style={{
                    width: 10,
                    height: 10,
                    borderRadius: "50%",
                    background: running ? C.good : C.steelDim,
                    boxShadow: running ? `0 0 9px ${C.good}` : undefined,
                  }}
                />
                <span>{nm}</span>
                <span style={{ marginLeft: "auto", color: C.faint, fontSize: 14 }}>{what}</span>
              </div>
            ))}
          </div>
        </div>

        <div style={{ paddingTop: 8, maxWidth: 640 }}>
          {[
            ["systemd owns lifecycle.", "udev is allowed one verb: ask systemd to start a unit."],
            ["Releases are immutable.", "Updating a service is moving one symlink. Un-updating is moving it back."],
            ["Health is a program, not a feeling.", "doctor.sh certifies every release — or refuses to."],
          ].map(([head, body], i) => (
            <div key={head} style={{ ...fadeUp(frame, 90 + i * 60, 16), marginTop: i ? 40 : 0 }}>
              <div style={{ fontFamily: F.mono, fontSize: 26, color: i === 1 ? C.accent : C.fg }}>
                <T c={i === 1 ? "acc" : "fg"}>{head}</T>
              </div>
              <div style={{ fontSize: 21, color: C.muted, marginTop: 8, lineHeight: 1.55 }}>{body}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
