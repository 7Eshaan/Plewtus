"use client";

import { useState, useRef, useEffect } from "react";
import {
  Search,
  ArrowUp,
  X,
  ExternalLink,
  Code2,
  FileJson,
  ListOrdered,
  Loader2,
  SlidersHorizontal,
  ChevronDown,
  RotateCcw,
} from "lucide-react";
import {
  type ProviderName,
  type ExtractionMode,
  type AdvancedOptions,
  DEFAULT_ADVANCED_OPTIONS,
} from "@/types/search";
import { ResultCard } from "./result-card";
import { ResultSkeleton } from "./result-skeleton";
import { EmptyState } from "./empty-state";
import { ErrorState } from "./error-state";
import { JsonViewer } from "./json-viewer";
import { CodeSnippets } from "./code-snippets";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];
type ActiveTab = "results" | "json" | "code";

interface SearchPreviewProps {
  query: string;
  onQueryChange: (query: string) => void;
  onSearch: (customQuery?: string) => void;
  loading: boolean;
  providers: ProviderName[];
  onProvidersChange: (providers: ProviderName[]) => void;
  extractionMode: ExtractionMode;
  onExtractionModeChange: (mode: ExtractionMode) => void;
  advancedOptions: AdvancedOptions;
  onAdvancedOptionsChange: (options: AdvancedOptions) => void;
  activeTab: ActiveTab;
  onTabChange: (tab: ActiveTab) => void;
  response: SearchResponse | null;
  error: ErrorResponse | null;
  searched: boolean;
  onSelectSuggestion: (suggestion: string) => void;
}

export function SearchPreview({
  query,
  onQueryChange,
  onSearch,
  loading,
  providers,
  onProvidersChange,
  extractionMode,
  onExtractionModeChange,
  advancedOptions,
  onAdvancedOptionsChange,
  activeTab,
  onTabChange,
  response,
  error,
  onSelectSuggestion,
}: SearchPreviewProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);

  // Keyboard shortcut listener: pressing "/" or "cmd+k" focuses the search bar
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (
        (e.key === "/" || ((e.metaKey || e.ctrlKey) && e.key === "k")) &&
        document.activeElement !== inputRef.current &&
        !["INPUT", "TEXTAREA"].includes(
          (document.activeElement as HTMLElement)?.tagName
        )
      ) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  function handleFormSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim() || loading) return;
    onSearch();
  }

  function handleClear() {
    onQueryChange("");
    inputRef.current?.focus();
  }

  function handleToggleProvider(p: ProviderName) {
    if (providers.includes(p)) {
      if (providers.length === 1) return; // Prevent disabling all
      onProvidersChange(providers.filter((item) => item !== p));
    } else {
      onProvidersChange([...providers, p]);
    }
  }

  function handleToggleAllProviders() {
    if (providers.length === 3) {
      onProvidersChange(["duckduckgo"]);
    } else {
      onProvidersChange(["duckduckgo", "bing", "yahoo"]);
    }
  }

  return (
    <div className="w-full max-w-4xl mx-auto rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#0c0c0e] shadow-lg overflow-hidden text-left transition-colors">
      {/* 1. TOP SEARCH INPUT BAR (matches Exa.ai spacing & layout) */}
      <form
        onSubmit={handleFormSubmit}
        className="px-4 py-3 sm:py-3.5 flex items-center gap-3 bg-white dark:bg-[#111114] border-b border-zinc-200 dark:border-zinc-800/80 focus-within:border-[#8700ff]/50 transition-colors"
        role="search"
        aria-label="Metasearch input"
      >
        <div className="flex items-center justify-center text-zinc-400 dark:text-zinc-500 pl-0.5 pointer-events-none">
          <Search className="w-4 h-4" />
        </div>

        {/* Main query input */}
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder="Search topics, queries, or docs across DuckDuckGo, Bing & Yahoo..."
          disabled={loading}
          className="flex-1 text-sm sm:text-base font-medium text-zinc-950 dark:text-zinc-50 placeholder-zinc-400 dark:placeholder-zinc-600 bg-transparent outline-none font-sans"
        />

        {/* Right Action buttons: Clear + Submit */}
        <div className="flex items-center gap-2">
          {query.length > 0 && !loading && (
            <button
              type="button"
              onClick={handleClear}
              className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
              title="Clear search"
              aria-label="Clear input"
            >
              <X className="w-4 h-4" />
            </button>
          )}

          <button
            type="submit"
            disabled={loading || !query.trim()}
            className="w-8 h-8 rounded-lg bg-[#8700ff] hover:bg-[#9924ff] active:scale-95 text-white shadow-xs shadow-[#8700ff]/25 transition-all flex items-center justify-center cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-[#8700ff]"
            title="Execute search"
            aria-label="Submit search"
          >
            {loading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <ArrowUp className="w-4 h-4 stroke-[2.5]" />
            )}
          </button>
        </div>
      </form>

      {/* 2. 2-COLUMN PLAYGROUND BODY (matches Exa.ai vertical height ~390px) */}
      <div className="flex flex-col md:flex-row md:h-[390px]">
        {/* LEFT COLUMN: CONTROLS */}
        <aside
          className="w-full md:w-64 lg:w-72 border-b md:border-b-0 md:border-r border-zinc-200 dark:border-zinc-800/80 p-3.5 sm:p-4 flex flex-col justify-between bg-zinc-50/50 dark:bg-zinc-950/20 overflow-y-auto flex-shrink-0"
          aria-label="Search parameters"
        >
          <div className="flex flex-col gap-3.5">
            {/* Real Feature 1: EXTRACTION MODE */}
            <div className="flex flex-col gap-1.5">
              <span className="text-[10px] font-mono tracking-wider text-zinc-400 dark:text-zinc-500 font-semibold uppercase">
                Extraction
              </span>
              <div className="grid grid-cols-3 p-1 rounded-lg bg-zinc-100 dark:bg-zinc-900 border border-zinc-200/80 dark:border-zinc-800 text-xs font-mono select-none">
                <button
                  type="button"
                  onClick={() => onExtractionModeChange("none")}
                  className={`py-1.5 px-2 rounded-md text-center transition-all cursor-pointer ${
                    extractionMode === "none"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                  title="Snippets only (fastest)"
                >
                  None
                </button>

                <button
                  type="button"
                  onClick={() => onExtractionModeChange("highlights")}
                  className={`py-1.5 px-2 rounded-md text-center transition-all cursor-pointer ${
                    extractionMode === "highlights"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                  title="Extract query-weighted highlights"
                >
                  Highlights
                </button>

                <button
                  type="button"
                  onClick={() => onExtractionModeChange("markdown")}
                  className={`py-1.5 px-2 rounded-md text-center transition-all cursor-pointer ${
                    extractionMode === "markdown"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200"
                  }`}
                  title="Extract full cleaned webpage Markdown"
                >
                  Markdown
                </button>
              </div>
            </div>

            {/* Real Feature 2: ENGINES (DuckDuckGo, Bing, Yahoo) */}
            <div className="flex flex-col gap-1.5">
              <span className="text-[10px] font-mono tracking-wider text-zinc-400 dark:text-zinc-500 font-semibold uppercase">
                Engines
              </span>
              <div className="grid grid-cols-2 gap-1.5 text-xs font-mono select-none">
                <button
                  type="button"
                  onClick={handleToggleAllProviders}
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    providers.length === 3
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  All (3)
                </button>

                <button
                  type="button"
                  onClick={() => handleToggleProvider("duckduckgo")}
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    providers.includes("duckduckgo")
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  DuckDuckGo
                </button>

                <button
                  type="button"
                  onClick={() => handleToggleProvider("bing")}
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    providers.includes("bing")
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Bing
                </button>

                <button
                  type="button"
                  onClick={() => handleToggleProvider("yahoo")}
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    providers.includes("yahoo")
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Yahoo
                </button>
              </div>
            </div>

            {/* Real Feature 3: FRESHNESS */}
            <div className="flex flex-col gap-1.5">
              <span className="text-[10px] font-mono tracking-wider text-zinc-400 dark:text-zinc-500 font-semibold uppercase">
                Freshness
              </span>
              <div className="grid grid-cols-2 gap-1.5 text-xs font-mono select-none">
                <button
                  type="button"
                  onClick={() =>
                    onAdvancedOptionsChange({ ...advancedOptions, freshness: "" })
                  }
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    advancedOptions.freshness === ""
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Anytime
                </button>

                <button
                  type="button"
                  onClick={() =>
                    onAdvancedOptionsChange({
                      ...advancedOptions,
                      freshness: "day",
                    })
                  }
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    advancedOptions.freshness === "day"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Past 24h
                </button>

                <button
                  type="button"
                  onClick={() =>
                    onAdvancedOptionsChange({
                      ...advancedOptions,
                      freshness: "week",
                    })
                  }
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    advancedOptions.freshness === "week"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Past Week
                </button>

                <button
                  type="button"
                  onClick={() =>
                    onAdvancedOptionsChange({
                      ...advancedOptions,
                      freshness: "month",
                    })
                  }
                  className={`py-1.5 px-2.5 rounded-lg border text-center transition-all cursor-pointer ${
                    advancedOptions.freshness === "month"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
                      : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700"
                  }`}
                >
                  Past Month
                </button>
              </div>
            </div>

            {/* Real Feature 4: Collapsible Advanced Parameters */}
            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => setAdvancedOpen(!advancedOpen)}
                className="inline-flex items-center justify-between px-2.5 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 text-xs font-mono text-zinc-600 dark:text-zinc-400 hover:text-zinc-950 dark:hover:text-zinc-100 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors cursor-pointer select-none"
              >
                <div className="inline-flex items-center gap-1.5">
                  <SlidersHorizontal className="w-3.5 h-3.5" />
                  <span>Advanced Settings</span>
                </div>
                <ChevronDown
                  className={`w-3.5 h-3.5 transition-transform ${
                    advancedOpen ? "rotate-180" : ""
                  }`}
                />
              </button>

              {advancedOpen && (
                <div className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 flex flex-col gap-2 text-xs animate-in fade-in slide-in-from-top-1">
                  <div className="flex items-center justify-between pb-1 border-b border-zinc-100 dark:border-zinc-800">
                    <span className="font-mono text-[9px] text-zinc-400 uppercase">
                      Tunables
                    </span>
                    <button
                      type="button"
                      onClick={() =>
                        onAdvancedOptionsChange({ ...DEFAULT_ADVANCED_OPTIONS })
                      }
                      className="text-[9px] font-mono text-zinc-500 hover:text-zinc-950 dark:hover:text-zinc-100 inline-flex items-center gap-1 cursor-pointer"
                    >
                      <RotateCcw className="w-2.5 h-2.5" />
                      Reset
                    </button>
                  </div>

                  <div className="flex flex-col gap-0.5">
                    <label className="text-[10px] font-mono text-zinc-500">
                      Results Limit (1–50): {advancedOptions.limit}
                    </label>
                    <input
                      type="range"
                      min={1}
                      max={50}
                      value={advancedOptions.limit}
                      onChange={(e) =>
                        onAdvancedOptionsChange({
                          ...advancedOptions,
                          limit: parseInt(e.target.value, 10),
                        })
                      }
                      className="w-full accent-[#8700ff] h-1.5 cursor-pointer"
                    />
                  </div>

                  <div className="flex flex-col gap-0.5">
                    <label className="text-[10px] font-mono text-zinc-500">
                      Country ISO (e.g. us, de)
                    </label>
                    <input
                      type="text"
                      maxLength={2}
                      value={advancedOptions.country}
                      onChange={(e) =>
                        onAdvancedOptionsChange({
                          ...advancedOptions,
                          country: e.target.value
                            .toLowerCase()
                            .replace(/[^a-z]/g, ""),
                        })
                      }
                      placeholder="e.g. us"
                      className="px-2 py-1 rounded bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-xs font-mono uppercase"
                    />
                  </div>

                  <div className="flex flex-col gap-0.5">
                    <label className="text-[10px] font-mono text-zinc-500">
                      Include Domains (comma-separated)
                    </label>
                    <input
                      type="text"
                      value={advancedOptions.includeDomains}
                      onChange={(e) =>
                        onAdvancedOptionsChange({
                          ...advancedOptions,
                          includeDomains: e.target.value,
                        })
                      }
                      placeholder="e.g. pkg.go.dev"
                      className="px-2 py-1 rounded bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-xs font-mono"
                    />
                  </div>

                  <div className="flex flex-col gap-0.5">
                    <label className="text-[10px] font-mono text-zinc-500">
                      Exclude Domains (comma-separated)
                    </label>
                    <input
                      type="text"
                      value={advancedOptions.excludeDomains}
                      onChange={(e) =>
                        onAdvancedOptionsChange({
                          ...advancedOptions,
                          excludeDomains: e.target.value,
                        })
                      }
                      placeholder="e.g. pinterest.com"
                      className="px-2 py-1 rounded bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-xs font-mono"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Bottom Actions matching Exa layout */}
          <div className="flex items-center gap-2 pt-3 border-t border-zinc-200 dark:border-zinc-800/80 mt-2">
            <a
              href="#mcp"
              className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-xs font-mono text-zinc-700 dark:text-zinc-300 hover:border-zinc-400 dark:hover:border-zinc-600 hover:text-zinc-950 dark:hover:text-white transition-all shadow-2xs select-none"
            >
              <span>API Playground</span>
              <ExternalLink className="w-3 h-3 text-zinc-400" />
            </a>

            <button
              type="button"
              onClick={() => onTabChange("code")}
              className={`p-1.5 rounded-lg border transition-all cursor-pointer ${
                activeTab === "code"
                  ? "bg-white dark:bg-zinc-800 border-[#8700ff]/60 text-[#8700ff] dark:text-[#a84dff] shadow-xs"
                  : "bg-white dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800 text-zinc-500 dark:text-zinc-400 hover:text-zinc-950 dark:hover:text-zinc-200"
              }`}
              title="View API Code Snippets"
              aria-label="View API Code Snippets"
            >
              <Code2 className="w-3.5 h-3.5" />
            </button>
          </div>
        </aside>

        {/* RIGHT COLUMN: OUTPUT & RESULTS SHOWCASE (matches Exa.ai: 3 cards + 1 peeking) */}
        <div className="flex-1 p-3.5 sm:p-4 flex flex-col gap-2.5 overflow-hidden bg-white/60 dark:bg-[#0c0c0e]/60 min-h-0">
          {/* Header Row: OUTPUT label + Tabs */}
          <div className="flex items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-2 flex-shrink-0">
            <div className="flex items-center gap-2.5">
              <span className="text-[10px] font-mono tracking-wider text-zinc-400 dark:text-zinc-500 font-semibold uppercase hidden sm:inline">
                Output
              </span>

              {/* Segmented output tabs */}
              <div
                className="inline-flex items-center p-0.5 rounded-lg bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 text-xs font-mono select-none"
                role="tablist"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={activeTab === "results"}
                  onClick={() => onTabChange("results")}
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md transition-all cursor-pointer ${
                    activeTab === "results"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  <ListOrdered className="w-3.5 h-3.5" />
                  <span>Results</span>
                  {response && response.results.length > 0 && (
                    <span className="text-[10px] font-bold px-1.5 rounded-full bg-[#8700ff]/10 text-[#8700ff] dark:text-[#a84dff] border border-[#8700ff]/20">
                      {response.results.length}
                    </span>
                  )}
                </button>

                <button
                  type="button"
                  role="tab"
                  aria-selected={activeTab === "json"}
                  onClick={() => onTabChange("json")}
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md transition-all cursor-pointer ${
                    activeTab === "json"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  <FileJson className="w-3.5 h-3.5 text-emerald-500" />
                  <span>JSON</span>
                </button>

                <button
                  type="button"
                  role="tab"
                  aria-selected={activeTab === "code"}
                  onClick={() => onTabChange("code")}
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md transition-all cursor-pointer ${
                    activeTab === "code"
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  <Code2 className="w-3.5 h-3.5" />
                  <span>API Code</span>
                </button>
              </div>
            </div>

            {/* Subtle loading spinner if searching */}
            {loading && (
              <div className="flex items-center gap-1.5 text-xs font-mono text-zinc-400">
                <Loader2 className="w-3.5 h-3.5 animate-spin text-[#8700ff]" />
                <span className="hidden sm:inline">Searching...</span>
              </div>
            )}
          </div>

          {/* TAB 1: Ranked Results View (Scrollable, 3 full cards + 1 peeking) */}
          {activeTab === "results" && (
            <div className="flex flex-col gap-2 flex-1 min-h-0">
              {/* Loading State */}
              {loading && <ResultSkeleton />}

              {/* Error State */}
              {!loading && error && (
                <ErrorState error={error} onRetry={() => onSearch()} />
              )}

              {/* Live Search Response */}
              {!loading && !error && response && (
                <>
                  {response.results.length > 0 ? (
                    <div className="flex flex-col gap-2 flex-1 min-h-0 overflow-y-auto pr-1">
                      {response.results.map((result) => (
                        <ResultCard key={result.id} result={result} />
                      ))}
                    </div>
                  ) : (
                    <EmptyState searched={true} query={response.query} />
                  )}
                </>
              )}

              {/* Fallback if results are cleared */}
              {!loading && !error && !response && (
                <div className="py-8 px-4 rounded-xl border border-dashed border-zinc-200 dark:border-zinc-800/80 flex flex-col items-center justify-center text-center gap-2 my-auto">
                  <p className="text-xs font-mono text-zinc-500">
                    Type a query above or click a suggestion to search
                  </p>
                  <div className="flex flex-wrap items-center justify-center gap-1.5 pt-1">
                    {["reciprocal rank fusion algorithm", "golang concurrency patterns", "model context protocol stdio"].map(
                      (sug) => (
                        <button
                          key={sug}
                          type="button"
                          onClick={() => onSelectSuggestion(sug)}
                          className="text-xs font-mono px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-600 dark:text-zinc-400 hover:border-[#8700ff]/40 hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-all cursor-pointer shadow-2xs"
                        >
                          {sug}
                        </button>
                      )
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* TAB 2: Real API JSON Response View */}
          {activeTab === "json" && (
            <div className="flex flex-col flex-1 min-h-0 overflow-hidden text-left">
              <JsonViewer response={response} loading={loading} />
            </div>
          )}

          {/* TAB 3: API Code Snippets View */}
          {activeTab === "code" && (
            <div className="flex flex-col flex-1 min-h-0 overflow-hidden text-left">
              <CodeSnippets
                query={query}
                providers={providers}
                extractionMode={extractionMode}
                advancedOptions={advancedOptions}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
