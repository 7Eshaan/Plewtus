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

/** A star layer painted as a repeating tile of this height (CSS px). */
export interface StarTileLayer extends StarLayer {
  tileHeight: number;
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

/** Reads a "r, g, b | r, g, b | ..." palette from a CSS custom property. */
function readColors(root: HTMLElement, colorsVar: string) {
  const palette = getComputedStyle(root)
    .getPropertyValue(colorsVar)
    .split("|")
    .map((c) => c.trim())
    .filter(Boolean);
  return palette.length ? palette : ["255, 255, 255"];
}

/**
 * Paints one layer of stars into a w×h (CSS px) area. `pad` keeps stars away
 * from the top and bottom edges so a vertically repeating tile has no seams.
 */
function paintStars(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  layer: StarLayer,
  colors: string[],
  seed: number,
  pad = 0,
) {
  const rand = mulberry32(seed);
  const count = Math.round((w * h) / layer.area);
  ctx.shadowBlur = layer.glow ?? 0;
  for (let n = 0; n < count; n++) {
    const x = rand() * w;
    const y = pad + rand() * (h - pad * 2);
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
}

/**
 * Runs `draw` now, and again (at most once per frame) when `root` resizes or
 * the theme switches (star colours come from the theme). Returns a cleanup.
 */
function watchRedraw(root: HTMLElement, draw: () => void) {
  let frame = 0;
  const redraw = () => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(draw);
  };

  const resizeObserver = new ResizeObserver(redraw);
  resizeObserver.observe(root);
  const themeObserver = new MutationObserver(redraw);
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  draw();

  return () => {
    cancelAnimationFrame(frame);
    resizeObserver.disconnect();
    themeObserver.disconnect();
  };
}

/**
 * Draws star layers onto viewport-sized canvases, once per size or theme
 * change. Callers move the canvases with transforms; nothing is redrawn while
 * animating.
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

    return watchRedraw(root, () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const colors = readColors(root, colorsVar);
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
        paintStars(ctx, w, h, layer, colors, seed + i * 7919);
      });
    });
  }, [rootRef, canvasRefs, layers, colorsVar, seed]);
}

/** Blob URL currently shown by each tile element, released when replaced. */
const tileUrls = new WeakMap<HTMLElement, string>();

/**
 * For tall areas: paints each layer once into a full-width tile and sets it as
 * a vertically repeating background on the matching element. Unlike a canvas
 * as tall as the page, only the part near the screen is ever rasterised.
 * Give layers different tile heights so the combined pattern never visibly
 * repeats.
 */
export function useStarTiles(
  rootRef: RefObject<HTMLElement | null>,
  tileRefs: RefObject<(HTMLElement | null)[]>,
  layers: readonly StarTileLayer[],
  colorsVar: string,
  seed: number,
) {
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const els = tileRefs.current;
    let active = true;

    const stop = watchRedraw(root, () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const colors = readColors(root, colorsVar);
      const w = root.clientWidth;
      if (!w) return;

      layers.forEach((layer, i) => {
        const el = els[i];
        if (!el) return;
        const h = layer.tileHeight;
        const canvas = document.createElement("canvas");
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        const ctx = canvas.getContext("2d");
        if (!ctx) return;
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        paintStars(ctx, w, h, layer, colors, seed + i * 7919, (layer.r[1] + (layer.glow ?? 0)) * 2);

        canvas.toBlob((blob) => {
          if (!blob || !active) return;
          const url = URL.createObjectURL(blob);
          const previous = tileUrls.get(el);
          el.style.backgroundImage = `url(${url})`;
          el.style.backgroundSize = `${w}px ${h}px`;
          tileUrls.set(el, url);
          // Released only after the new tile is shown, so nothing flashes.
          if (previous) URL.revokeObjectURL(previous);
        });
      });
    });

    return () => {
      active = false;
      stop();
      els.forEach((el) => {
        const url = el && tileUrls.get(el);
        if (url) URL.revokeObjectURL(url);
        if (el) tileUrls.delete(el);
      });
    };
  }, [rootRef, tileRefs, layers, colorsVar, seed]);
}
