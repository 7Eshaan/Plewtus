"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SearchPreview, type ActiveTab } from "@/components/search-preview";
import { buildSearchRequest } from "@/lib/build-request";
import {
  DEFAULT_ADVANCED_OPTIONS,
  DEFAULT_PROVIDERS,
  EXAMPLE_QUERIES,
  MAX_RESULT_WINDOW,
  type AdvancedOptions,
  type ExtractionMode,
  type ProviderName,
} from "@/types/search";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

const SESSION_KEY = "plewtus:last-search";

/** What a completed search was run with, to tell when the controls have since changed. */
interface SearchSnapshot {
  query: string;
  /** Serialized request (without offset); compared against the current controls. */
  signature: string;
  /** One API response per page fetched so far ("Load more" appends pages). */
  pages: SearchResponse[];
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
        const next: SearchSnapshot = { query: q, signature: JSON.stringify(request), pages: [result] };
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
        if (parsed?.query && Array.isArray(parsed.pages) && parsed.pages[0]?.results) {
          // eslint-disable-next-line react-hooks/set-state-in-effect -- one-time restore from storage
          setQuery(parsed.query);
          // Only the first page is restored; its signature matches the default controls.
          setSnapshot({ ...parsed, pages: [parsed.pages[0]] });
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
