"use client";

import { useState, useCallback } from "react";
import { Header } from "@/components/header";
import { SearchBar } from "@/components/search-bar";
import { type ProviderName } from "@/components/provider-chips";
import { type ExtractionMode } from "@/components/extraction-toggle";
import {
  type AdvancedOptions,
  DEFAULT_ADVANCED_OPTIONS,
} from "@/components/advanced-dropdown";
import { MetadataStrip } from "@/components/metadata-strip";
import { ResultCard } from "@/components/result-card";
import { ResultSkeleton } from "@/components/result-skeleton";
import { EmptyState } from "@/components/empty-state";
import { ErrorState } from "@/components/error-state";
import { JsonViewer } from "@/components/json-viewer";
import { CodeSnippets } from "@/components/code-snippets";
import { HowItWorks } from "@/components/how-it-works";
import { McpSection } from "@/components/mcp-section";
import { UsageLimits } from "@/components/usage-limits";
import { ListOrdered, FileJson, Code2 } from "lucide-react";
import type { components } from "@/types/api";

type SearchRequest = components["schemas"]["SearchRequest"];
type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];
type ActiveTab = "results" | "json" | "code";

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

  function handleSuggestionSelect(suggestion: string) {
    setQuery(suggestion);
    executeSearch(suggestion);
  }

  return (
    <div className="min-h-screen flex flex-col bg-white dark:bg-[#09090b] text-zinc-900 dark:text-zinc-100 transition-colors">
      <Header />

      <main className="flex-1 max-w-4xl w-full mx-auto px-4 py-8 sm:py-12 flex flex-col gap-10">
        {/* Hero Section */}
        <section id="demo" className="flex flex-col items-center text-center gap-3 pt-2 sm:pt-4">
          <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-mono border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
            <span>Multi-Engine Metasearch Engine</span>
          </div>

          <h1 className="text-3xl sm:text-5xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans max-w-2xl text-balance">
            Federated Search for AI Agents
          </h1>

          <p className="text-sm sm:text-base text-zinc-500 dark:text-zinc-400 max-w-xl text-balance leading-relaxed">
            Query DuckDuckGo, Bing, and Yahoo simultaneously. Clean canonical URLs,
            multi-engine consensus, and Reciprocal Rank Fusion.
          </p>
        </section>

        {/* Hero Search Bar with Attached Options */}
        <section className="w-full max-w-3xl mx-auto" aria-label="Search controls">
          <SearchBar
            query={query}
            onChange={setQuery}
            onSubmit={() => executeSearch()}
            loading={loading}
            providers={providers}
            onProvidersChange={setProviders}
            extractionMode={extractionMode}
            onExtractionModeChange={setExtractionMode}
            advancedOptions={advancedOptions}
            onAdvancedOptionsChange={setAdvancedOptions}
          />
        </section>

        {/* Main Search Demonstration Area with Tabs */}
        <section className="w-full flex flex-col gap-5" aria-label="Search results and developer tools">
          {/* Developer Tools Tabs Bar */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-200 dark:border-zinc-800 pb-3">
            <div
              className="inline-flex items-center p-0.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-100/70 dark:bg-zinc-900/70 text-xs font-mono select-none"
              role="tablist"
              aria-label="Result formats"
            >
              <button
                type="button"
                role="tab"
                aria-selected={activeTab === "results"}
                onClick={() => setActiveTab("results")}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all cursor-pointer ${
                  activeTab === "results"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                }`}
              >
                <ListOrdered className="w-3.5 h-3.5" />
                <span>Ranked Results</span>
                {response && response.results.length > 0 && (
                  <span className="text-[10px] font-bold px-1.5 py-0.2 rounded-full bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-300">
                    {response.results.length}
                  </span>
                )}
              </button>

              <button
                type="button"
                role="tab"
                aria-selected={activeTab === "json"}
                onClick={() => setActiveTab("json")}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all cursor-pointer ${
                  activeTab === "json"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                }`}
              >
                <FileJson className="w-3.5 h-3.5 text-emerald-500" />
                <span>JSON Response</span>
                {response && (
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                )}
              </button>

              <button
                type="button"
                role="tab"
                aria-selected={activeTab === "code"}
                onClick={() => setActiveTab("code")}
                className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md transition-all cursor-pointer ${
                  activeTab === "code"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                }`}
              >
                <Code2 className="w-3.5 h-3.5" />
                <span>API Code</span>
              </button>
            </div>

            {/* Hint for developers */}
            <div className="hidden sm:flex items-center gap-2 text-xs font-mono text-zinc-400 dark:text-zinc-500">
              <span>Direct API endpoint:</span>
              <code className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 text-zinc-700 dark:text-zinc-300">
                POST /search
              </code>
            </div>
          </div>

          {/* TAB 1: Ranked Results View */}
          {activeTab === "results" && (
            <div className="flex flex-col gap-4">
              {/* Loading Skeleton */}
              {loading && <ResultSkeleton />}

              {/* Error Banner */}
              {!loading && error && (
                <ErrorState error={error} onRetry={() => executeSearch()} />
              )}

              {/* Successful Results */}
              {!loading && !error && response && (
                <div className="flex flex-col gap-4">
                  {/* Metadata Strip */}
                  <MetadataStrip metadata={response.metadata} />

                  {/* Result Cards or Zero Results */}
                  {response.results.length > 0 ? (
                    <div className="flex flex-col gap-3">
                      {response.results.map((result) => (
                        <ResultCard key={result.id} result={result} />
                      ))}
                    </div>
                  ) : (
                    <EmptyState searched={true} query={response.query} />
                  )}
                </div>
              )}

              {/* Initial Pre-Search State */}
              {!loading && !searched && (
                <EmptyState
                  searched={false}
                  onSelectSuggestion={handleSuggestionSelect}
                />
              )}
            </div>
          )}

          {/* TAB 2: Real API JSON Response View */}
          {activeTab === "json" && (
            <div className="flex flex-col gap-4">
              <JsonViewer response={response} loading={loading} />
            </div>
          )}

          {/* TAB 3: Dynamic Code Snippets View */}
          {activeTab === "code" && (
            <div className="flex flex-col gap-4">
              <CodeSnippets
                query={query}
                providers={providers}
                extractionMode={extractionMode}
                advancedOptions={advancedOptions}
              />
            </div>
          )}
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
