"use client";

import { useEffect, type RefObject } from "react";

/** One star layer: px² of canvas per star, radius and alpha ranges, optional glow. */
export interface StarLayer {
  name: string;
  area: number;
  r: readonly [number, number];
  a: readonly [number, number];
  glow?: number;
}

/** Small deterministic PRNG so a sky looks the same on every load. */
function mulberry32(seed: number) {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * Draws star layers onto canvases, once per size or theme change. Star colours
 * come from a CSS custom property on `root` holding "r, g, b | r, g, b | ...",
 * so they follow the theme. Callers move the canvases with transforms; nothing
 * is redrawn while animating.
 */
export function useStarfield(
  rootRef: RefObject<HTMLElement | null>,
  canvasRefs: RefObject<(HTMLCanvasElement | null)[]>,
  layers: readonly StarLayer[],
  colorsVar: string,
  seed: number,
) {
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;

    const draw = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const palette = getComputedStyle(root)
        .getPropertyValue(colorsVar)
        .split("|")
        .map((c) => c.trim())
        .filter(Boolean);
      const colors = palette.length ? palette : ["255, 255, 255"];

      layers.forEach((layer, i) => {
        const canvas = canvasRefs.current[i];
        const ctx = canvas?.getContext("2d");
        if (!canvas || !ctx) return;
        const w = canvas.clientWidth;
        const h = canvas.clientHeight;
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        ctx.clearRect(0, 0, w, h);

        const rand = mulberry32(seed + i * 7919);
        const count = Math.round((w * h) / layer.area);
        ctx.shadowBlur = layer.glow ?? 0;
        for (let n = 0; n < count; n++) {
          const x = rand() * w;
          const y = rand() * h;
          const r = layer.r[0] + rand() * (layer.r[1] - layer.r[0]);
          const a = layer.a[0] + rand() * (layer.a[1] - layer.a[0]);
          const color = `rgba(${colors[Math.floor(rand() * colors.length)]}, ${a})`;
          if (layer.glow) ctx.shadowColor = color;
          ctx.fillStyle = color;
          ctx.beginPath();
          ctx.arc(x, y, r, 0, Math.PI * 2);
          ctx.fill();
        }
        ctx.shadowBlur = 0;
      });
    };

    let frame = 0;
    const redraw = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(draw);
    };

    const resizeObserver = new ResizeObserver(redraw);
    resizeObserver.observe(root);
    // The star colours come from the theme, so redraw when it switches.
    const themeObserver = new MutationObserver(redraw);
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    draw();

    return () => {
      cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      themeObserver.disconnect();
    };
  }, [rootRef, canvasRefs, layers, colorsVar, seed]);
}
