/**
 * Generates src/app/m3-theme.css: the Material 3 color scheme derived from the
 * brand seed color, plus spring easing curves for M3 Expressive motion.
 *
 *   npm run generate:theme
 *
 * The output is committed, so the color library is only a dev dependency and
 * nothing extra ships to the browser. Re-run after changing SEED or VARIANT.
 */

import { registerHooks } from "node:module";
import { writeFileSync } from "node:fs";

// @material/material-color-utilities 0.4.0 has an import without a ".js"
// extension, which Node's ESM resolver rejects; retry such imports with one.
registerHooks({
  resolve(specifier, context, nextResolve) {
    try {
      return nextResolve(specifier, context);
    } catch (error) {
      const relative = specifier.startsWith("./") || specifier.startsWith("../");
      if ((error as { code?: string }).code === "ERR_MODULE_NOT_FOUND" && relative && !specifier.endsWith(".js")) {
        return nextResolve(`${specifier}.js`, context);
      }
      throw error;
    }
  },
});

const { Hct, argbFromHex, hexFromArgb, MaterialDynamicColors, SchemeContent, DynamicScheme, TonalPalette } = await import(
  "@material/material-color-utilities"
);

/** Brand purple. */
const SEED = "#8700ff";
/**
 * SchemeContent keeps the seed's full chroma: primary-container is the seed
 * itself in both themes. (SchemeTonalSpot, the M3 default, would desaturate
 * it to a dusty lavender.) Its tertiary is an analogous magenta.
 */
const VARIANT = SchemeContent;
const SPEC = "2025";
/** contrastLevel for users who ask for more contrast (0 = standard, 1 = max). */
const HIGH_CONTRAST = 0.5;

const OUTPUT = new URL("../src/app/m3-theme.css", import.meta.url);

const source = Hct.fromInt(argbFromHex(SEED));
/**
 * background/on_background duplicate surface/on_surface, and the plain *_dim
 * accent roles are only defined for the watch platform. (surface_dim and the
 * *_fixed_dim roles are regular roles and are kept.)
 */
const SKIPPED_ROLES = new Set(["background", "on_background", "primary_dim", "secondary_dim", "tertiary_dim", "error_dim"]);
const colors = new MaterialDynamicColors().allColors.filter((c) => !SKIPPED_ROLES.has(c.name));

/**
 * Dark mode is true black with neutral greys: surfaces and outlines come
 * from a chroma-0 palette, so purple appears only in the accent roles
 * (primary, secondary, tertiary and their containers). Light mode keeps the
 * standard, lightly purple-tinted M3 surfaces.
 */
const NEUTRAL = TonalPalette.fromHueAndChroma(source.hue, 0);

/**
 * Dark surface tones: the page is black (tone 0) and containers step up in
 * neutral grey. M3's own dark surfaces sit at tones 4-24, which reads as
 * dark grey rather than black.
 */
const DARK_SURFACE_TONES: Record<string, number> = {
  surface: 0,
  surface_dim: 0,
  surface_container_lowest: 0,
  surface_container_low: 6,
  surface_container: 9,
  surface_container_high: 13,
  surface_container_highest: 18,
  surface_bright: 24,
};

function scheme(dark: boolean, contrast: number) {
  const base = new VARIANT(source, dark, contrast, SPEC);
  if (!dark) return base;
  return new DynamicScheme({
    sourceColorHct: source,
    variant: base.variant,
    contrastLevel: contrast,
    isDark: true,
    specVersion: SPEC,
    primaryPalette: base.primaryPalette,
    secondaryPalette: base.secondaryPalette,
    tertiaryPalette: base.tertiaryPalette,
    errorPalette: base.errorPalette,
    neutralPalette: NEUTRAL,
    neutralVariantPalette: NEUTRAL,
  });
}

/** Hex value of a color role in one scheme. */
function roleHex(name: string, dark: boolean, contrast: number): string {
  if (contrast === 0) {
    const accent = accentOverrides(dark)[name];
    if (accent) return accent;
  }
  if (dark && name in DARK_SURFACE_TONES) return hexFromArgb(NEUTRAL.tone(DARK_SURFACE_TONES[name]));
  return hexFromArgb(colors.find((c) => c.name === name)!.getArgb(scheme(dark, contrast)));
}

// ---------------------------------------------------------------------------
// Bright accents. M3's defaults put pale, low-chroma purples on the accent
// roles (e.g. #d8b9ff text, #5b2a99 containers), which read as washed out.
// In the standard-contrast schemes the accents are kept as vivid as text
// contrast allows instead. (The high-contrast schemes keep M3's own roles.)
// ---------------------------------------------------------------------------

/** WCAG relative luminance of a #rrggbb color. */
function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const v = parseInt(hex.slice(i, i + 2), 16) / 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrastRatio(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

/** White or black, whichever reads better on `background`. */
function onColor(background: string): string {
  return contrastRatio("#ffffff", background) >= contrastRatio("#000000", background) ? "#ffffff" : "#000000";
}

/**
 * The most vivid purple usable as text and icons on the scheme's busiest
 * surface (surface-container-high): the seed itself when it passes 4.5:1,
 * otherwise the primary palette tone nearest the seed's that does (lighter
 * on dark backgrounds, darker on light ones), keeping as much chroma as
 * possible.
 */
function readableAccent(dark: boolean): string {
  const backgroundHex = dark
    ? hexFromArgb(NEUTRAL.tone(DARK_SURFACE_TONES.surface_container_high))
    : hexFromArgb(colors.find((c) => c.name === "surface_container_high")!.getArgb(scheme(false, 0)));
  if (contrastRatio(SEED, backgroundHex) >= 4.5) return SEED;
  const palette = new VARIANT(source, dark, 0, SPEC).primaryPalette;
  for (let tone = Math.round(source.tone); tone >= 0 && tone <= 100; tone += dark ? 1 : -1) {
    const hex = hexFromArgb(palette.tone(tone));
    if (contrastRatio(hex, backgroundHex) >= 4.5) return hex;
  }
  return hexFromArgb(palette.tone(dark ? 80 : 40));
}

const accentCache = new Map<boolean, Record<string, string>>();

/**
 * Accent roles for the standard schemes. Filled accents (buttons, selected
 * chips, the active nav pill, badges) are the brand seed itself, with
 * white or black text for contrast; accent text and icons use readableAccent.
 */
function accentOverrides(dark: boolean): Record<string, string> {
  const cached = accentCache.get(dark);
  if (cached) return cached;
  const accent = readableAccent(dark);
  const overrides = {
    primary_container: SEED,
    on_primary_container: onColor(SEED),
    secondary_container: SEED,
    on_secondary_container: onColor(SEED),
    primary: accent,
    on_primary: onColor(accent),
    secondary: accent,
    on_secondary: onColor(accent),
  };
  accentCache.set(dark, overrides);
  return overrides;
}

function roleDeclarations(dark: boolean, contrast: number, indent: string): string[] {
  return colors.map((c) => `${indent}--md-sys-color-${c.name.replaceAll("_", "-")}: ${roleHex(c.name, dark, contrast)};`);
}

// ---------------------------------------------------------------------------
// Springs: M3 Expressive motion tokens (damping ratio, stiffness), sampled
// into CSS linear() easing functions with a matching settle duration.
// ---------------------------------------------------------------------------

interface Spring {
  name: string;
  damping: number;
  stiffness: number;
}

const SPRINGS: Spring[] = [
  // Spatial: position and size; may overshoot slightly.
  { name: "spatial-fast", damping: 0.6, stiffness: 800 },
  { name: "spatial", damping: 0.8, stiffness: 380 },
  { name: "spatial-slow", damping: 0.8, stiffness: 200 },
  // Effects: color and opacity; critically damped, never overshoots.
  { name: "effects-fast", damping: 1, stiffness: 3800 },
  { name: "effects", damping: 1, stiffness: 1600 },
  { name: "effects-slow", damping: 1, stiffness: 800 },
];

/** Step response (0 -> 1) of a unit-mass spring at time t (seconds). */
function springPosition({ damping, stiffness }: Spring, t: number): number {
  const w0 = Math.sqrt(stiffness);
  if (damping < 1) {
    const wd = w0 * Math.sqrt(1 - damping * damping);
    const decay = Math.exp(-damping * w0 * t);
    return 1 - decay * (Math.cos(wd * t) + ((damping * w0) / wd) * Math.sin(wd * t));
  }
  return 1 - Math.exp(-w0 * t) * (1 + w0 * t);
}

/** Time after which the spring stays within 0.1% of its target. */
function settleTime(spring: Spring): number {
  const step = 0.001;
  let lastOutside = 0;
  for (let t = 0; t < 5; t += step) {
    if (Math.abs(1 - springPosition(spring, t)) > 0.001) lastOutside = t;
  }
  return lastOutside + step;
}

function springEasing(spring: Spring): { easing: string; ms: number } {
  const duration = settleTime(spring);
  const samples = 40;
  const points: string[] = [];
  for (let i = 0; i <= samples; i++) {
    const value = i === samples ? 1 : springPosition(spring, (duration * i) / samples);
    points.push(String(Number(value.toFixed(4))));
  }
  return { easing: `linear(${points.join(", ")})`, ms: Math.round(duration * 1000) };
}

// ---------------------------------------------------------------------------

const roleNames = colors.map((c) => c.name.replaceAll("_", "-"));
const codeColor = (name: string) => roleHex(name, true, 0);
const springs = SPRINGS.map((spring) => ({ ...spring, ...springEasing(spring) }));

const css = `/*
 * GENERATED by scripts/generate-m3-theme.ts — do not edit by hand.
 * Material 3 scheme: seed ${SEED}, ${VARIANT.name}, spec ${SPEC}.
 */

/* Light scheme (default) */
:root {
${roleDeclarations(false, 0, "  ").join("\n")}
  --md-sys-color-shadow: #000000;
  --md-sys-color-scrim: #000000;
}

/* Dark scheme (next-themes sets .dark on <html>) */
.dark {
${roleDeclarations(true, 0, "  ").join("\n")}
}

/* Higher-contrast schemes for users who ask for more contrast. */
@media (prefers-contrast: more) {
  :root:not(.dark) {
${roleDeclarations(false, HIGH_CONTRAST, "    ").join("\n")}
  }

  :root.dark {
${roleDeclarations(true, HIGH_CONTRAST, "    ").join("\n")}
  }
}

/* Spring settle durations, paired with the --ease-spatial-* / --ease-effects-* curves. */
:root {
${springs.map((s) => `  --md-motion-${s.name}-duration: ${s.ms}ms;`).join("\n")}
}

/* Color roles as Tailwind utilities: bg-primary-container, text-on-surface-variant, ... */
@theme inline {
${roleNames.map((n) => `  --color-${n}: var(--md-sys-color-${n});`).join("\n")}
  --color-shadow: var(--md-sys-color-shadow);
  --color-scrim: var(--md-sys-color-scrim);
}

@theme {
  /* Code blocks stay dark in both themes: taken from the dark scheme (a
     step above its black page, so panels stand out in dark mode). */
  --color-code-surface: ${codeColor("surface_container_low")};
  --color-code-surface-high: ${codeColor("surface_container_high")};
  --color-code-on-surface: ${codeColor("on_surface")};
  --color-code-on-surface-variant: ${codeColor("on_surface_variant")};
  --color-code-outline: ${codeColor("outline_variant")};
  --color-code-primary: ${codeColor("primary")};

  /* M3 Expressive springs (damping, stiffness) as easing curves: ease-spatial, ease-effects-fast, ... */
${springs.map((s) => `  --ease-${s.name}: ${s.easing}; /* ${s.damping} / ${s.stiffness} */`).join("\n")}
}

/* motion-spatial, motion-effects-fast, ...: a spring curve with its settle duration. */
${springs
  .map(
    (s) => `@utility motion-${s.name} {
  transition-timing-function: var(--ease-${s.name});
  transition-duration: var(--md-motion-${s.name}-duration);
}`,
  )
  .join("\n\n")}
`;

writeFileSync(OUTPUT, css);
console.log(`Wrote ${OUTPUT.pathname} (${roleNames.length} color roles, ${springs.length} springs).`);
