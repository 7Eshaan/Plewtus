"use client";

import { Search, Sparkles } from "lucide-react";

interface EmptyStateProps {
  searched: boolean;
  query?: string;
  onSelectSuggestion?: (suggestion: string) => void;
}

const SUGGESTIONS = [
  "golang concurrency patterns",
  "reciprocal rank fusion algorithm",
  "model context protocol stdio",
  "distributed consensus raft",
];

export function EmptyState({ searched, query, onSelectSuggestion }: EmptyStateProps) {
  if (searched) {
    return (
      <div className="py-16 px-4 rounded-xl border border-dashed border-zinc-300 dark:border-zinc-800 flex flex-col items-center justify-center text-center gap-3">
        <div className="w-10 h-10 rounded-full bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex items-center justify-center text-zinc-500">
          <Search className="w-5 h-5" />
        </div>
        <div className="flex flex-col gap-1 max-w-sm">
          <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            No matching results
          </h3>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            No organic results found across DuckDuckGo, Bing, or Yahoo for &ldquo;{query}&rdquo;.
            Try adjusting your search terms.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="py-12 px-4 rounded-xl border border-zinc-200/80 dark:border-zinc-800/80 bg-zinc-50/50 dark:bg-[#121215]/40 flex flex-col items-center justify-center text-center gap-4">
      <div className="w-9 h-9 rounded-lg bg-[#8700ff]/10 dark:bg-[#8700ff]/20 border border-[#8700ff]/30 flex items-center justify-center text-[#8700ff] dark:text-[#a84dff]">
        <Sparkles className="w-4 h-4" />
      </div>
      <div className="flex flex-col gap-1 max-w-md">
        <h3 className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
          Federated AI Search Engine
        </h3>
        <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
          Dispatches concurrent searches across DuckDuckGo, Bing, and Yahoo, normalizes canonical URLs,
          and ranks results using Reciprocal Rank Fusion.
        </p>
      </div>

      {onSelectSuggestion && (
        <div className="flex flex-wrap items-center justify-center gap-2 pt-2 max-w-lg">
          <span className="text-[11px] font-mono text-zinc-400 dark:text-zinc-500 mr-1">
            Try:
          </span>
          {SUGGESTIONS.map((sug) => (
            <button
              key={sug}
              type="button"
              onClick={() => onSelectSuggestion(sug)}
              className="text-xs font-mono px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:border-[#8700ff]/40 hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-all cursor-pointer shadow-2xs"
            >
              {sug}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
