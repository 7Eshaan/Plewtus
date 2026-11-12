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
 *
 * Performance: every per-frame change is a transform, so the GPU only moves
 * already-painted layers. The rim highlight is painted once at the top of a
 * circle and *rotated* about the planet's centre to slide along the rim, so
 * the large gradients and blurs are never repainted while animating.
 */
export function HorizonBackground({ className = "" }: { className?: string }) {
  const rootRef = useRef<HTMLDivElement>(null);
  const planetRef = useRef<HTMLDivElement>(null);
  const starRefs = useRef<(HTMLCanvasElement | null)[]>([]);

  useStarfield(rootRef, starRefs, STAR_LAYERS, "--hz-star-colors", 1337);

  useEffect(() => {
    const root = rootRef.current;
    const planet = planetRef.current;
    if (!root || !planet) return;

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

    // Layout is measured on resize only, never inside the frame loop.
    const size = { width: 1, planet: 1 };
    const measure = () => {
      size.width = root.clientWidth || 1;
      size.planet = planet.offsetWidth || 1;
    };

    const apply = () => {
      // Viewport x -> fraction across the planet's width -> angle from the
      // top of the circle. Rotating by that angle puts the highlight there.
      const planetLeft = (size.width - size.planet) / 2;
      const rimX = (current.x * size.width - planetLeft) / size.planet;
      const angle = Math.asin(Math.min(1, Math.max(-1, (rimX - 0.5) * 2)));

      root.style.setProperty("--hz-rot", `${((angle * 180) / Math.PI).toFixed(3)}deg`);
      root.style.setProperty("--hz-glow-x", `${(current.x * size.width).toFixed(1)}px`);
      root.style.setProperty("--hz-shift-x", `${((current.x - 0.5) * -14).toFixed(2)}px`);
      root.style.setProperty("--hz-shift-y", `${((current.y - 0.5) * -10).toFixed(2)}px`);
    };

    const onResize = () => {
      measure();
      apply();
    };

    const resizeObserver = new ResizeObserver(onResize);
    resizeObserver.observe(root);
    onResize();

    if (reduceMotion) {
      return () => resizeObserver.disconnect();
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

    // Off-screen: stop the frame loop and pause the CSS twinkle/breathing.
    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      root.classList.toggle("is-offscreen", !visible);
      schedule();
    });
    observer.observe(root);

    window.addEventListener("pointermove", onPointerMove, { passive: true });
    document.documentElement.addEventListener("pointerleave", onPointerLeave);
    schedule();

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      resizeObserver.disconnect();
      window.removeEventListener("pointermove", onPointerMove);
      document.documentElement.removeEventListener("pointerleave", onPointerLeave);
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
      <div ref={planetRef} className="horizon-bg__planet-wrap">
        <div className="horizon-bg__atmosphere">
          <div className="horizon-bg__atmosphere-glow" />
        </div>
        <div className="horizon-bg__rim" />
        <div className="horizon-bg__rim-light" />
        <div className="horizon-bg__planet" />
        <div className="horizon-bg__planet-light" />
      </div>
      <div className="horizon-bg__corner-glow">
        <div className="horizon-bg__corner-glow-fill" />
      </div>
    </div>
  );
}
