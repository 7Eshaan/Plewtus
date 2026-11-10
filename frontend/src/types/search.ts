export type ProviderName = "duckduckgo" | "bing" | "yahoo";

export const ALL_PROVIDERS: ProviderName[] = ["duckduckgo", "bing", "yahoo"];

export const PROVIDER_LABELS: Record<ProviderName, string> = {
  duckduckgo: "DuckDuckGo",
  bing: "Bing",
  yahoo: "Yahoo",
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
