export type ProviderName = "duckduckgo" | "bing" | "yahoo";

export type ExtractionMode = "none" | "highlights" | "markdown";

export interface AdvancedOptions {
  limit: number;
  freshness: "" | "day" | "week" | "month" | "year";
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
