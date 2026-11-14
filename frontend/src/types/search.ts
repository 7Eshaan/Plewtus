export type ProviderName = "duckduckgo" | "bing" | "yahoo" | "marginalia" | "brave" | "qwant";

export const ALL_PROVIDERS: ProviderName[] = ["duckduckgo", "bing", "yahoo", "marginalia", "brave", "qwant"];

/** Engines the backend queries when a request names none (PLEWTUS_DEFAULT_PROVIDERS). */
export const DEFAULT_PROVIDERS: ProviderName[] = ["duckduckgo", "bing", "yahoo"];

/** Extra context shown on engine toggles. */
export const PROVIDER_HINTS: Partial<Record<ProviderName, string>> = {
  marginalia: "Independent small-web index (mostly English). Only 'Past year' freshness is supported; the shared API key is often rate limited.",
  brave: "Independent index. Without a Brave API key on the server, Brave usually answers with a captcha.",
  qwant: "European engine. Only available in some countries; no freshness filter.",
};

export const PROVIDER_LABELS: Record<ProviderName, string> = {
  duckduckgo: "DuckDuckGo",
  bing: "Bing",
  yahoo: "Yahoo",
  marginalia: "Marginalia",
  brave: "Brave",
  qwant: "Qwant",
};

export type ExtractionMode = "none" | "highlights" | "markdown" | "text";

export type Freshness = "" | "day" | "week" | "month" | "year";

export interface AdvancedOptions {
  limit: number;
  freshness: Freshness;
  country: string;
  language: string;
  includeDomains: string;
  excludeDomains: string;
}

export const DEFAULT_ADVANCED_OPTIONS: AdvancedOptions = {
  limit: 10,
  freshness: "",
  country: "",
  language: "",
  includeDomains: "",
  excludeDomains: "",
};

export const LIMIT_CHOICES = [5, 10, 20, 50] as const;

/** Deepest result position the API serves (offset + limit). */
export const MAX_RESULT_WINDOW = 50;

/** Example queries used for the initial search and empty-state suggestions. */
export const EXAMPLE_QUERIES = [
  "reciprocal rank fusion algorithm",
  "golang concurrency patterns",
  "model context protocol stdio",
  "distributed consensus raft",
];
