import {
  ALL_PROVIDERS,
  DEFAULT_ADVANCED_OPTIONS,
  type AdvancedOptions,
  type ExtractionMode,
  type ProviderName,
} from "@/types/search";
import type { SearchRequest } from "@/lib/sanitize-request";

function splitDomains(value: string): string[] {
  return value
    .split(",")
    .map((d) => d.trim())
    .filter(Boolean);
}

/**
 * Builds the API payload for the playground state. Options left at their
 * server-side defaults are omitted so generated code snippets stay minimal.
 */
export function buildSearchRequest(
  query: string,
  providers: ProviderName[],
  extractionMode: ExtractionMode,
  options: AdvancedOptions
): SearchRequest {
  const payload: SearchRequest = { query: query.trim() };

  if (options.limit !== DEFAULT_ADVANCED_OPTIONS.limit) payload.limit = options.limit;
  if (providers.length > 0 && providers.length < ALL_PROVIDERS.length) payload.providers = providers;
  if (options.freshness) payload.freshness = options.freshness;

  const country = options.country.trim().toLowerCase();
  if (country) payload.country = country;
  const language = options.language.trim().toLowerCase();
  if (language) payload.language = language;

  const include = splitDomains(options.includeDomains);
  if (include.length) payload.include_domains = include;
  const exclude = splitDomains(options.excludeDomains);
  if (exclude.length) payload.exclude_domains = exclude;

  if (extractionMode !== "none") {
    payload.extract = {
      enabled: true,
      mode: extractionMode,
      max_pages: options.maxPages,
      max_chars_per_page: options.maxCharsPerPage,
    };
  }
  return payload;
}
