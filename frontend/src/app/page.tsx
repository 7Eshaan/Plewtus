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
import { ApiQuickstart } from "@/components/api-quickstart";
import { AsciiHeroField } from "@/components/ascii-hero-field";
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
const HERO_ACCENT_COLOR: [number, number, number] = [135 / 255, 0, 1];

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

      <main className="flex-1 w-full flex flex-col">
        {/* Full Viewport Hero with Interactive Skewed Background Boxes */}
        <section
          id="demo"
          className="relative w-full min-h-[calc(100vh-3.5rem)] min-h-[calc(100dvh-3.5rem)] flex flex-col items-center justify-center overflow-hidden scroll-mt-14"
        >
          {/* Interactive ASCII Fluid Flow Field Background (shadcn/ui style with #8700ff accent) */}
          <AsciiHeroField word="PLEWTUS METASEARCH " accentColor={HERO_ACCENT_COLOR} />

          {/* Centered Hero Content Container */}
          <div className="relative z-10 w-full max-w-4xl mx-auto px-4 py-8 sm:py-12 flex flex-col items-center text-center gap-5">
            <div className="flex flex-col items-center gap-2.5 max-w-2xl">
              <h1 className="text-3xl sm:text-5xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans text-balance">
                Federated Search for AI Agents
              </h1>

              <p className="text-sm sm:text-base text-zinc-500 dark:text-zinc-400 max-w-xl text-balance leading-relaxed">
                Query DuckDuckGo, Bing, and Yahoo simultaneously. Clean canonical URLs,
                multi-engine consensus, and Reciprocal Rank Fusion.
              </p>
            </div>

            {/* Centered Metasearch Playground Card with subtle #8700ff ambient glow */}
            <div className="relative w-full max-w-4xl mx-auto pt-2">
              <div
                className="absolute -inset-3 bg-gradient-to-r from-[#8700ff]/10 via-[#8700ff]/5 to-[#8700ff]/10 rounded-3xl blur-2xl opacity-70 dark:opacity-40 pointer-events-none -z-10"
                aria-hidden="true"
              />
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
            </div>
          </div>
        </section>

        {/* 2nd Page Container: Documentation, MCP & Rate Limits */}
        <div className="w-full max-w-4xl mx-auto px-4 flex flex-col gap-10 pb-16">
          {/* Section Divider - Appears on 2nd page scroll */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80" />

          {/* How It Works Section */}
          <HowItWorks />

          {/* Section Divider */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

          {/* Model Context Protocol (MCP) Section */}
          <McpSection />

          {/* Section Divider */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

          {/* API Quickstart & Code Integration Section */}
          <ApiQuickstart />
        </div>
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
            <a href="#demo" className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors">
              Search Demo
            </a>
            <a href="#how-it-works" className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors">
              How It Works
            </a>
            <a href="#mcp" className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors">
              MCP Stdio
            </a>
            <a href="#quickstart" className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors">
              API Quickstart
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
