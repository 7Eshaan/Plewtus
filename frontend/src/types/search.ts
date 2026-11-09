export type ProviderName = "duckduckgo" | "bing" | "yahoo";

export const ALL_PROVIDERS: ProviderName[] = ["duckduckgo", "bing", "yahoo"];

export const PROVIDER_LABELS: Record<ProviderName, string> = {
  duckduckgo: "DuckDuckGo",
  bing: "Bing",
  yahoo: "Yahoo",
};

export type ExtractionMode = "none" | "highlights" | "markdown";

export type Freshness = "" | "day" | "week" | "month" | "year";

export interface AdvancedOptions {
  limit: number;
  freshness: Freshness;
  country: string;
  language: string;
  includeDomains: string;
  excludeDomains: string;
  maxPages: number;
  maxCharsPerPage: number;
}

export const DEFAULT_ADVANCED_OPTIONS: AdvancedOptions = {
  limit: 10,
  freshness: "",
  country: "",
  language: "",
  includeDomains: "",
  excludeDomains: "",
  maxPages: 3,
  maxCharsPerPage: 4000,
};

export const LIMIT_CHOICES = [5, 10, 20, 50] as const;

/** Example queries used for the initial search and empty-state suggestions. */
export const EXAMPLE_QUERIES = [
  "reciprocal rank fusion algorithm",
  "golang concurrency patterns",
  "model context protocol stdio",
  "distributed consensus raft",
];
