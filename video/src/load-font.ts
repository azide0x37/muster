import { continueRender, delayRender } from "remotion";
import { STENCIL_WOFF2_B64 } from "./font-data";

// Same subset woff2 the site embeds; loaded once per render worker.
let loaded = false;

export const ensureStencil = () => {
  if (loaded || typeof document === "undefined") return;
  loaded = true;
  const handle = delayRender("loading Muster Stencil");
  const face = new FontFace(
    "Muster Stencil",
    `url(data:font/woff2;base64,${STENCIL_WOFF2_B64})`
  );
  face
    .load()
    .then((f) => {
      (document.fonts as unknown as { add: (f: FontFace) => void }).add(f);
      continueRender(handle);
    })
    .catch(() => continueRender(handle));
};
