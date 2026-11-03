"use client";

import { SlidersHorizontal, ChevronDown, RotateCcw } from "lucide-react";
import type { ExtractionMode } from "./extraction-toggle";

export interface AdvancedOptions {
  limit: number;
  freshness: "" | "day" | "week" | "month" | "year";
  country: string;
  language: string;
  includeDomains: string;
  excludeDomains: string;
  maxPages: number;
  maxCharsPerPage: number;
}

export const DEFAULT_ADVANCED_OPTIONS: AdvancedOptions = {
  limit: 10,
  freshness: "",
  country: "",
  language: "",
  includeDomains: "",
  excludeDomains: "",
  maxPages: 3,
  maxCharsPerPage: 4000,
};

interface AdvancedDropdownProps {
  options: AdvancedOptions;
  onChange: (options: AdvancedOptions) => void;
  open: boolean;
  onToggle: () => void;
  extractionMode: ExtractionMode;
  disabled?: boolean;
}

export function AdvancedDropdown({
  options,
  onChange,
  open,
  onToggle,
  extractionMode,
  disabled,
}: AdvancedDropdownProps) {
  // Count active non-default filters
  let activeCount = 0;
  if (options.limit !== 10) activeCount++;
  if (options.freshness !== "") activeCount++;
  if (options.country.trim() !== "") activeCount++;
  if (options.language.trim() !== "") activeCount++;
  if (options.includeDomains.trim() !== "") activeCount++;
  if (options.excludeDomains.trim() !== "") activeCount++;
  if (extractionMode !== "none") {
    if (options.maxPages !== 3) activeCount++;
    if (options.maxCharsPerPage !== 4000) activeCount++;
  }

  function handleReset() {
    onChange({ ...DEFAULT_ADVANCED_OPTIONS });
  }

  return (
    <div className="w-full">
      {/* Toggle Button */}
      <button
        type="button"
        onClick={onToggle}
        disabled={disabled}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs font-mono transition-colors select-none cursor-pointer ${
          open || activeCount > 0
            ? "border-zinc-400 dark:border-zinc-600 bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-medium"
            : "border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400 hover:border-zinc-300 dark:hover:border-zinc-700"
        } ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
        aria-expanded={open}
      >
        <SlidersHorizontal className="w-3.5 h-3.5" />
        <span>Advanced</span>
        {activeCount > 0 && (
          <span className="w-4 h-4 rounded-full bg-zinc-900 text-white dark:bg-white dark:text-zinc-950 text-[10px] font-bold flex items-center justify-center">
            {activeCount}
          </span>
        )}
        <ChevronDown
          className={`w-3 h-3 transition-transform duration-150 ${open ? "rotate-180" : ""}`}
        />
      </button>

      {/* Collapsible Panel */}
      {open && (
        <div className="mt-2.5 p-4 sm:p-5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] shadow-sm flex flex-col gap-4 text-xs font-sans animate-in fade-in slide-in-from-top-1 duration-150">
          <div className="flex items-center justify-between border-b border-zinc-100 dark:border-zinc-800/80 pb-2.5">
            <span className="font-mono text-xs font-semibold text-zinc-900 dark:text-zinc-100 uppercase tracking-wide">
              Advanced Search Parameters
            </span>
            {activeCount > 0 && (
              <button
                type="button"
                onClick={handleReset}
                className="inline-flex items-center gap-1 text-[11px] font-mono text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors cursor-pointer"
              >
                <RotateCcw className="w-3 h-3" />
                <span>Reset filters</span>
              </button>
            )}
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3.5">
            {/* Limit */}
            <div className="flex flex-col gap-1">
              <label htmlFor="filter-limit" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Results Limit (1–50)
              </label>
              <input
                id="filter-limit"
                type="number"
                min={1}
                max={50}
                value={options.limit}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  onChange({ ...options, limit: isNaN(val) ? 10 : Math.min(50, Math.max(1, val)) });
                }}
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
              />
            </div>

            {/* Freshness */}
            <div className="flex flex-col gap-1">
              <label htmlFor="filter-freshness" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Freshness / Recency
              </label>
              <select
                id="filter-freshness"
                value={options.freshness}
                onChange={(e) =>
                  onChange({
                    ...options,
                    freshness: e.target.value as AdvancedOptions["freshness"],
                  })
                }
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
              >
                <option value="">Any time (Default)</option>
                <option value="day">Past 24 hours (day)</option>
                <option value="week">Past week (week)</option>
                <option value="month">Past month (month)</option>
                <option value="year">Past year (year)</option>
              </select>
            </div>

            {/* Country */}
            <div className="flex flex-col gap-1">
              <label htmlFor="filter-country" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Country (ISO 2-letter)
              </label>
              <input
                id="filter-country"
                type="text"
                maxLength={2}
                placeholder="e.g. us, gb, de"
                value={options.country}
                onChange={(e) =>
                  onChange({
                    ...options,
                    country: e.target.value.toLowerCase().replace(/[^a-z]/g, ""),
                  })
                }
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600 uppercase placeholder:normal-case"
              />
            </div>

            {/* Language */}
            <div className="flex flex-col gap-1">
              <label htmlFor="filter-language" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Language (ISO 2-letter)
              </label>
              <input
                id="filter-language"
                type="text"
                maxLength={2}
                placeholder="e.g. en, de, fr"
                value={options.language}
                onChange={(e) =>
                  onChange({
                    ...options,
                    language: e.target.value.toLowerCase().replace(/[^a-z]/g, ""),
                  })
                }
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600 uppercase placeholder:normal-case"
              />
            </div>

            {/* Include Domains */}
            <div className="flex flex-col gap-1 sm:col-span-2">
              <label htmlFor="filter-include-domains" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Include Domains (comma-separated)
              </label>
              <input
                id="filter-include-domains"
                type="text"
                placeholder="e.g. go.dev, github.com"
                value={options.includeDomains}
                onChange={(e) => onChange({ ...options, includeDomains: e.target.value })}
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
              />
            </div>

            {/* Exclude Domains */}
            <div className="flex flex-col gap-1 sm:col-span-2">
              <label htmlFor="filter-exclude-domains" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                Exclude Domains (comma-separated)
              </label>
              <input
                id="filter-exclude-domains"
                type="text"
                placeholder="e.g. pinterest.com, medium.com"
                value={options.excludeDomains}
                onChange={(e) => onChange({ ...options, excludeDomains: e.target.value })}
                className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
              />
            </div>
          </div>

          {/* Conditional Extraction Parameters */}
          {extractionMode !== "none" && (
            <div className="pt-3 border-t border-zinc-100 dark:border-zinc-800/80 flex flex-col gap-2">
              <span className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                Extraction Tunables ({extractionMode})
              </span>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
                <div className="flex flex-col gap-1">
                  <label htmlFor="extract-max-pages" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                    Max Pages to Extract (1–10)
                  </label>
                  <input
                    id="extract-max-pages"
                    type="number"
                    min={1}
                    max={10}
                    value={options.maxPages}
                    onChange={(e) => {
                      const val = parseInt(e.target.value, 10);
                      onChange({ ...options, maxPages: isNaN(val) ? 3 : Math.min(10, Math.max(1, val)) });
                    }}
                    className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <label htmlFor="extract-max-chars" className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400">
                    Max Chars Per Page (500–20000)
                  </label>
                  <input
                    id="extract-max-chars"
                    type="number"
                    min={500}
                    max={20000}
                    step={500}
                    value={options.maxCharsPerPage}
                    onChange={(e) => {
                      const val = parseInt(e.target.value, 10);
                      onChange({ ...options, maxCharsPerPage: isNaN(val) ? 4000 : Math.min(20000, Math.max(500, val)) });
                    }}
                    className="px-2.5 py-1.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 font-mono text-xs focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600"
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
