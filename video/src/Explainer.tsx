import React from "react";
import { Audio, Sequence, staticFile, useCurrentFrame } from "remotion";
import { SCENES, SECTION_LABEL, VO_FILE, WITH_VO, type SceneKey } from "./timeline";
import { Shell } from "./ui";
import { HeroScene } from "./scenes/Hero";
import { ProblemScene } from "./scenes/Problem";
import { ShapeScene } from "./scenes/Shape";
import { RailScene } from "./scenes/Rail";
import { PatternsScene } from "./scenes/Patterns";
import { ABScene } from "./scenes/AB";
import { OutroScene } from "./scenes/Outro";

const SCENE_COMPONENTS: Record<SceneKey, React.FC> = {
  hero: HeroScene,
  problem: ProblemScene,
  shape: ShapeScene,
  rail: RailScene,
  patterns: PatternsScene,
  ab: ABScene,
  outro: OutroScene,
};

export const Explainer: React.FC = () => {
  const frame = useCurrentFrame();
  const active =
    (Object.keys(SCENES) as SceneKey[]).find(
      (k) => frame >= SCENES[k].from && frame < SCENES[k].from + SCENES[k].dur
    ) ?? "outro";

  return (
    <Shell section={SECTION_LABEL[active]}>
      {(Object.keys(SCENES) as SceneKey[]).map((k) => {
        const Scene = SCENE_COMPONENTS[k];
        return (
          <Sequence key={k} from={SCENES[k].from} durationInFrames={SCENES[k].dur} name={k}>
            <Scene />
          </Sequence>
        );
      })}
      {WITH_VO ? <Audio src={staticFile(VO_FILE)} /> : null}
    </Shell>
  );
};
