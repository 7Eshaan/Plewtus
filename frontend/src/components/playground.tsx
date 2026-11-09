"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { SearchPreview, type ActiveTab } from "@/components/search-preview";
import { buildSearchRequest } from "@/lib/build-request";
import {
  ALL_PROVIDERS,
  DEFAULT_ADVANCED_OPTIONS,
  EXAMPLE_QUERIES,
  type AdvancedOptions,
  type ExtractionMode,
  type ProviderName,
} from "@/types/search";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

const SESSION_KEY = "plewtus:last-search";

function isErrorResponse(value: unknown): value is ErrorResponse {
  return Boolean(value && typeof value === "object" && "error" in value && "message" in value);
}

export function Playground() {
  const [query, setQuery] = useState("");
  const [providers, setProviders] = useState<ProviderName[]>(ALL_PROVIDERS);
  const [extractionMode, setExtractionMode] = useState<ExtractionMode>("none");
  const [advancedOptions, setAdvancedOptions] = useState<AdvancedOptions>({ ...DEFAULT_ADVANCED_OPTIONS });
  const [activeTab, setActiveTab] = useState<ActiveTab>("results");
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<SearchResponse | null>(null);
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
      setError(null);

      try {
        const res = await fetch("/api/search", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(buildSearchRequest(q, providers, extractionMode, advancedOptions)),
          signal: controller.signal,
        });

        let data: unknown;
        try {
          data = await res.json();
        } catch {
          data = null;
        }

        if (!res.ok || !data) {
          setError(
            isErrorResponse(data)
              ? data
              : { error: "unexpected_response", message: `The search proxy returned HTTP ${res.status}.`, code: res.status }
          );
          setResponse(null);
          return;
        }

        const result = data as SearchResponse;
        setResponse(result);
        try {
          sessionStorage.setItem(SESSION_KEY, JSON.stringify({ query: q, response: result }));
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
        setResponse(null);
      } finally {
        if (abortRef.current === controller) {
          abortRef.current = null;
          setLoading(false);
        }
      }
    },
    [query, providers, extractionMode, advancedOptions]
  );

  // On first load, restore this tab's last search; otherwise run one example
  // query so the playground shows live results without a search per reload.
  useEffect(() => {
    if (initializedRef.current) return;
    initializedRef.current = true;
    try {
      const saved = sessionStorage.getItem(SESSION_KEY);
      if (saved) {
        const parsed = JSON.parse(saved) as { query: string; response: SearchResponse };
        if (parsed?.query && parsed.response?.results) {
          // eslint-disable-next-line react-hooks/set-state-in-effect -- one-time restore from storage
          setQuery(parsed.query);
          setResponse(parsed.response);
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
      response={response}
      error={error}
      onSelectSuggestion={runSuggestion}
    />
  );
}
