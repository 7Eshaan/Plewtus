import type { components } from "@/types/api";

type ApiSearchRequest = components["schemas"]["SearchRequest"];
type ExtractOptions = Partial<ApiSearchRequest["extract"]> & { enabled: true };
/** Outgoing payload: fields with server-side defaults may be omitted. */
export type SearchRequest = Partial<Omit<ApiSearchRequest, "extract">> & {
  query: string;
  extract?: ExtractOptions;
};

/** Upper bounds for requests made with the demo's server-side API key. */
export const DEMO_LIMITS = {
  queryChars: 400,
  results: 50,
  domains: 10,
  domainChars: 253,
  extractPages: 5,
  extractChars: 8000,
} as const;

const PROVIDERS = ["duckduckgo", "bing", "yahoo"] as const;
const FRESHNESS = ["day", "week", "month", "year"] as const;
const EXTRACT_MODES = ["highlights", "markdown", "text", "metadata"] as const;
const ISO_CODE = /^[a-z]{2}$/;

function clampInt(value: unknown, min: number, max: number): number | undefined {
  if (typeof value !== "number" || !Number.isFinite(value)) return undefined;
  return Math.min(max, Math.max(min, Math.trunc(value)));
}

function pickEnum<T extends string>(value: unknown, allowed: readonly T[]): T | undefined {
  return typeof value === "string" && (allowed as readonly string[]).includes(value) ? (value as T) : undefined;
}

function domainList(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const domains = value
    .filter((d): d is string => typeof d === "string")
    .map((d) => d.trim().toLowerCase())
    .filter((d) => d.length > 0 && d.length <= DEMO_LIMITS.domainChars)
    .slice(0, DEMO_LIMITS.domains);
  return domains.length ? domains : undefined;
}

export type SanitizeResult = { ok: true; request: SearchRequest } | { ok: false; message: string };

/**
 * Rebuilds a search request from known fields only, clamping costly options.
 * The proxy authenticates with a server-side key, so it must never forward
 * arbitrary caller-controlled payloads to the backend.
 */
export function sanitizeSearchRequest(body: unknown): SanitizeResult {
  if (!body || typeof body !== "object" || Array.isArray(body)) {
    return { ok: false, message: "Request body must be a JSON object." };
  }
  const input = body as Record<string, unknown>;
  const query = typeof input.query === "string" ? input.query.trim() : "";
  if (!query) return { ok: false, message: "Field 'query' is required and must not be empty." };
  if (query.length > DEMO_LIMITS.queryChars) {
    return { ok: false, message: `Field 'query' must be at most ${DEMO_LIMITS.queryChars} characters.` };
  }

  const request: SearchRequest = { query };
  const limit = clampInt(input.limit, 1, DEMO_LIMITS.results);
  if (limit !== undefined) request.limit = limit;
  const offset = clampInt(input.offset, 0, DEMO_LIMITS.results - 1);
  if (offset) request.offset = offset;

  if (Array.isArray(input.providers)) {
    const providers = PROVIDERS.filter((p) => (input.providers as unknown[]).includes(p));
    if (providers.length) request.providers = providers;
  }
  const freshness = pickEnum(input.freshness, FRESHNESS);
  if (freshness) request.freshness = freshness;
  for (const key of ["country", "language"] as const) {
    const value = typeof input[key] === "string" ? (input[key] as string).trim().toLowerCase() : "";
    if (ISO_CODE.test(value)) request[key] = value;
  }
  const include = domainList(input.include_domains);
  if (include) request.include_domains = include;
  const exclude = domainList(input.exclude_domains);
  if (exclude) request.exclude_domains = exclude;

  const extract = input.extract as Record<string, unknown> | undefined;
  if (extract && typeof extract === "object" && extract.enabled === true) {
    const options: ExtractOptions = { enabled: true };
    const mode = pickEnum(extract.mode, EXTRACT_MODES);
    if (mode) options.mode = mode;
    const pages = clampInt(extract.max_pages, 1, DEMO_LIMITS.extractPages);
    if (pages !== undefined) options.max_pages = pages;
    const chars = clampInt(extract.max_chars_per_page, 100, DEMO_LIMITS.extractChars);
    if (chars !== undefined) options.max_chars_per_page = chars;
    request.extract = options;
  }

  return { ok: true, request };
}
