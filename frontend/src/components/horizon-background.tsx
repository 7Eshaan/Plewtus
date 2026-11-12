"use client";

import { useEffect, useRef } from "react";
import { useStarfield, type StarLayer } from "@/lib/starfield";

/** Star layers, far to near: density (px² per star), radius and alpha ranges. */
const STAR_LAYERS: readonly StarLayer[] = [
  { name: "far", area: 2600, r: [0.3, 0.7], a: [0.25, 0.6] },
  { name: "mid", area: 9000, r: [0.6, 1.0], a: [0.5, 0.85] },
  { name: "near", area: 32000, r: [1.0, 1.6], a: [0.75, 1], glow: 6 },
];

/**
 * Hero background: a starfield over deep space, with the glowing rim of a
 * planet rising from the bottom edge.
 *
 * It reacts to the pointer: the brightest point of the rim slides toward the
 * cursor and the three star layers move at different depths for parallax.
 * When idle (or on touch screens) the highlight drifts slowly on its own.
 * Everything is plain CSS driven by a few custom properties, so the frame loop
 * only writes style variables and stops when off-screen.
 */
export function HorizonBackground({ className = "" }: { className?: string }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const starRefs = useRef<(HTMLCanvasElement | null)[]>([]);

  useStarfield(rootRef, starRefs, STAR_LAYERS, "--hz-star-colors", 1337);

  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;

    const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    // Positions are fractions of the element (0..1). The rim highlight rests
    // left of centre, like light coming from behind the planet.
    const rest = { x: 0.32, y: 0.4 };
    const target = { ...rest };
    const current = { ...rest };
    let pointerActive = false;
    let lastPointer = 0;
    let visible = true;
    let frame = 0;
    const start = performance.now();

    const apply = () => {
      const rect = root.getBoundingClientRect();
      // The planet is wider than the viewport, so map the viewport x onto the
      // planet's own width for the rim gradient.
      const planetWidth = Math.max(rect.width * 1.8, 1400);
      const planetLeft = (rect.width - planetWidth) / 2;
      const rimX = (current.x * rect.width - planetLeft) / planetWidth;
      // Where the circle's top edge sits at that x, as a fraction of its height.
      const dx = rimX - 0.5;
      const rimY = 0.5 - Math.sqrt(Math.max(0, 0.25 - dx * dx));

      root.style.setProperty("--hz-mx", `${(current.x * 100).toFixed(2)}%`);
      root.style.setProperty("--hz-rim-x", `${(rimX * 100).toFixed(2)}%`);
      root.style.setProperty("--hz-rim-y", `${(rimY * 100).toFixed(2)}%`);
      root.style.setProperty("--hz-shift-x", `${((current.x - 0.5) * -14).toFixed(2)}px`);
      root.style.setProperty("--hz-shift-y", `${((current.y - 0.5) * -10).toFixed(2)}px`);
    };

    if (reduceMotion) {
      apply();
      return;
    }

    const tick = (now: number) => {
      frame = 0;
      if (!visible) return;

      // Drift on a slow loop once the pointer has been still for a while.
      if (!pointerActive || now - lastPointer > 4000) {
        const t = (now - start) / 1000;
        target.x = rest.x + Math.sin(t * 0.18) * 0.16;
        target.y = rest.y + Math.cos(t * 0.13) * 0.06;
      }

      current.x += (target.x - current.x) * 0.06;
      current.y += (target.y - current.y) * 0.06;
      apply();
      frame = requestAnimationFrame(tick);
    };

    const schedule = () => {
      if (!frame && visible) frame = requestAnimationFrame(tick);
    };

    const onPointerMove = (event: PointerEvent) => {
      if (event.pointerType === "touch") return;
      const rect = root.getBoundingClientRect();
      if (event.clientY < rect.top || event.clientY > rect.bottom) {
        pointerActive = false;
        return;
      }
      target.x = Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width));
      target.y = Math.min(1, Math.max(0, (event.clientY - rect.top) / rect.height));
      pointerActive = true;
      lastPointer = performance.now();
      schedule();
    };

    const onPointerLeave = () => {
      pointerActive = false;
    };

    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      schedule();
    });
    observer.observe(root);

    window.addEventListener("pointermove", onPointerMove, { passive: true });
    document.documentElement.addEventListener("pointerleave", onPointerLeave);
    window.addEventListener("resize", apply);
    apply();
    schedule();

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      window.removeEventListener("pointermove", onPointerMove);
      document.documentElement.removeEventListener("pointerleave", onPointerLeave);
      window.removeEventListener("resize", apply);
    };
  }, []);

  return (
    <div ref={rootRef} className={`horizon-bg ${className}`} aria-hidden="true">
      <div className="horizon-bg__sky" />
      {STAR_LAYERS.map((layer, i) => (
        <canvas
          key={layer.name}
          ref={(el) => {
            starRefs.current[i] = el;
          }}
          className={`horizon-bg__stars horizon-bg__stars--${layer.name}`}
        />
      ))}
      <div className="horizon-bg__planet-wrap">
        <div className="horizon-bg__atmosphere" />
        <div className="horizon-bg__rim" />
        <div className="horizon-bg__planet" />
      </div>
      <div className="horizon-bg__corner-glow" />
    </div>
  );
}
