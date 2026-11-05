"use client";

import { useRef, useEffect, useState } from "react";
import { Search, X, Loader2, ArrowRight } from "lucide-react";
import { ProviderChips, type ProviderName } from "./provider-chips";
import { ExtractionToggle, type ExtractionMode } from "./extraction-toggle";
import { AdvancedDropdown, type AdvancedOptions } from "./advanced-dropdown";

interface SearchBarProps {
  query: string;
  onChange: (value: string) => void;
  onSubmit: (e?: React.FormEvent) => void;
  loading: boolean;
  providers: ProviderName[];
  onProvidersChange: (providers: ProviderName[]) => void;
  extractionMode: ExtractionMode;
  onExtractionModeChange: (mode: ExtractionMode) => void;
  advancedOptions: AdvancedOptions;
  onAdvancedOptionsChange: (options: AdvancedOptions) => void;
  autoFocus?: boolean;
}

export function SearchBar({
  query,
  onChange,
  onSubmit,
  loading,
  providers,
  onProvidersChange,
  extractionMode,
  onExtractionModeChange,
  advancedOptions,
  onAdvancedOptionsChange,
  autoFocus = true,
}: SearchBarProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);

  // Keyboard shortcut listener: pressing "/" or "cmd+k" focuses the search bar
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (
        (e.key === "/" || ((e.metaKey || e.ctrlKey) && e.key === "k")) &&
        document.activeElement !== inputRef.current &&
        !["INPUT", "TEXTAREA"].includes((document.activeElement as HTMLElement)?.tagName)
      ) {
        e.preventDefault();
        inputRef.current?.focus();
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  function handleClear() {
    onChange("");
    inputRef.current?.focus();
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim() || loading) return;
    onSubmit(e);
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="w-full relative flex flex-col gap-2.5"
      role="search"
      aria-label="Metasearch"
    >
      {/* Search Input Box */}
      <div className="relative group flex items-center w-full rounded-xl border border-zinc-300 dark:border-zinc-800 bg-white dark:bg-[#121215] shadow-xs hover:border-zinc-400 dark:hover:border-zinc-700 focus-within:border-[#8700ff] focus-within:ring-2 focus-within:ring-[#8700ff]/20 transition-all duration-150">
        {/* Left search icon */}
        <div className="pl-4 pr-2 text-zinc-400 dark:text-zinc-500 flex items-center justify-center pointer-events-none">
          <Search className="w-4 h-4" />
        </div>

        {/* Input */}
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Search topics, queries, or docs across DuckDuckGo, Bing & Yahoo..."
          autoFocus={autoFocus}
          disabled={loading}
          className="flex-1 py-3.5 pr-2 bg-transparent text-sm sm:text-base text-zinc-950 dark:text-zinc-50 placeholder-zinc-400 dark:placeholder-zinc-500 focus:outline-none font-sans"
        />

        {/* Right action cluster */}
        <div className="pr-2 flex items-center gap-1.5">
          {/* Clear button */}
          {query.length > 0 && !loading && (
            <button
              type="button"
              onClick={handleClear}
              className="p-1 rounded-md text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
              aria-label="Clear search input"
              title="Clear input"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}

          {/* Keyboard shortcut hint */}
          {query.length === 0 && (
            <kbd className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400 dark:text-zinc-500 border border-zinc-200 dark:border-zinc-800 rounded bg-zinc-50 dark:bg-zinc-900 select-none">
              /
            </kbd>
          )}

          {/* Submit button */}
          <button
            type="submit"
            disabled={loading || !query.trim()}
            className="px-3.5 py-2 rounded-lg bg-[#8700ff] hover:bg-[#9924ff] text-white text-xs sm:text-sm font-medium active:scale-98 transition-all disabled:opacity-30 disabled:cursor-not-allowed flex items-center gap-1.5 cursor-pointer shadow-xs shadow-[#8700ff]/25"
            aria-label="Submit search"
          >
            {loading ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                <span className="hidden sm:inline">Searching...</span>
              </>
            ) : (
              <>
                <span>Search</span>
                <ArrowRight className="w-3 h-3 hidden sm:inline" />
              </>
            )}
          </button>
        </div>
      </div>

      {/* Feature Controls Strip (Directly attached below input) */}
      <div className="flex flex-wrap items-center justify-between gap-2.5 px-1 pt-0.5">
        {/* Provider Chips */}
        <ProviderChips
          selected={providers}
          onChange={onProvidersChange}
          disabled={loading}
        />

        <div className="flex items-center gap-2">
          {/* Extraction Mode Segmented Toggle */}
          <ExtractionToggle
            value={extractionMode}
            onChange={onExtractionModeChange}
            disabled={loading}
          />

          {/* Advanced Dropdown Toggle */}
          <AdvancedDropdown
            options={advancedOptions}
            onChange={onAdvancedOptionsChange}
            open={advancedOpen}
            onToggle={() => setAdvancedOpen(!advancedOpen)}
            extractionMode={extractionMode}
            disabled={loading}
          />
        </div>
      </div>
    </form>
  );
}
