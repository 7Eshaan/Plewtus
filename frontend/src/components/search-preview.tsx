"use client";

import { useEffect, useId, useRef, useState } from "react";
import {
  Search,
  ArrowUp,
  X,
  BookOpen,
  Code2,
  FileJson,
  ListOrdered,
  Loader2,
  SlidersHorizontal,
  ChevronDown,
  RotateCcw,
} from "lucide-react";
import {
  ALL_PROVIDERS,
  DEFAULT_ADVANCED_OPTIONS,
  EXAMPLE_QUERIES,
  LIMIT_CHOICES,
  PROVIDER_LABELS,
  type AdvancedOptions,
  type ExtractionMode,
  type Freshness,
  type ProviderName,
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
export type ActiveTab = "results" | "json" | "code";

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
  onSelectSuggestion: (suggestion: string) => void;
}

const EXTRACTION_OPTIONS: { value: ExtractionMode; label: string; hint: string }[] = [
  { value: "none", label: "None", hint: "Snippets only (fastest)" },
  { value: "highlights", label: "Highlights", hint: "Key sentences from each page" },
  { value: "markdown", label: "Markdown", hint: "Full cleaned page as Markdown" },
];

const FRESHNESS_OPTIONS: { value: Freshness; label: string }[] = [
  { value: "", label: "Anytime" },
  { value: "day", label: "Past 24h" },
  { value: "week", label: "Past week" },
  { value: "month", label: "Past month" },
  { value: "year", label: "Past year" },
];

const TABS: { value: ActiveTab; label: string; icon: typeof ListOrdered }[] = [
  { value: "results", label: "Results", icon: ListOrdered },
  { value: "json", label: "JSON", icon: FileJson },
  { value: "code", label: "API Code", icon: Code2 },
];

function SectionLabel({ children, id }: { children: React.ReactNode; id?: string }) {
  return (
    <span id={id} className="text-[10px] font-mono tracking-wider text-zinc-500 font-semibold uppercase">
      {children}
    </span>
  );
}

/** Pill-style toggle used for single- and multi-select option groups. */
function OptionButton({
  selected,
  onClick,
  children,
  title,
  role = "button",
}: {
  selected: boolean;
  onClick: () => void;
  children: React.ReactNode;
  title?: string;
  role?: "button" | "radio";
}) {
  return (
    <button
      type="button"
      role={role}
      aria-pressed={role === "button" ? selected : undefined}
      aria-checked={role === "radio" ? selected : undefined}
      onClick={onClick}
      title={title}
      className={`py-1.5 px-2.5 rounded-lg border text-center text-xs font-mono transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 ${
        selected
          ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 border-zinc-300 dark:border-zinc-700 font-semibold shadow-xs"
          : "bg-zinc-100/60 dark:bg-zinc-900/60 text-zinc-500 dark:text-zinc-400 border-zinc-200/80 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 hover:text-zinc-800 dark:hover:text-zinc-200"
      }`}
    >
      {children}
    </button>
  );
}

function TextField({
  label,
  value,
  onChange,
  placeholder,
  maxLength,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  maxLength?: number;
}) {
  const id = useId();
  return (
    <div className="flex flex-col gap-0.5">
      <label htmlFor={id} className="text-[10px] font-mono text-zinc-500">
        {label}
      </label>
      <input
        id={id}
        type="text"
        value={value}
        maxLength={maxLength}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="px-2 py-1 rounded bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 text-xs font-mono focus-visible:outline-none focus-visible:border-brand/60"
      />
    </div>
  );
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
  const baseId = useId();
  const panelId = `${baseId}-panel`;

  // "/" or Cmd/Ctrl+K focuses the search bar from anywhere on the page.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement | null;
      const typing = target && (["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) || target.isContentEditable);
      const shortcut = (e.key === "/" && !typing) || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k");
      if (shortcut && document.activeElement !== inputRef.current) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const setOption = <K extends keyof AdvancedOptions>(key: K, value: AdvancedOptions[K]) =>
    onAdvancedOptionsChange({ ...advancedOptions, [key]: value });

  function handleFormSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (query.trim()) onSearch();
  }

  function toggleProvider(p: ProviderName) {
    if (providers.includes(p)) {
      if (providers.length > 1) onProvidersChange(providers.filter((item) => item !== p));
    } else {
      onProvidersChange(ALL_PROVIDERS.filter((item) => item === p || providers.includes(item)));
    }
  }

  const allProviders = providers.length === ALL_PROVIDERS.length;
  const advancedActive =
    advancedOptions.limit !== DEFAULT_ADVANCED_OPTIONS.limit ||
    Boolean(advancedOptions.country || advancedOptions.language || advancedOptions.includeDomains || advancedOptions.excludeDomains);

  return (
    <div className="w-full max-w-4xl mx-auto rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#0c0c0e] shadow-lg overflow-hidden text-left">
      {/* Search bar */}
      <form
        onSubmit={handleFormSubmit}
        className="px-4 py-3 sm:py-3.5 flex items-center gap-3 bg-white dark:bg-[#111114] border-b border-zinc-200 dark:border-zinc-800/80 focus-within:border-brand/50 transition-colors"
        role="search"
      >
        <Search className="w-4 h-4 text-zinc-400 dark:text-zinc-500 shrink-0" aria-hidden="true" />
        <input
          ref={inputRef}
          type="search"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder="Search the web across DuckDuckGo, Bing & Yahoo…"
          aria-label="Search query"
          enterKeyHint="search"
          className="flex-1 min-w-0 text-sm sm:text-base font-medium text-zinc-950 dark:text-zinc-50 placeholder-zinc-400 dark:placeholder-zinc-600 bg-transparent outline-none [&::-webkit-search-cancel-button]:hidden"
        />
        <kbd className="hidden sm:inline-flex items-center px-1.5 py-0.5 rounded border border-zinc-200 dark:border-zinc-700 text-[10px] font-mono text-zinc-400">
          /
        </kbd>
        {query.length > 0 && (
          <button
            type="button"
            onClick={() => {
              onQueryChange("");
              inputRef.current?.focus();
            }}
            className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
            aria-label="Clear search"
          >
            <X className="w-4 h-4" />
          </button>
        )}
        <button
          type="submit"
          disabled={!query.trim()}
          className="w-8 h-8 rounded-lg bg-brand hover:bg-brand-hover active:scale-95 text-white shadow-xs shadow-brand/25 transition-all flex items-center justify-center cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-zinc-900"
          aria-label={loading ? "Searching (submit to restart)" : "Search"}
        >
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <ArrowUp className="w-4 h-4 stroke-[2.5]" />}
        </button>
      </form>

      <div className="flex flex-col md:flex-row md:h-[400px]">
        {/* Controls */}
        <aside
          className="w-full md:w-64 lg:w-72 border-b md:border-b-0 md:border-r border-zinc-200 dark:border-zinc-800/80 p-3.5 sm:p-4 flex flex-col justify-between gap-3 bg-zinc-50/50 dark:bg-zinc-950/20 overflow-y-auto shrink-0"
          aria-label="Search options"
        >
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5" role="radiogroup" aria-labelledby={`${baseId}-extract`}>
              <SectionLabel id={`${baseId}-extract`}>Extraction</SectionLabel>
              <div className="grid grid-cols-3 gap-1.5">
                {EXTRACTION_OPTIONS.map((opt) => (
                  <OptionButton
                    key={opt.value}
                    role="radio"
                    selected={extractionMode === opt.value}
                    onClick={() => onExtractionModeChange(opt.value)}
                    title={opt.hint}
                  >
                    {opt.label}
                  </OptionButton>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-1.5" role="group" aria-labelledby={`${baseId}-engines`}>
              <SectionLabel id={`${baseId}-engines`}>Engines</SectionLabel>
              <div className="grid grid-cols-2 gap-1.5">
                <OptionButton
                  selected={allProviders}
                  onClick={() => onProvidersChange(allProviders ? ["duckduckgo"] : ALL_PROVIDERS)}
                  title={allProviders ? "Query DuckDuckGo only" : "Query all engines"}
                >
                  All ({ALL_PROVIDERS.length})
                </OptionButton>
                {ALL_PROVIDERS.map((p) => (
                  <OptionButton key={p} selected={providers.includes(p)} onClick={() => toggleProvider(p)}>
                    {PROVIDER_LABELS[p]}
                  </OptionButton>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-1.5" role="radiogroup" aria-labelledby={`${baseId}-fresh`}>
              <SectionLabel id={`${baseId}-fresh`}>Freshness</SectionLabel>
              <div className="flex flex-wrap gap-1.5">
                {FRESHNESS_OPTIONS.map((opt) => (
                  <OptionButton
                    key={opt.value || "any"}
                    role="radio"
                    selected={advancedOptions.freshness === opt.value}
                    onClick={() => setOption("freshness", opt.value)}
                  >
                    {opt.label}
                  </OptionButton>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-1">
              <button
                type="button"
                onClick={() => setAdvancedOpen((open) => !open)}
                aria-expanded={advancedOpen}
                aria-controls={`${baseId}-advanced`}
                className="inline-flex items-center justify-between px-2.5 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 text-xs font-mono text-zinc-600 dark:text-zinc-400 hover:text-zinc-950 dark:hover:text-zinc-100 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60"
              >
                <span className="inline-flex items-center gap-1.5">
                  <SlidersHorizontal className="w-3.5 h-3.5" />
                  Advanced
                  {advancedActive && <span className="w-1.5 h-1.5 rounded-full bg-brand" aria-label="(modified)" />}
                </span>
                <ChevronDown className={`w-3.5 h-3.5 transition-transform ${advancedOpen ? "rotate-180" : ""}`} />
              </button>

              {advancedOpen && (
                <div
                  id={`${baseId}-advanced`}
                  className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 flex flex-col gap-2 text-xs"
                >
                  <div className="flex items-center justify-between pb-1 border-b border-zinc-100 dark:border-zinc-800">
                    <span className="font-mono text-[10px] text-zinc-500 uppercase">Options</span>
                    <button
                      type="button"
                      onClick={() => onAdvancedOptionsChange({ ...DEFAULT_ADVANCED_OPTIONS })}
                      className="text-[10px] font-mono text-zinc-500 hover:text-zinc-950 dark:hover:text-zinc-100 inline-flex items-center gap-1 cursor-pointer"
                    >
                      <RotateCcw className="w-2.5 h-2.5" />
                      Reset
                    </button>
                  </div>

                  <div className="flex flex-col gap-1" role="radiogroup" aria-label="Number of results">
                    <span className="text-[10px] font-mono text-zinc-500">Results</span>
                    <div className="grid grid-cols-4 gap-1">
                      {LIMIT_CHOICES.map((n) => (
                        <OptionButton key={n} role="radio" selected={advancedOptions.limit === n} onClick={() => setOption("limit", n)}>
                          {n}
                        </OptionButton>
                      ))}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <TextField
                      label="Country (ISO)"
                      value={advancedOptions.country}
                      maxLength={2}
                      placeholder="us"
                      onChange={(v) => setOption("country", v.toLowerCase().replace(/[^a-z]/g, ""))}
                    />
                    <TextField
                      label="Language (ISO)"
                      value={advancedOptions.language}
                      maxLength={2}
                      placeholder="en"
                      onChange={(v) => setOption("language", v.toLowerCase().replace(/[^a-z]/g, ""))}
                    />
                  </div>
                  <TextField
                    label="Include domains (comma-separated)"
                    value={advancedOptions.includeDomains}
                    placeholder="go.dev, pkg.go.dev"
                    onChange={(v) => setOption("includeDomains", v)}
                  />
                  <TextField
                    label="Exclude domains (comma-separated)"
                    value={advancedOptions.excludeDomains}
                    placeholder="pinterest.com"
                    onChange={(v) => setOption("excludeDomains", v)}
                  />
                </div>
              )}
            </div>
          </div>

          <a
            href="#quickstart"
            className="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-xs font-mono text-zinc-700 dark:text-zinc-300 hover:border-zinc-400 dark:hover:border-zinc-600 hover:text-zinc-950 dark:hover:text-white transition-colors select-none"
          >
            <BookOpen className="w-3 h-3 text-zinc-400" />
            Use it from your code
          </a>
        </aside>

        {/* Output */}
        <div className="flex-1 p-3.5 sm:p-4 flex flex-col gap-2.5 overflow-hidden bg-white/60 dark:bg-[#0c0c0e]/60 min-h-[320px] md:min-h-0">
          <div className="flex items-center justify-between gap-2 border-b border-zinc-200 dark:border-zinc-800/80 pb-2 shrink-0">
            <div
              className="inline-flex items-center p-0.5 rounded-lg bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 text-xs font-mono select-none"
              role="tablist"
              aria-label="Output view"
            >
              {TABS.map(({ value, label, icon: Icon }) => (
                <button
                  key={value}
                  type="button"
                  role="tab"
                  id={`${baseId}-tab-${value}`}
                  aria-selected={activeTab === value}
                  aria-controls={panelId}
                  onClick={() => onTabChange(value)}
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 ${
                    activeTab === value
                      ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 shadow-xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  <Icon className="w-3.5 h-3.5" />
                  <span>{label}</span>
                  {value === "results" && response && response.results.length > 0 && (
                    <span className="text-[10px] font-bold px-1.5 rounded-full bg-brand/10 text-brand dark:text-brand-light border border-brand/20">
                      {response.results.length}
                    </span>
                  )}
                </button>
              ))}
            </div>

            <div className="flex items-center gap-1.5 text-xs font-mono text-zinc-500" aria-live="polite">
              {loading ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin text-brand" />
                  <span className="hidden sm:inline">Searching…</span>
                </>
              ) : (
                response && (
                  <span className="hidden sm:inline">
                    {response.metadata.took_ms} ms{response.metadata.cached ? " · cached" : ""}
                  </span>
                )
              )}
            </div>
          </div>

          <div
            id={panelId}
            role="tabpanel"
            aria-labelledby={`${baseId}-tab-${activeTab}`}
            aria-busy={loading}
            className="flex flex-col flex-1 min-h-0"
          >
            {activeTab === "results" && (
              <>
                {loading && <ResultSkeleton />}
                {!loading && error && <ErrorState error={error} onRetry={() => onSearch()} />}
                {!loading && !error && response && (
                  <>
                    {response.metadata.providers.failed.length > 0 && (
                      <p className="text-[11px] font-mono text-amber-700 dark:text-amber-400/90 mb-2 truncate" title={response.metadata.providers.failed.map((f) => `${f.engine}: ${f.error}`).join("\n")}>
                        Unavailable: {response.metadata.providers.failed.map((f) => PROVIDER_LABELS[f.engine as ProviderName] ?? f.engine).join(", ")}
                      </p>
                    )}
                    {response.results.length > 0 ? (
                      <ol className="flex flex-col gap-2 flex-1 min-h-0 overflow-y-auto pr-1">
                        {response.results.map((result) => (
                          <li key={result.id}>
                            <ResultCard result={result} />
                          </li>
                        ))}
                      </ol>
                    ) : (
                      <EmptyState query={response.query} />
                    )}
                  </>
                )}
                {!loading && !error && !response && (
                  <div className="py-8 px-4 rounded-xl border border-dashed border-zinc-200 dark:border-zinc-800/80 flex flex-col items-center justify-center text-center gap-2 my-auto">
                    <p className="text-xs font-mono text-zinc-500">Type a query above or try an example</p>
                    <div className="flex flex-wrap items-center justify-center gap-1.5 pt-1">
                      {EXAMPLE_QUERIES.map((sug) => (
                        <button
                          key={sug}
                          type="button"
                          onClick={() => onSelectSuggestion(sug)}
                          className="text-xs font-mono px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-600 dark:text-zinc-400 hover:border-brand/40 hover:text-brand dark:hover:text-brand-light transition-colors cursor-pointer"
                        >
                          {sug}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </>
            )}

            {activeTab === "json" && <JsonViewer response={response} loading={loading} />}

            {activeTab === "code" && (
              <CodeSnippets
                query={query}
                providers={providers}
                extractionMode={extractionMode}
                advancedOptions={advancedOptions}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
