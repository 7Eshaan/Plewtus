import {
  DEFAULT_ADVANCED_OPTIONS,
  DEFAULT_PROVIDERS,
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
  options: AdvancedOptions,
  offset = 0
): SearchRequest {
  const payload: SearchRequest = { query: query.trim() };

  if (options.limit !== DEFAULT_ADVANCED_OPTIONS.limit) payload.limit = options.limit;
  // The backend's default engine set is implied; anything else is sent explicitly.
  const isDefaultSet =
    providers.length === DEFAULT_PROVIDERS.length && DEFAULT_PROVIDERS.every((p) => providers.includes(p));
  if (providers.length > 0 && !isDefaultSet) payload.providers = providers;
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
    payload.extract = { enabled: true, mode: extractionMode };
  }
  if (offset > 0) payload.offset = offset;
  return payload;
}
