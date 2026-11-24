"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SearchPreview, type ActiveTab } from "@/components/search-preview";
import { buildSearchRequest } from "@/lib/build-request";
import {
  ALL_PROVIDERS,
  DEFAULT_ADVANCED_OPTIONS,
  DEFAULT_PROVIDERS,
  EXAMPLE_QUERIES,
  LIMIT_CHOICES,
  MAX_RESULT_WINDOW,
  type AdvancedOptions,
  type ExtractionMode,
  type ProviderName,
} from "@/types/search";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

const SESSION_KEY = "plewtus:last-search";

/** The option controls a search was run with. */
interface SearchControls {
  providers: ProviderName[];
  extractionMode: ExtractionMode;
  advancedOptions: AdvancedOptions;
}

/** What a completed search was run with, to tell when the controls have since changed. */
interface SearchSnapshot {
  query: string;
  /** Serialized request (without offset); compared against the current controls. */
  signature: string;
  /** One API response per page fetched so far ("Load more" appends pages). */
  pages: SearchResponse[];
  /** The controls behind `signature`, restored with the search after a reload. */
  controls: SearchControls;
}

const EXTRACTION_MODES: readonly ExtractionMode[] = ["none", "highlights", "markdown", "text"];
const FRESHNESS_VALUES: readonly string[] = ["", "day", "week", "month", "year"];

/**
 * Validates controls read back from sessionStorage, so a stale or edited entry
 * cannot put the playground into a state its controls cannot show.
 */
function parseControls(value: unknown): SearchControls | null {
  if (!value || typeof value !== "object") return null;
  const v = value as Record<string, unknown>;
  const providers = Array.isArray(v.providers)
    ? ALL_PROVIDERS.filter((p) => (v.providers as unknown[]).includes(p))
    : [];
  const mode = EXTRACTION_MODES.find((m) => m === v.extractionMode);
  const o = v.advancedOptions as Record<string, unknown> | undefined;
  if (providers.length === 0 || !mode || !o || typeof o !== "object") return null;

  const text = (key: keyof AdvancedOptions) => (typeof o[key] === "string" ? (o[key] as string) : "");
  const limit = LIMIT_CHOICES.find((n) => n === o.limit) ?? DEFAULT_ADVANCED_OPTIONS.limit;
  const freshness = FRESHNESS_VALUES.includes(text("freshness")) ? (text("freshness") as AdvancedOptions["freshness"]) : "";
  return {
    providers,
    extractionMode: mode,
    advancedOptions: {
      limit,
      freshness,
      country: text("country").toLowerCase().replace(/[^a-z]/g, "").slice(0, 2),
      language: text("language").toLowerCase().replace(/[^a-z]/g, "").slice(0, 2),
      includeDomains: text("includeDomains"),
      excludeDomains: text("excludeDomains"),
    },
  };
}

function isErrorResponse(value: unknown): value is ErrorResponse {
  return Boolean(value && typeof value === "object" && "error" in value && "message" in value);
}

async function postSearch(body: unknown, signal: AbortSignal): Promise<SearchResponse | ErrorResponse> {
  const res = await fetch("/api/search", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
  let data: unknown = null;
  try {
    data = await res.json();
  } catch {
    // Non-JSON body; handled below.
  }
  if (!res.ok || !data) {
    return isErrorResponse(data)
      ? data
      : { error: "unexpected_response", message: `The search proxy returned HTTP ${res.status}.`, code: res.status };
  }
  return data as SearchResponse;
}

export function Playground() {
  const [query, setQuery] = useState("");
  const [providers, setProviders] = useState<ProviderName[]>(DEFAULT_PROVIDERS);
  const [extractionMode, setExtractionMode] = useState<ExtractionMode>("none");
  const [advancedOptions, setAdvancedOptions] = useState<AdvancedOptions>({ ...DEFAULT_ADVANCED_OPTIONS });
  const [activeTab, setActiveTab] = useState<ActiveTab>("results");
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [snapshot, setSnapshot] = useState<SearchSnapshot | null>(null);
  const [error, setError] = useState<ErrorResponse | null>(null);

  const abortRef = useRef<AbortController | null>(null);
  const initializedRef = useRef(false);

  const handleQueryChange = useCallback((nextQuery: string) => {
    setQuery(nextQuery);
    setError(null);
  }, []);

  const executeSearch = useCallback(
    async (searchQuery?: string) => {
      const q = (searchQuery ?? query).trim();
      if (!q) return;

      // A new search supersedes any request still in flight.
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setLoading(true);
      setLoadingMore(false);
      setError(null);

      const request = buildSearchRequest(q, providers, extractionMode, advancedOptions);
      try {
        const result = await postSearch(request, controller.signal);
        if (isErrorResponse(result)) {
          setError(result);
          setSnapshot(null);
          return;
        }
        const next: SearchSnapshot = {
          query: q,
          signature: JSON.stringify(request),
          pages: [result],
          controls: { providers, extractionMode, advancedOptions },
        };
        setSnapshot(next);
        try {
          sessionStorage.setItem(SESSION_KEY, JSON.stringify(next));
        } catch {
          // Storage may be full or disabled; restoring is best-effort.
        }
      } catch (err) {
        if (controller.signal.aborted) return;
        setError({
          error: "network_error",
          message: err instanceof Error ? err.message : "Failed to reach the search proxy.",
          code: 0,
        });
        setSnapshot(null);
      } finally {
        if (abortRef.current === controller) {
          abortRef.current = null;
          setLoading(false);
        }
      }
    },
    [query, providers, extractionMode, advancedOptions]
  );

  const results = useMemo(() => snapshot?.pages.flatMap((page) => page.results) ?? [], [snapshot]);
  const lastPage = snapshot ? snapshot.pages[snapshot.pages.length - 1] : null;

  // The controls no longer describe the results on screen.
  const currentSignature = snapshot
    ? JSON.stringify(buildSearchRequest(snapshot.query, providers, extractionMode, advancedOptions))
    : "";
  const stale = Boolean(snapshot) && currentSignature !== snapshot?.signature;

  const pageSize = advancedOptions.limit;
  // Engines return more on deeper pages than the first page reveals, so offer
  // more until a page comes back empty or the result window is exhausted.
  const canLoadMore =
    Boolean(lastPage) && !stale && lastPage!.results.length > 0 && results.length < MAX_RESULT_WINDOW;

  const loadMore = useCallback(async () => {
    if (!snapshot || loading || loadingMore) return;
    const offset = snapshot.pages.reduce((n, page) => n + page.results.length, 0);
    const limit = Math.min(pageSize, MAX_RESULT_WINDOW - offset);
    if (limit <= 0) return;

    const controller = new AbortController();
    abortRef.current = controller;
    setLoadingMore(true);
    setError(null);
    try {
      const request = { ...buildSearchRequest(snapshot.query, providers, extractionMode, advancedOptions, offset), limit };
      const result = await postSearch(request, controller.signal);
      if (isErrorResponse(result)) {
        setError(result);
        return;
      }
      setSnapshot((prev) => (prev ? { ...prev, pages: [...prev.pages, result] } : prev));
    } catch (err) {
      if (controller.signal.aborted) return;
      setError({
        error: "network_error",
        message: err instanceof Error ? err.message : "Failed to reach the search proxy.",
        code: 0,
      });
    } finally {
      if (abortRef.current === controller) abortRef.current = null;
      setLoadingMore(false);
    }
  }, [snapshot, loading, loadingMore, pageSize, providers, extractionMode, advancedOptions]);

  // On first load, restore this tab's last search; otherwise run one example
  // query so the playground shows live results without a search per reload.
  useEffect(() => {
    if (initializedRef.current) return;
    initializedRef.current = true;
    try {
      const saved = sessionStorage.getItem(SESSION_KEY);
      if (saved) {
        const parsed = JSON.parse(saved) as SearchSnapshot;
        // Entries saved before the controls were stored are skipped: their
        // options are unknown, so they would show as stale straight away.
        const controls = parseControls(parsed?.controls);
        if (parsed?.query && controls && Array.isArray(parsed.pages) && parsed.pages[0]?.results) {
          // Restore the controls with the results, so the signature matches
          // and the "Options changed" notice only appears after a real change.
          // eslint-disable-next-line react-hooks/set-state-in-effect -- one-time restore from storage
          setQuery(parsed.query);
          setProviders(controls.providers);
          setExtractionMode(controls.extractionMode);
          setAdvancedOptions(controls.advancedOptions);
          // Only the first page is restored ("Load more" fetches the rest again).
          const signature = JSON.stringify(
            buildSearchRequest(parsed.query, controls.providers, controls.extractionMode, controls.advancedOptions)
          );
          setSnapshot({ query: parsed.query, signature, pages: [parsed.pages[0]], controls });
          return;
        }
      }
    } catch {
      // Ignore malformed storage and fall through to a live search.
    }
    const example = EXAMPLE_QUERIES[Math.floor(Math.random() * EXAMPLE_QUERIES.length)];
    setQuery(example);
    void executeSearch(example);
  }, [executeSearch]);

  useEffect(() => () => abortRef.current?.abort(), []);

  const runSuggestion = useCallback(
    (suggestion: string) => {
      setQuery(suggestion);
      void executeSearch(suggestion);
    },
    [executeSearch]
  );

  return (
    <SearchPreview
      query={query}
      onQueryChange={handleQueryChange}
      onSearch={executeSearch}
      loading={loading}
      providers={providers}
      onProvidersChange={setProviders}
      extractionMode={extractionMode}
      onExtractionModeChange={setExtractionMode}
      advancedOptions={advancedOptions}
      onAdvancedOptionsChange={setAdvancedOptions}
      activeTab={activeTab}
      onTabChange={setActiveTab}
      response={lastPage}
      results={results}
      searchedQuery={snapshot?.query ?? ""}
      stale={stale}
      canLoadMore={canLoadMore}
      loadingMore={loadingMore}
      onLoadMore={loadMore}
      error={error}
      onSelectSuggestion={runSuggestion}
    />
  );
}
