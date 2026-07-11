import React from "react";
import { Composition } from "remotion";
import { Explainer } from "./Explainer";
import { FPS, H, SCENES, SECTION_LABEL, TOTAL_FRAMES, W, type SceneKey } from "./timeline";
import { Shell } from "./ui";
import { HeroScene } from "./scenes/Hero";
import { ProblemScene } from "./scenes/Problem";
import { ShapeScene } from "./scenes/Shape";
import { RailScene } from "./scenes/Rail";
import { PatternsScene } from "./scenes/Patterns";
import { ABScene } from "./scenes/AB";
import { OutroScene } from "./scenes/Outro";

const standalone = (key: SceneKey, Scene: React.FC): React.FC => {
  const Wrapped: React.FC = () => (
    <Shell section={SECTION_LABEL[key]}>
      <Scene />
    </Shell>
  );
  return Wrapped;
};

const SCENE_LIST: Array<[SceneKey, React.FC]> = [
  ["hero", HeroScene],
  ["problem", ProblemScene],
  ["shape", ShapeScene],
  ["rail", RailScene],
  ["patterns", PatternsScene],
  ["ab", ABScene],
  ["outro", OutroScene],
];

export const RemotionRoot: React.FC = () => (
  <>
    <Composition
      id="MusterExplainer"
      component={Explainer}
      durationInFrames={TOTAL_FRAMES}
      fps={FPS}
      width={W}
      height={H}
    />
    {SCENE_LIST.map(([key, Scene]) => (
      <Composition
        key={key}
        id={`Scene-${key}`}
        component={standalone(key, Scene)}
        durationInFrames={SCENES[key].dur}
        fps={FPS}
        width={W}
        height={H}
      />
    ))}
  </>
);
