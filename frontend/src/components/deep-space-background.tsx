"use client";

import { useEffect, useRef } from "react";
import { useStarfield, type StarLayer } from "@/lib/starfield";

/** Sparser and dimmer than the hero sky so long sections stay readable. */
const STAR_LAYERS: readonly StarLayer[] = [
  { name: "far", area: 7000, r: [0.3, 0.7], a: [0.15, 0.45] },
  { name: "near", area: 52000, r: [0.8, 1.3], a: [0.4, 0.8], glow: 4 },
];

/**
 * Background for the sections below the hero: the space beyond the planet.
 * The planet's dark surface fades into a sparse, neutral starfield. The star
 * layers drift at different speeds as the page scrolls, so the space keeps
 * its depth.
 */
export function DeepSpaceBackground() {
  const rootRef = useRef<HTMLDivElement>(null);
  const starRefs = useRef<(HTMLCanvasElement | null)[]>([]);

  useStarfield(rootRef, starRefs, STAR_LAYERS, "--ds-star-colors", 4242);

  useEffect(() => {
    const root = rootRef.current;
    if (!root || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    let frame = 0;
    let visible = false;

    // Scroll progress through the section block, 0 (just entering) to 1 (leaving).
    const update = () => {
      frame = 0;
      const rect = root.getBoundingClientRect();
      const vh = window.innerHeight;
      const p = Math.min(1, Math.max(0, (vh - rect.top) / (rect.height + vh)));
      root.style.setProperty("--ds-p", p.toFixed(4));
    };

    const onScroll = () => {
      if (visible && !frame) frame = requestAnimationFrame(update);
    };

    const observer = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting;
      onScroll();
    });
    observer.observe(root);
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    update();

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
    };
  }, []);

  return (
    <div ref={rootRef} className="deep-space" aria-hidden="true">
      <div className="deep-space__surface" />
      {STAR_LAYERS.map((layer, i) => (
        <canvas
          key={layer.name}
          ref={(el) => {
            starRefs.current[i] = el;
          }}
          className={`deep-space__stars deep-space__stars--${layer.name}`}
        />
      ))}
    </div>
  );
}
