"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { Header } from "@/components/header";
import { SearchPreview } from "@/components/search-preview";
import { type ProviderName } from "@/components/provider-chips";
import { type ExtractionMode } from "@/components/extraction-toggle";
import {
  type AdvancedOptions,
  DEFAULT_ADVANCED_OPTIONS,
} from "@/components/advanced-dropdown";
import { HowItWorks } from "@/components/how-it-works";
import { McpSection } from "@/components/mcp-section";
import { UsageLimits } from "@/components/usage-limits";
import type { components } from "@/types/api";

type SearchRequest = components["schemas"]["SearchRequest"];
type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];
type ActiveTab = "results" | "json" | "code";

const DEFAULT_QUERIES = [
  "reciprocal rank fusion algorithm",
  "golang concurrency patterns",
  "model context protocol stdio",
  "distributed consensus raft",
];

export default function HomePage() {
  const [query, setQuery] = useState("");
  const [providers, setProviders] = useState<ProviderName[]>([
    "duckduckgo",
    "bing",
    "yahoo",
  ]);
  const [extractionMode, setExtractionMode] = useState<ExtractionMode>("none");
  const [advancedOptions, setAdvancedOptions] = useState<AdvancedOptions>({
    ...DEFAULT_ADVANCED_OPTIONS,
  });

  const [activeTab, setActiveTab] = useState<ActiveTab>("results");
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [response, setResponse] = useState<SearchResponse | null>(null);
  const [error, setError] = useState<ErrorResponse | null>(null);

  const initialSearchDoneRef = useRef(false);

  const executeSearch = useCallback(
    async (searchQuery?: string) => {
      const q = (searchQuery ?? query).trim();
      if (!q || loading) return;

      setLoading(true);
      setError(null);
      setSearched(true);

      // Build payload strictly adhering to SearchRequest OpenAPI schema
      const payload: SearchRequest = {
        query: q,
        limit: advancedOptions.limit,
        offset: 0,
        providers: providers.length === 3 ? undefined : providers,
      };

      if (advancedOptions.freshness) {
        payload.freshness = advancedOptions.freshness;
      }

      if (advancedOptions.country.trim()) {
        payload.country = advancedOptions.country.trim().toLowerCase();
      }

      if (advancedOptions.language.trim()) {
        payload.language = advancedOptions.language.trim().toLowerCase();
      }

      if (advancedOptions.includeDomains.trim()) {
        payload.include_domains = advancedOptions.includeDomains
          .split(",")
          .map((d) => d.trim())
          .filter(Boolean);
      }

      if (advancedOptions.excludeDomains.trim()) {
        payload.exclude_domains = advancedOptions.excludeDomains
          .split(",")
          .map((d) => d.trim())
          .filter(Boolean);
      }

      if (extractionMode !== "none") {
        payload.extract = {
          enabled: true,
          mode: extractionMode,
          max_pages: advancedOptions.maxPages,
          max_chars_per_page: advancedOptions.maxCharsPerPage,
        };
      }

      try {
        const res = await fetch("/api/search", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(payload),
        });

        const data = await res.json();
        if (!res.ok) {
          setError(data as ErrorResponse);
          setResponse(null);
        } else {
          setResponse(data as SearchResponse);
          setError(null);
        }
      } catch (err) {
        setError({
          error: "network_error",
          message:
            err instanceof Error
              ? err.message
              : "Failed to connect to search proxy",
          code: 0,
        });
        setResponse(null);
      } finally {
        setLoading(false);
      }
    },
    [query, loading, providers, extractionMode, advancedOptions]
  );

  // Automatically execute a random default query on initial page load
  useEffect(() => {
    if (initialSearchDoneRef.current) return;
    initialSearchDoneRef.current = true;
    const randomQuery =
      DEFAULT_QUERIES[Math.floor(Math.random() * DEFAULT_QUERIES.length)];
    setQuery(randomQuery);
    executeSearch(randomQuery);
  }, [executeSearch]);

  function handleSuggestionSelect(suggestion: string) {
    setQuery(suggestion);
    executeSearch(suggestion);
  }

  return (
    <div className="min-h-screen flex flex-col bg-white dark:bg-[#09090b] text-zinc-900 dark:text-zinc-100 transition-colors">
      <Header />

      <main className="flex-1 max-w-4xl w-full mx-auto px-4 py-5 sm:py-8 flex flex-col gap-6 sm:gap-8">
        {/* Compact Hero Section */}
        <section id="demo" className="flex flex-col items-center text-center gap-2 pt-1 sm:pt-2">
          <h1 className="text-2xl sm:text-4xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans max-w-xl text-balance">
            Federated Search for AI Agents
          </h1>

          <p className="text-xs sm:text-sm text-zinc-500 dark:text-zinc-400 max-w-lg text-balance leading-relaxed">
            Query DuckDuckGo, Bing, and Yahoo simultaneously. Clean canonical URLs,
            multi-engine consensus, and Reciprocal Rank Fusion.
          </p>
        </section>

        {/* Search Box Preview & Showcase (matches attached reference style) */}
        <section className="w-full" aria-label="Metasearch playground">
          <SearchPreview
            query={query}
            onQueryChange={setQuery}
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
            searched={searched}
            onSelectSuggestion={handleSuggestionSelect}
          />
        </section>

        {/* Section Divider */}
        <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

        {/* How It Works Section */}
        <HowItWorks />

        {/* Section Divider */}
        <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

        {/* Model Context Protocol (MCP) Section */}
        <McpSection />

        {/* Section Divider */}
        <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

        {/* Usage & Rate Limits Section */}
        <UsageLimits />
      </main>

      {/* Footer */}
      <footer className="w-full border-t border-zinc-200 dark:border-zinc-800/80 py-8 text-xs font-mono text-zinc-500 dark:text-zinc-500 bg-zinc-50/50 dark:bg-zinc-950/40">
        <div className="max-w-4xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-zinc-900 dark:text-zinc-100">Plewtus Metasearch</span>
            <span>•</span>
            <span>AI-Agent Metasearch API</span>
          </div>

          <div className="flex items-center gap-4 text-xs">
            <a href="#demo" className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors">
              Search Demo
            </a>
            <a href="#how-it-works" className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors">
              How It Works
            </a>
            <a href="#mcp" className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors">
              MCP Stdio
            </a>
            <a href="#limits" className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors">
              Limits
            </a>
          </div>

          <div className="text-[11px] text-zinc-400 dark:text-zinc-600">
            DuckDuckGo • Bing • Yahoo
          </div>
        </div>
      </footer>
    </div>
  );
}
