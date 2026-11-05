"use client";

import React, { useEffect, useRef } from "react";
import { useTheme } from "next-themes";

interface AsciiHeroFieldProps {
  word?: string;
  className?: string;
  accentColor?: [number, number, number]; // [r, g, b] in 0..1
}

// Monospace Character Sets matching shadcn/ui Hero Field
const CHARS_TIER = [" ", "·", ":", "+"];
const CHARS_ALPHABET = Array.from("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#-");
const CHARS_SCRAMBLE = Array.from("#%&*=/<>?$@");
const ALL_CHARS = [...CHARS_TIER, ...CHARS_ALPHABET, ...CHARS_SCRAMBLE];

// Vertex shader (Full-screen triangle using gl_VertexID)
const VERTEX_SHADER = `#version 300 es
void main() {
  vec2 p = vec2((gl_VertexID << 1) & 2, gl_VertexID & 2);
  gl_Position = vec4(p * 2. - 1., 0., 1.);
}`;

// Fragment shader: Cells grid & ASCII glyph selection
const CELLS_FRAGMENT_SHADER = `#version 300 es
precision highp float;
uniform vec2 uResolution, uCell;
uniform float uTime, uClock, uScale, uTrailOn, uReveal, uIntro, uWordLength;
uniform vec4 uClear;
uniform ivec4 uWord[16];
uniform sampler2D uTrail;
out vec4 outCell;

const float SCRAMBLE = ${CHARS_SCRAMBLE.length}.;
const float SCRAMBLE_AT = ${CHARS_TIER.length + CHARS_ALPHABET.length}.;
const float FEATHER = .14, FADE = 0.16, STRENGTH = 0.9;
const float BAYER[16] = float[16](0.,8.,2.,10.,12.,4.,14.,6.,3.,11.,1.,9.,15.,7.,13.,5.);

float hash(vec2 p) {
  p = fract(p * vec2(123.34, 456.21));
  p += dot(p, p + 45.32);
  return fract(p.x * p.y);
}

float noise(vec2 p) {
  vec2 i = floor(p), f = fract(p), u = f * f * (3. - 2. * f);
  return mix(mix(hash(i), hash(i + vec2(1., 0.)), u.x),
             mix(hash(i + vec2(0., 1.)), hash(i + vec2(1., 1.)), u.x), u.y);
}

float fbm(vec2 p) {
  float v = 0., a = .5;
  for(int i = 0; i < 4; i++) {
    v += a * noise(p);
    p = p * 2.02 + vec2(1.7, 9.2);
    a *= .5;
  }
  return v;
}

float trail(vec2 uv) {
  ivec2 size = textureSize(uTrail, 0);
  ivec2 last = size - 1;
  vec2 p = uv * vec2(size) - .5;
  vec2 f = fract(p);
  ivec2 i = ivec2(floor(p));
  float a = texelFetch(uTrail, clamp(i, ivec2(0), last), 0).a;
  float b = texelFetch(uTrail, clamp(i + ivec2(1, 0), ivec2(0), last), 0).a;
  float c = texelFetch(uTrail, clamp(i + ivec2(0, 1), ivec2(0), last), 0).a;
  float d = texelFetch(uTrail, clamp(i + ivec2(1, 1), ivec2(0), last), 0).a;
  return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

void main() {
  vec2 cell = floor(gl_FragCoord.xy);
  vec2 uv = (cell + .5) * uCell / uResolution;
  float aspect = uResolution.x / uResolution.y;
  vec2 p = (uv - .5) * vec2(aspect, 1.) * uScale;

  float t = uTime * .05;
  vec2 q = vec2(fbm(p * 1.1 + vec2(0., t)), fbm(p * 1.1 + vec2(5.2, 1.3 - t)));
  float tone = smoothstep(.32, .68, fbm(p * 1.4 + q * 1.5 + vec2(t * .7, -t * .4))) * uReveal;

  float k = 0.;
  if (uTrailOn > .5) {
    k = clamp(trail(uv) * STRENGTH / .38, 0., 1.);
    k = k * k * (3. - 2. * k);
    tone += (1. - tone) * k * .92;
  }

  float keep = smoothstep(0., FADE, uv.y) * smoothstep(0., FADE, 1. - uv.y);
  if (uClear.z > 0.) {
    vec2 half_ = uClear.zw * .5 * vec2(aspect, 1.);
    vec2 c = (uv - uClear.xy - uClear.zw * .5) * vec2(aspect, 1.);
    keep *= smoothstep(0., FEATHER, (length(c / max(half_, vec2(1e-4))) - 1.) * min(half_.x, half_.y));
  }
  tone *= keep;
  k *= keep;

  ivec2 b = ivec2(mod(cell, 4.));
  float level = tone * 7.;
  float tier = level >= 3.5 ? 4. : clamp(floor(level + (BAYER[b.x + b.y * 4] + .5) / 16.), 0., 3.);
  float row = cell.y, column = cell.x;
  int letter = int(mod(column + floor(hash(vec2(row, 7.13)) * uWordLength), uWordLength));
  float index = tier < 4. ? tier : float(uWord[letter / 4][letter % 4]);

  if (tier >= 4. && hash(cell + vec2(3.1, 8.7)) < max(k * .9 - .2, uIntro)) {
    index = SCRAMBLE_AT + floor(hash(cell + floor(uClock * 10.) * vec2(.71, .37)) * SCRAMBLE);
  }
  float strength = tier < 4. ? .32 + tier * .07 : .58 + .32 * tone;
  outCell = vec4(mix(strength, 1., k), 0., 0., index / 255.);
}`;

// Fragment shader: Render glyph atlas to screen
const GLYPHS_FRAGMENT_SHADER = `#version 300 es
precision highp float;
uniform sampler2D uAtlas, uCells;
uniform vec2 uResolution, uCell;
uniform vec3 uInk;
uniform float uOpacity;
out vec4 outColor;

void main() {
  vec2 frag = vec2(gl_FragCoord.x, uResolution.y - gl_FragCoord.y);
  vec2 cell = floor(frag / uCell);
  vec4 c = texelFetch(uCells, ivec2(cell), 0);
  float index = floor(c.a * 255. + .5);
  if (index < .5) {
    outColor = vec4(0.);
    return;
  }
  ivec2 texel = ivec2(int(index) * int(uCell.x), 0) + ivec2(frag - cell * uCell);
  float a = texelFetch(uAtlas, texel, 0).a * c.r * uOpacity;
  outColor = vec4(uInk * a, a);
}`;

// Fluid simulation base code
const FLUID_BASE = `#version 300 es
precision highp float;
uniform sampler2D uState, uCurl, uPressure;
uniform vec2 uGrid;
uniform float uDt, uFirst;
out vec4 outColor;
ivec2 here() { return ivec2(gl_FragCoord.xy); }
bool edge(ivec2 c) { return c.x < 1 || c.y < 1 || c.x > int(uGrid.x) - 2 || c.y > int(uGrid.y) - 2; }
vec4 at(ivec2 c) { return texelFetch(uState, c, 0); }
float cu(ivec2 c) { return texelFetch(uCurl, c, 0).x; }
float pr(ivec2 c) { return uFirst > .5 ? 0. : texelFetch(uPressure, c, 0).x; }
const ivec2 X = ivec2(1, 0), Y = ivec2(0, 1);
`;

const SPLAT_SHADER = `${FLUID_BASE}
uniform vec4 uFrom[8], uShape[8];
uniform int uCount;
void main() {
  ivec2 c = here();
  vec4 s = at(c);
  if (edge(c)) { outColor = vec4(0.); return; }
  vec2 x = vec2(c);
  for (int k = 0; k < 8; k++) {
    if (k >= uCount) break;
    vec2 a = uFrom[k].xy, ab = uFrom[k].zw;
    float l = length(ab);
    float f = uShape[k].x, m = uShape[k].y, push = uShape[k].z, d = uShape[k].w;
    vec2 g = ab / l;
    float along = dot(x - a, g);
    float sum = 0.;
    vec2 dv = vec2(0.);
    float reach = (3. * f + 1.) * 1.415;
    float lo = max(0., floor((along - reach) / l * d - .5));
    float hi = min(d - 1., ceil((along + reach) / l * d - .5));
    for (float n = lo; n <= hi; n++) {
      vec2 o = a + ab * ((n + .5) / d);
      vec2 e = x - o;
      if (any(lessThan(x, floor(o - 3. * f))) || any(greaterThan(x, ceil(o + 3. * f)))) continue;
      float u = dot(e, g) / f, r = (e.y * g.x - e.x * g.y) / f;
      float w = exp(-u * u * .65 - r * r * 1.7);
      float side = 1. - r * r * .55, twist = u * r * .7;
      sum += w;
      dv += vec2(g.x * side - g.y * twist, g.y * side + g.x * twist) * w * push;
    }
    s.z = 1. - (1. - s.z) * exp(-sum * m);
    s.xy = clamp(s.xy + dv, -80., 80.);
  }
  outColor = s;
}`;

const ADVECT_SHADER = `${FLUID_BASE}
void main() {
  ivec2 c = here();
  if (edge(c)) { outColor = vec4(0.); return; }
  vec2 p = clamp(vec2(c) - at(c).xy * uDt, vec2(.5), uGrid - 1.5);
  ivec2 i = ivec2(p);
  vec2 f = p - vec2(i);
  vec4 s = mix(mix(at(i), at(i + X), f.x), mix(at(i + Y), at(i + X + Y), f.x), f.y);
  outColor = vec4(s.xy * exp(-1.9 * uDt), s.z * exp(-3.2 * uDt), 0.);
}`;

const CURL_SHADER = `${FLUID_BASE}
void main() {
  ivec2 c = here();
  outColor = vec4(edge(c) ? 0. : .5 * (at(c + X).y - at(c - X).y - at(c + Y).x + at(c - Y).x), 0., 0., 0.);
}`;

const CONFINE_SHADER = `${FLUID_BASE}
void main() {
  ivec2 c = here();
  if (edge(c)) { outColor = vec4(0.); return; }
  vec4 s = at(c);
  vec2 o = vec2(abs(cu(c + X)) - abs(cu(c - X)), abs(cu(c + Y)) - abs(cu(c - Y)));
  float l = length(o) + 1e-4;
  float k = cu(c) * uDt * 10.;
  float speed = length(s.xy);
  vec2 v = s.xy + vec2(o.y, -o.x) / l * k;
  outColor = vec4(v * min(1., speed / (length(v) + 1e-6)), s.z, 0.);
}`;

const PRESSURE_SHADER = `${FLUID_BASE}
void main() {
  ivec2 c = here();
  if (edge(c)) { outColor = vec4(0.); return; }
  float div = -.5 * (at(c + X).x - at(c - X).x + at(c + Y).y - at(c - Y).y);
  outColor = vec4(.25 * (div + pr(c - X) + pr(c + X) + pr(c - Y) + pr(c + Y)), 0., 0., 0.);
}`;

const PROJECT_SHADER = `${FLUID_BASE}
void main() {
  ivec2 c = here();
  if (edge(c)) { outColor = vec4(0.); return; }
  vec4 s = at(c);
  vec2 v = clamp(s.xy - .5 * vec2(pr(c + X) - pr(c - X), pr(c + Y) - pr(c - Y)), -80., 80.);
  outColor = vec4(v, s.z, 1. - exp(-.06 * length(v) - .16 * s.z));
}`;

interface Target {
  buffer: WebGLFramebuffer;
  texture: WebGLTexture;
}

interface Segment {
  ax: number;
  ay: number;
  bx: number;
  by: number;
  count: number;
  ink: number;
  push: number;
  width: number;
}

export const AsciiHeroField = React.memo(function AsciiHeroField({
  word = "PLEWTUS METASEARCH ",
  className,
  accentColor = [135 / 255, 0 / 255, 255 / 255], // #8700ff
}: AsciiHeroFieldProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) return;

    let gl: WebGL2RenderingContext | null = null;
    try {
      gl = canvas.getContext("webgl2", {
        alpha: true,
        antialias: false,
        depth: false,
        powerPreference: "low-power",
        premultipliedAlpha: true,
        stencil: false,
      });
    } catch {
      return;
    }

    if (!gl) return;

    // Dummy VAO for standard WebGL2 drawArrays with gl_VertexID
    const vao = gl.createVertexArray();
    gl.bindVertexArray(vao);

    // Float texture support
    let floatPrecision: 16 | 32 | null = null;
    if (gl.getExtension("EXT_color_buffer_float")) {
      floatPrecision = 32;
    } else if (gl.getExtension("EXT_color_buffer_half_float")) {
      floatPrecision = 16;
    }

    // Compile shader program helper
    const vertShader = gl.createShader(gl.VERTEX_SHADER)!;
    gl.shaderSource(vertShader, VERTEX_SHADER);
    gl.compileShader(vertShader);

    function createProgram(
      fragSrc: string,
      samplers: Record<string, number>
    ) {
      if (!gl) throw new Error("No gl");
      const fragShader = gl.createShader(gl.FRAGMENT_SHADER)!;
      const prog = gl.createProgram()!;
      gl.shaderSource(fragShader, fragSrc);
      gl.compileShader(fragShader);
      gl.attachShader(prog, vertShader);
      gl.attachShader(prog, fragShader);
      gl.linkProgram(prog);
      gl.deleteShader(fragShader);

      const uniformMap = new Map<string, WebGLUniformLocation | null>();
      const getUniform = (name: string) => {
        if (!gl) return null;
        if (!uniformMap.has(name)) {
          uniformMap.set(name, gl.getUniformLocation(prog, name));
        }
        return uniformMap.get(name) ?? null;
      };

      gl.useProgram(prog);
      for (const [name, unit] of Object.entries(samplers)) {
        gl.uniform1i(getUniform(name), unit);
      }

      return { program: prog, uniform: getUniform };
    }

    const fluidSamplers = { uCurl: 1, uPressure: 2, uState: 0 };
    const programs = {
      cells: createProgram(CELLS_FRAGMENT_SHADER, { uTrail: 0 }),
      glyphs: createProgram(GLYPHS_FRAGMENT_SHADER, { uAtlas: 3, uCells: 4 }),
      splat: createProgram(SPLAT_SHADER, fluidSamplers),
      advect: createProgram(ADVECT_SHADER, fluidSamplers),
      curl: createProgram(CURL_SHADER, fluidSamplers),
      confine: createProgram(CONFINE_SHADER, fluidSamplers),
      pressure: createProgram(PRESSURE_SHADER, fluidSamplers),
      project: createProgram(PROJECT_SHADER, fluidSamplers),
    };

    function createTarget(
      w: number,
      h: number,
      type: "r" | "rgba" | "rgba8",
      filter: number
    ): Target {
      if (!gl) throw new Error("No gl");
      const is32 = floatPrecision === 32;
      const internalFormat =
        type === "rgba8"
          ? gl.RGBA8
          : type === "r"
          ? is32
            ? gl.R32F
            : gl.R16F
          : is32
          ? gl.RGBA32F
          : gl.RGBA16F;

      const tex = gl.createTexture()!;
      const buf = gl.createFramebuffer()!;
      gl.bindTexture(gl.TEXTURE_2D, tex);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
      gl.texStorage2D(gl.TEXTURE_2D, 1, internalFormat, w, h);
      gl.bindFramebuffer(gl.FRAMEBUFFER, buf);
      gl.framebufferTexture2D(
        gl.FRAMEBUFFER,
        gl.COLOR_ATTACHMENT0,
        gl.TEXTURE_2D,
        tex,
        0
      );
      return { buffer: buf, texture: tex };
    }

    function releaseTarget(t: Target | null) {
      if (!t || !gl) return;
      gl.deleteTexture(t.texture);
      gl.deleteFramebuffer(t.buffer);
    }

    // Canvas state variables
    let width = 0;
    let height = 0;
    let dpr = 1;
    let cellW = 0;
    let cellH = 0;
    let atlasTex: WebGLTexture | null = null;
    let atlasKey = "";
    let cellsTarget: Target | null = null;

    let fluidGrid = { width: 0, height: 0 };
    let stateTargets: Target[] = [];
    let curlTarget: Target | null = null;
    let pressureTargets: Target[] = [];
    let trailOn = false;
    let trailUntil = 0;
    let prevPointer: { x: number; y: number; time: number } | null = null;
    const segments: Segment[] = [];

    let time = 0;
    let clock = 0;
    let reveal = 0;
    let swap = 1;
    let simDt = 0;
    let animId = 0;
    let lastTick = performance.now();
    let clearBounds = [0, 0, 0, 0]; // Normalized clearance rect for content

    // Encode word into uniform array
    const wordUniform = new Int32Array(64);
    const cleanWord = word.toUpperCase().slice(0, 64);
    for (let i = 0; i < cleanWord.length; i++) {
      const idx = CHARS_ALPHABET.indexOf(cleanWord[i]);
      wordUniform[i] = idx < 0 ? 0 : CHARS_TIER.length + idx;
    }
    const wordLength = Math.max(1, cleanWord.length);

    function buildAtlas(w: number, h: number) {
      if (!gl) return;
      const offscreen = document.createElement("canvas");
      offscreen.width = w * ALL_CHARS.length;
      offscreen.height = h;
      const ctx = offscreen.getContext("2d");
      if (!ctx) return;

      ctx.fillStyle = "#ffffff";
      ctx.textAlign = "center";
      ctx.textBaseline = "alphabetic";
      const fontSize = Math.max(1, Math.round(0.74 * h));
      ctx.font = `500 ${fontSize}px ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, monospace`;

      const metrics = ctx.measureText("M");
      const ascent = metrics.actualBoundingBoxAscent || 0.74 * h * 0.7;
      const descent = metrics.actualBoundingBoxDescent || 0;
      const baseline = (h + ascent - descent) / 2;

      ALL_CHARS.forEach((char, idx) => {
        const cx = idx * w + w / 2;
        const cw = ctx.measureText(char).width;
        ctx.save();
        ctx.beginPath();
        ctx.rect(cx - w / 2, 0, w, h);
        ctx.clip();
        ctx.translate(cx, baseline);
        if (cw > 0.96 * w) {
          ctx.scale((0.96 * w) / cw, 1);
        }
        ctx.fillText(char, 0, 0);
        ctx.restore();
      });

      if (atlasTex) gl.deleteTexture(atlasTex);
      atlasTex = gl.createTexture();
      gl.activeTexture(gl.TEXTURE3);
      gl.bindTexture(gl.TEXTURE_2D, atlasTex);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
      gl.texImage2D(
        gl.TEXTURE_2D,
        0,
        gl.RGBA,
        gl.RGBA,
        gl.UNSIGNED_BYTE,
        offscreen
      );
    }

    function handleResize() {
      if (!container || !canvas || !gl) return;
      const rect = container.getBoundingClientRect();
      const pw = rect.width;
      const ph = rect.height;
      if (pw <= 0 || ph <= 0) return;

      dpr = Math.min(window.devicePixelRatio || 1, 2);
      width = Math.max(1, Math.round(pw * dpr));
      height = Math.max(1, Math.round(ph * dpr));

      canvas.width = width;
      canvas.height = height;
      canvas.style.width = `${pw}px`;
      canvas.style.height = `${ph}px`;

      // Character cell dimensions (crisp monospace grid)
      cellH = Math.max(2, Math.round(11 * dpr));
      cellW = Math.max(2, Math.round(0.62 * cellH));

      const key = `${cellW}x${cellH}`;
      if (key !== atlasKey) {
        buildAtlas(cellW, cellH);
        atlasKey = key;
      }

      // Reallocate cells texture
      releaseTarget(cellsTarget);
      cellsTarget = createTarget(
        Math.ceil(width / cellW - 1e-4),
        Math.ceil(height / cellH - 1e-4),
        "rgba8",
        gl.NEAREST
      );

      // Reallocate fluid grid
      if (floatPrecision) {
        const aspect = width / height;
        fluidGrid = {
          height: Math.round(aspect >= 1 ? 128 / aspect : 128),
          width: Math.round(aspect >= 1 ? 128 : 128 * aspect),
        };

        stateTargets.forEach(releaseTarget);
        if (curlTarget) releaseTarget(curlTarget);
        pressureTargets.forEach(releaseTarget);

        stateTargets = [
          createTarget(fluidGrid.width, fluidGrid.height, "rgba", gl.NEAREST),
          createTarget(fluidGrid.width, fluidGrid.height, "rgba", gl.NEAREST),
        ];
        curlTarget = createTarget(
          fluidGrid.width,
          fluidGrid.height,
          "r",
          gl.NEAREST
        );
        pressureTargets = [
          createTarget(fluidGrid.width, fluidGrid.height, "r", gl.NEAREST),
          createTarget(fluidGrid.width, fluidGrid.height, "r", gl.NEAREST),
        ];

        trailOn = false;
        trailUntil = 0;
        prevPointer = null;
      }

      // Measure clear area around center content
      const centerEl = container.querySelector(".z-10");
      if (centerEl) {
        const cRect = centerEl.getBoundingClientRect();
        const pad = 24 * dpr;
        clearBounds = [
          (cRect.left * dpr - pad) / width,
          (cRect.top * dpr - pad) / height,
          (cRect.width * dpr + 2 * pad) / width,
          (cRect.height * dpr + 2 * pad) / height,
        ];
      }
    }

    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(container);
    handleResize();

    // Pointer tracking for fluid wake
    function onPointerMove(e: PointerEvent) {
      if (!canvas || !floatPrecision || fluidGrid.width <= 0) return;
      const rect = canvas.getBoundingClientRect();
      const nx = (e.clientX - rect.left) / rect.width;
      const ny = (e.clientY - rect.top) / rect.height;

      if (nx < -0.05 || nx > 1.05 || ny < -0.05 || ny > 1.05) {
        prevPointer = null;
        return;
      }

      const clX = Math.max(0, Math.min(1, nx));
      const clY = Math.max(0, Math.min(1, ny));
      const now = performance.now();
      const prev = prevPointer;
      prevPointer = { x: clX, y: clY, time: now };

      if (!prev) return;

      const gw = fluidGrid.width;
      const gh = fluidGrid.height;
      const dx = (clX - prev.x) * (gw - 1);
      const dy = (clY - prev.y) * (gh - 1);
      const dist = Math.hypot(dx, dy);

      if (dist < 0.001) return;

      const minDim = Math.min(gw, gh);
      const dt = now > prev.time ? Math.max(1 / 240, (now - prev.time) / 1000) : 1 / 60;
      const speedNorm = Math.min(12, dist / minDim / dt);
      const decay = 1 - Math.exp(-speedNorm / 2.2);
      const strokeW = 0.045 * minDim * (0.65 + 0.95 * decay);
      const sampleCount = Math.min(128, Math.max(1, Math.ceil(dist / (0.4 * strokeW))));

      segments.push({
        ax: prev.x * (gw - 1),
        ay: prev.y * (gh - 1),
        bx: dx,
        by: dy,
        count: sampleCount,
        ink: (dist / strokeW) * (0.32 + 0.22 * decay) / sampleCount,
        push: (dist * (2 + 18 * decay)) / sampleCount,
        width: strokeW,
      });

      trailUntil = now + 6000;
    }

    function onPointerLeave() {
      prevPointer = null;
    }

    window.addEventListener("pointermove", onPointerMove, { passive: true });
    document.addEventListener("pointerleave", onPointerLeave);

    // Fluid simulation helpers
    function executePass(
      target: Target,
      prog: { program: WebGLProgram; uniform: (name: string) => WebGLUniformLocation | null },
      preDraw?: () => void
    ) {
      if (!gl) return;
      gl.bindFramebuffer(gl.FRAMEBUFFER, target.buffer);
      gl.useProgram(prog.program);
      gl.uniform2f(prog.uniform("uGrid"), fluidGrid.width, fluidGrid.height);
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, stateTargets[0]?.texture ?? null);
      preDraw?.();
      gl.drawArrays(gl.TRIANGLES, 0, 3);
    }

    function flipState() {
      stateTargets.reverse();
    }

    function runSimulation() {
      if (!gl || stateTargets.length < 2 || !curlTarget) return;

      gl.viewport(0, 0, fluidGrid.width, fluidGrid.height);

      // Splat pointer inputs
      while (segments.length) {
        const batch = segments.splice(0, 8);
        const fromArr = new Float32Array(32);
        const shapeArr = new Float32Array(32);

        batch.forEach((seg, i) => {
          fromArr.set([seg.ax, seg.ay, seg.bx, seg.by], 4 * i);
          shapeArr.set([seg.width, seg.ink, seg.push, seg.count], 4 * i);
        });

        executePass(stateTargets[1], programs.splat, () => {
          gl?.uniform4fv(programs.splat.uniform("uFrom"), fromArr);
          gl?.uniform4fv(programs.splat.uniform("uShape"), shapeArr);
          gl?.uniform1i(programs.splat.uniform("uCount"), batch.length);
        });
        flipState();
      }

      // Advect, Curl, Confine, Pressure & Project iterations
      let remaining = Math.min(0.05, simDt);
      simDt = 0;

      while (remaining > 1e-7) {
        const step = Math.min(1 / 60, remaining);
        remaining -= step;

        const runStep = (
          t: Target,
          p: { program: WebGLProgram; uniform: (name: string) => WebGLUniformLocation | null },
          setup?: () => void
        ) => {
          executePass(t, p, () => {
            gl?.uniform1f(p.uniform("uDt"), step);
            setup?.();
          });
        };

        // Advect
        runStep(stateTargets[1], programs.advect);
        flipState();

        // Curl
        runStep(curlTarget, programs.curl, () => {
          gl?.activeTexture(gl.TEXTURE1);
          gl?.bindTexture(gl.TEXTURE_2D, null);
        });

        // Confine
        runStep(stateTargets[1], programs.confine, () => {
          gl?.activeTexture(gl.TEXTURE1);
          gl?.bindTexture(gl.TEXTURE_2D, curlTarget?.texture ?? null);
        });
        flipState();

        // Pressure Jacobi iterations
        for (let i = 0; i < 4; i++) {
          const [readP, writeP] = pressureTargets;
          runStep(writeP, programs.pressure, () => {
            gl?.uniform1f(programs.pressure.uniform("uFirst"), i === 0 ? 1 : 0);
            gl?.activeTexture(gl.TEXTURE2);
            gl?.bindTexture(gl.TEXTURE_2D, readP?.texture ?? null);
          });
          pressureTargets.reverse();
        }

        // Project
        runStep(stateTargets[1], programs.project, () => {
          gl?.uniform1f(programs.project.uniform("uFirst"), 0);
          gl?.activeTexture(gl.TEXTURE2);
          gl?.bindTexture(gl.TEXTURE_2D, pressureTargets[0]?.texture ?? null);
        });
        flipState();
      }

      gl.activeTexture(gl.TEXTURE1);
      gl.bindTexture(gl.TEXTURE_2D, null);
      gl.activeTexture(gl.TEXTURE2);
      gl.bindTexture(gl.TEXTURE_2D, null);
    }

    // Main 60 FPS animation loop
    function render(now: number) {
      if (!gl || !cellsTarget || !atlasTex || width <= 0 || height <= 0) {
        animId = requestAnimationFrame(render);
        return;
      }

      const dt = Math.min(0.05, Math.max(1 / 480, (now - lastTick) / 1000));
      lastTick = now;

      time += dt;
      const isTrailActive = trailUntil > now;
      if (isTrailActive) simDt += dt;
      if (isTrailActive || reveal < 1) clock += dt;

      // Decode animations
      if (reveal < 1) reveal = Math.min(1, reveal + (1000 * dt) / 1400);
      if (swap < 1) swap = Math.min(1, swap + (1000 * dt) / 900);

      // 1. Simulate Fluid Wake
      if (isTrailActive && floatPrecision) {
        runSimulation();
        trailOn = true;
      } else if (trailOn) {
        gl.clearColor(0, 0, 0, 0);
        stateTargets.forEach((t) => {
          gl?.bindFramebuffer(gl.FRAMEBUFFER, t.buffer);
          gl?.clear(gl.COLOR_BUFFER_BIT);
        });
        trailOn = false;
      }

      // 2. Render Cells Grid
      const cols = Math.ceil(width / cellW - 1e-4);
      const rows = Math.ceil(height / cellH - 1e-4);
      const isNarrow = window.innerWidth < 768;
      const introFactor = (1 - reveal) ** 3;

      gl.bindFramebuffer(gl.FRAMEBUFFER, cellsTarget.buffer);
      gl.viewport(0, 0, cols, rows);
      gl.useProgram(programs.cells.program);

      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, stateTargets[0]?.texture ?? null);

      gl.uniform2f(programs.cells.uniform("uResolution"), width, height);
      gl.uniform2f(programs.cells.uniform("uCell"), cellW, cellH);
      gl.uniform1f(programs.cells.uniform("uTime"), time);
      gl.uniform1f(programs.cells.uniform("uClock"), clock);
      gl.uniform1f(programs.cells.uniform("uScale"), isNarrow ? 1.6 : 1.0);
      gl.uniform1f(programs.cells.uniform("uTrailOn"), trailOn ? 1.0 : 0.0);
      gl.uniform1f(programs.cells.uniform("uReveal"), 1 - introFactor);
      gl.uniform1f(programs.cells.uniform("uIntro"), introFactor);
      gl.uniform4iv(programs.cells.uniform("uWord"), wordUniform);
      gl.uniform1f(programs.cells.uniform("uWordLength"), wordLength);
      gl.uniform4f(
        programs.cells.uniform("uClear"),
        clearBounds[0],
        clearBounds[1],
        clearBounds[2],
        clearBounds[3]
      );

      gl.drawArrays(gl.TRIANGLES, 0, 3);

      // 3. Render Glyphs from Atlas to Screen
      gl.bindFramebuffer(gl.FRAMEBUFFER, null);
      gl.viewport(0, 0, width, height);
      gl.useProgram(programs.glyphs.program);

      gl.activeTexture(gl.TEXTURE3);
      gl.bindTexture(gl.TEXTURE_2D, atlasTex);
      gl.activeTexture(gl.TEXTURE4);
      gl.bindTexture(gl.TEXTURE_2D, cellsTarget.texture);

      gl.uniform2f(programs.glyphs.uniform("uResolution"), width, height);
      gl.uniform2f(programs.glyphs.uniform("uCell"), cellW, cellH);
      gl.uniform3f(programs.glyphs.uniform("uInk"), ...accentColor);
      gl.uniform1f(programs.glyphs.uniform("uOpacity"), 0.94);

      gl.drawArrays(gl.TRIANGLES, 0, 3);

      animId = requestAnimationFrame(render);
    }

    animId = requestAnimationFrame(render);

    return () => {
      cancelAnimationFrame(animId);
      resizeObserver.disconnect();
      window.removeEventListener("pointermove", onPointerMove);
      document.removeEventListener("pointerleave", onPointerLeave);

      if (gl) {
        releaseTarget(cellsTarget);
        stateTargets.forEach(releaseTarget);
        if (curlTarget) releaseTarget(curlTarget);
        pressureTargets.forEach(releaseTarget);
        if (atlasTex) gl.deleteTexture(atlasTex);
        if (vao) gl.deleteVertexArray(vao);
      }
    };
  }, [word, accentColor, resolvedTheme]);

  return (
    <div
      ref={containerRef}
      className={`absolute inset-0 overflow-hidden pointer-events-auto select-none ${className ?? ""}`}
      style={{
        maskImage:
          "radial-gradient(ellipse at 50% 50%, black 50%, transparent 88%)",
        WebkitMaskImage:
          "radial-gradient(ellipse at 50% 50%, black 50%, transparent 88%)",
      }}
      aria-hidden="true"
    >
      <canvas ref={canvasRef} className="block w-full h-full" />
    </div>
  );
});
