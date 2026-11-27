"use client";

import { useEffect, useId, useRef, useState, type CSSProperties, type ReactNode } from "react";
import {
  Search,
  ArrowUp,
  X,
  Code2,
  FileJson,
  ListOrdered,
  SlidersHorizontal,
  RotateCcw,
  Layers,
  Clock,
  FileText,
} from "lucide-react";
import {
  ALL_PROVIDERS,
  DEFAULT_ADVANCED_OPTIONS,
  DEFAULT_PROVIDERS,
  EXAMPLE_QUERIES,
  LIMIT_CHOICES,
  PROVIDER_HINTS,
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
import { handleArrowNav } from "@/lib/arrow-nav";
import { cn } from "@/lib/cn";
import { CodeSnippets } from "./code-snippets";
import { Dropdown, DropdownOption } from "./ui/dropdown";
import { Button } from "./ui/button";
import { IconButton } from "./ui/icon-button";
import { Chip, chipClasses } from "./ui/chip";
import { Tabs } from "./ui/tabs";
import { SegmentedControl } from "./ui/segmented-control";
import { TextField } from "./ui/text-field";
import { CircularProgress, LinearProgress } from "./ui/progress";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type SearchResult = components["schemas"]["SearchResult"];
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
  /** The most recent API response (the last page when more were loaded). */
  response: SearchResponse | null;
  /** Results from every page loaded so far. */
  results: SearchResult[];
  /** The query the results on screen belong to. */
  searchedQuery: string;
  /** True when the controls changed since the results were fetched. */
  stale: boolean;
  canLoadMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  error: ErrorResponse | null;
  onSelectSuggestion: (suggestion: string) => void;
}

const EXTRACTION_OPTIONS: { value: ExtractionMode; label: string; hint: string }[] = [
  { value: "none", label: "Snippets", hint: "Search result snippets only (fastest)" },
  { value: "highlights", label: "Highlights", hint: "Key sentences from each of the top 5 pages" },
  { value: "markdown", label: "Markdown", hint: "Top 5 pages cleaned up as Markdown" },
  { value: "text", label: "Text", hint: "Top 5 pages cleaned up as plain text" },
];

const FRESHNESS_OPTIONS: { value: Freshness; label: string }[] = [
  { value: "", label: "Anytime" },
  { value: "day", label: "Past 24h" },
  { value: "week", label: "Past week" },
  { value: "month", label: "Past month" },
  { value: "year", label: "Past year" },
];

/** Title row at the top of an options menu, with optional actions. */
function MenuHeader({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="mb-1 flex items-center justify-between gap-2 border-b border-outline-variant pb-1.5 pl-3 pr-1">
      <span className="text-title-sm text-on-surface-variant">{title}</span>
      {children && <div className="flex items-center gap-0.5">{children}</div>}
    </div>
  );
}

/** One tag per queried engine: how long it took, or why it contributed nothing. */
function EngineStatus({ response }: { response: SearchResponse }) {
  const { queried, failed, timing_ms: timing } = response.metadata.providers;
  const failures = new Map(failed.map((f) => [f.engine, f.error]));
  const engines = [...new Set([...queried, ...failures.keys()])];
  if (engines.length === 0) return null;
  return (
    <ul className="flex flex-wrap items-center gap-1.5" aria-label="Engine status">
      {engines.map((engine) => {
        const failure = failures.get(engine);
        const label = PROVIDER_LABELS[engine as ProviderName] ?? engine;
        const ms = timing?.[engine];
        return (
          <li
            key={engine}
            title={failure ? `${label}: ${failure}` : `${label} answered in ${ms ?? "?"} ms`}
            className={chipClasses({ interactive: false, dense: true, hasIcon: true })}
          >
            {/* Only the dot says whether the engine failed; the tag itself stays neutral. */}
            <span className={cn("size-2 rounded-full", failure ? "bg-red-500" : "bg-primary")} aria-hidden="true" />
            <span>{label}</span>
            <span className="opacity-70 tabular-nums">{failure ? "no results used" : ms !== undefined ? `${ms} ms` : ""}</span>
          </li>
        );
      })}
    </ul>
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
  results,
  searchedQuery,
  stale,
  canLoadMore,
  loadingMore,
  onLoadMore,
  error,
  onSelectSuggestion,
}: SearchPreviewProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  // Which options menu is open; opening one closes the others.
  const [openMenu, setOpenMenu] = useState<"engines" | "freshness" | "content" | "advanced" | null>(null);
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
  const defaultProviders =
    providers.length === DEFAULT_PROVIDERS.length && DEFAULT_PROVIDERS.every((p) => providers.includes(p));
  const engineSummary = allProviders
    ? `All ${ALL_PROVIDERS.length}`
    : providers.length === 1
      ? PROVIDER_LABELS[providers[0]]
      : `${PROVIDER_LABELS[providers[0]]} +${providers.length - 1}`;
  const freshnessLabel = FRESHNESS_OPTIONS.find((o) => o.value === advancedOptions.freshness)?.label ?? "Anytime";
  const extractionLabel = EXTRACTION_OPTIONS.find((o) => o.value === extractionMode)?.label ?? "Snippets";
  const filtersSet = Boolean(
    advancedOptions.country || advancedOptions.language || advancedOptions.includeDomains || advancedOptions.excludeDomains
  );
  const advancedActive = advancedOptions.limit !== DEFAULT_ADVANCED_OPTIONS.limit || filtersSet;
  const optionsLabel = `${advancedOptions.limit} results${filtersSet ? " + filters" : ""}`;

  return (
    <div className="w-full max-w-4xl mx-auto overflow-hidden rounded-corner-xl bg-surface-container-low text-left text-on-surface shadow-elevation-3">
      {/* M3 search bar */}
      <form onSubmit={handleFormSubmit} role="search" className="p-3 pb-2">
        <div className="flex h-14 items-center gap-1 rounded-corner-full bg-surface-container-high pl-4 pr-2 transition-shadow motion-effects focus-within:ring-2 focus-within:ring-primary">
          <Search className="mr-2 size-5 shrink-0 text-on-surface-variant" aria-hidden="true" />
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Search the web…"
            aria-label="Search query"
            enterKeyHint="search"
            className="min-w-0 flex-1 bg-transparent text-body-lg text-on-surface placeholder:text-on-surface-variant outline-none [&::-webkit-search-cancel-button]:hidden"
          />
          <kbd className="mr-1 hidden h-6 items-center rounded-corner-xs border border-outline-variant px-1.5 font-sans text-label-sm text-on-surface-variant sm:inline-flex">
            /
          </kbd>
          {query.length > 0 && (
            <IconButton
              label="Clear search"
              size="xs"
              onClick={() => {
                onQueryChange("");
                inputRef.current?.focus();
              }}
            >
              <X />
            </IconButton>
          )}
          <IconButton type="submit" variant="filled" label={loading ? "Searching (submit to restart)" : "Search"} disabled={!query.trim()}>
            {loading ? <CircularProgress size={20} color="current" decorative /> : <ArrowUp className="stroke-[2.5]" />}
          </IconButton>
        </div>
      </form>

      {/* Options: one chip + menu per setting, so results get the full width */}
      <div className="relative flex flex-wrap items-center gap-2 px-3 pb-3" role="toolbar" aria-label="Search options">
        <Dropdown
          id={`${baseId}-engines`}
          label="Engines"
          value={engineSummary}
          icon={<Layers />}
          open={openMenu === "engines"}
          onOpenChange={(open) => setOpenMenu(open ? "engines" : null)}
          modified={!defaultProviders}
          panelClassName="sm:w-80"
        >
          <MenuHeader title="Engines">
            <Button variant="text" size="xs" onClick={() => onProvidersChange(DEFAULT_PROVIDERS)} disabled={defaultProviders}>
              Defaults
            </Button>
            <Button variant="text" size="xs" onClick={() => onProvidersChange(ALL_PROVIDERS)} disabled={allProviders}>
              All ({ALL_PROVIDERS.length})
            </Button>
          </MenuHeader>
          <div role="group" aria-label="Engines" className="flex flex-col">
            {ALL_PROVIDERS.map((p) => {
              const selected = providers.includes(p);
              return (
                <DropdownOption
                  key={p}
                  kind="checkbox"
                  checked={selected}
                  onSelect={() => toggleProvider(p)}
                  label={PROVIDER_LABELS[p]}
                  hint={PROVIDER_HINTS[p] ?? (DEFAULT_PROVIDERS.includes(p) ? "Queried by default." : undefined)}
                  disabled={selected && providers.length === 1}
                  disabledReason="At least one engine is required"
                />
              );
            })}
          </div>
        </Dropdown>

        <Dropdown
          id={`${baseId}-freshness`}
          label="Freshness"
          value={freshnessLabel}
          icon={<Clock />}
          open={openMenu === "freshness"}
          onOpenChange={(open) => setOpenMenu(open ? "freshness" : null)}
          modified={advancedOptions.freshness !== ""}
          panelClassName="sm:w-52"
        >
          <div role="radiogroup" aria-label="Freshness" onKeyDown={handleArrowNav} className="flex flex-col">
            {FRESHNESS_OPTIONS.map((opt) => (
              <DropdownOption
                key={opt.value || "any"}
                kind="radio"
                checked={advancedOptions.freshness === opt.value}
                onSelect={() => {
                  setOption("freshness", opt.value);
                  setOpenMenu(null);
                }}
                label={opt.label}
              />
            ))}
          </div>
        </Dropdown>

        <Dropdown
          id={`${baseId}-content`}
          label="Content"
          value={extractionLabel}
          icon={<FileText />}
          open={openMenu === "content"}
          onOpenChange={(open) => setOpenMenu(open ? "content" : null)}
          modified={extractionMode !== "none"}
          panelClassName="sm:w-72"
        >
          <div role="radiogroup" aria-label="Page content" onKeyDown={handleArrowNav} className="flex flex-col">
            {EXTRACTION_OPTIONS.map((opt) => (
              <DropdownOption
                key={opt.value}
                kind="radio"
                checked={extractionMode === opt.value}
                onSelect={() => {
                  onExtractionModeChange(opt.value);
                  setOpenMenu(null);
                }}
                label={opt.label}
                hint={opt.hint}
              />
            ))}
          </div>
        </Dropdown>

        <Dropdown
          id={`${baseId}-advanced`}
          label="Options"
          value={optionsLabel}
          icon={<SlidersHorizontal />}
          open={openMenu === "advanced"}
          onOpenChange={(open) => setOpenMenu(open ? "advanced" : null)}
          modified={advancedActive}
          panelClassName="sm:w-80"
        >
          <MenuHeader title="Options">
            <Button
              variant="text"
              size="xs"
              icon={<RotateCcw />}
              onClick={() => onAdvancedOptionsChange({ ...DEFAULT_ADVANCED_OPTIONS, freshness: advancedOptions.freshness })}
              disabled={!advancedActive}
            >
              Reset
            </Button>
          </MenuHeader>
          <div className="flex flex-col gap-3 px-2 pb-2 pt-1">
            <div className="flex flex-col gap-1.5">
              <span className="text-body-sm text-on-surface-variant">Results</span>
              <SegmentedControl
                ariaLabel="Number of results"
                fullWidth
                value={String(advancedOptions.limit)}
                onChange={(v) => setOption("limit", Number(v))}
                options={LIMIT_CHOICES.map((n) => ({ value: String(n), label: String(n) }))}
              />
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
        </Dropdown>
      </div>

      {/* Output */}
      <div className="flex flex-col md:h-[460px]">
        <div className="relative flex shrink-0 items-center justify-between gap-2 border-y border-outline-variant sm:pr-4">
          <Tabs
            ariaLabel="Output view"
            idBase={baseId}
            panelId={panelId}
            value={activeTab}
            onChange={onTabChange}
            tabs={[
              {
                value: "results",
                label: "Results",
                icon: <ListOrdered />,
                badge:
                  results.length > 0 ? (
                    <span className="min-w-5 rounded-corner-full bg-primary-container px-1.5 text-center text-label-sm text-on-primary-container tabular-nums">
                      {results.length}
                    </span>
                  ) : undefined,
              },
              { value: "json", label: "JSON", icon: <FileJson /> },
              { value: "code", label: "API Code", icon: <Code2 /> },
            ]}
          />
          <div className="hidden shrink-0 text-label-md text-on-surface-variant sm:block" aria-live="polite">
            {loading ? "Searching…" : response && `${response.metadata.took_ms} ms${response.metadata.cached ? " · cached" : ""}`}
          </div>
          {loading && (
            <div className="absolute inset-x-0 -bottom-px">
              <LinearProgress label="Searching" />
            </div>
          )}
        </div>

        <div
          // Keyed by tab so each newly chosen view fades in.
          key={activeTab}
          id={panelId}
          role="tabpanel"
          aria-labelledby={`${baseId}-tab-${activeTab}`}
          aria-busy={loading}
          className="m3-view-in flex min-h-[320px] flex-1 flex-col overflow-hidden p-3 sm:p-4 md:min-h-0"
        >
          {activeTab === "results" && (
            <>
              {loading && <ResultSkeleton />}
              {!loading && error && (
                <div className={response ? "mb-3 shrink-0" : ""}>
                  <ErrorState error={error} onRetry={response ? undefined : () => onSearch()} />
                </div>
              )}
              {!loading && response && (
                <>
                  <div className="mb-3 flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
                    <EngineStatus response={response} />
                    {results.length > 0 && (
                      <span className="text-label-md text-on-surface-variant">
                        {results.length} {results.length === 1 ? "result" : "results"}
                      </span>
                    )}
                  </div>
                  {stale && (
                    <div className="mb-3 flex shrink-0 items-center justify-between gap-3 rounded-corner-md border border-primary-container/50 bg-primary-container/12 py-1.5 pl-4 pr-1.5 text-body-md text-on-surface">
                      <span>Options changed since this search.</span>
                      <Button variant="filled" size="xs" icon={<RotateCcw />} onClick={() => onSearch(searchedQuery)}>
                        Update results
                      </Button>
                    </div>
                  )}
                  {results.length > 0 ? (
                    <ol className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto pr-2.5 [scrollbar-gutter:stable]">
                      {results.map((result, index) => (
                        <li
                          key={result.id}
                          className="m3-enter"
                          // Stagger within each loaded page, capped so long pages don't drag.
                          style={{ "--m3-stagger": Math.min(index % advancedOptions.limit, 10) } as CSSProperties}
                        >
                          <ResultCard result={result} />
                        </li>
                      ))}
                      {canLoadMore && (
                        <li className="flex justify-center py-2">
                          <Button
                            variant="tonal"
                            onClick={onLoadMore}
                            disabled={loadingMore}
                            icon={loadingMore ? <CircularProgress size={18} color="current" decorative /> : undefined}
                          >
                            {loadingMore ? "Loading…" : "Load more results"}
                          </Button>
                        </li>
                      )}
                    </ol>
                  ) : (
                    !error && <EmptyState query={response.query} />
                  )}
                </>
              )}
              {!loading && !error && !response && (
                <div className="my-auto flex flex-col items-center justify-center gap-3 rounded-corner-lg border border-dashed border-outline-variant px-4 py-8 text-center">
                  <p className="text-body-md text-on-surface-variant">Type a query above or try an example</p>
                  <div className="flex flex-wrap items-center justify-center gap-2">
                    {EXAMPLE_QUERIES.map((suggestion) => (
                      <Chip key={suggestion} icon={<Search className="text-primary" />} onClick={() => onSelectSuggestion(suggestion)}>
                        {suggestion}
                      </Chip>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}

          {activeTab === "json" && <JsonViewer response={response} error={error} loading={loading} />}

          {activeTab === "code" && (
            <CodeSnippets query={query} providers={providers} extractionMode={extractionMode} advancedOptions={advancedOptions} />
          )}
        </div>
      </div>
    </div>
  );
}
