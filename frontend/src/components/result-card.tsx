"use client";

import { ExternalLink, Calendar } from "lucide-react";
import { ContentViewer } from "./content-viewer";
import type { components } from "@/types/api";

type SearchResult = components["schemas"]["SearchResult"];

interface ResultCardProps {
  result: SearchResult;
}

export function ResultCard({ result }: ResultCardProps) {
  // Format display domain and path cleanly
  let displayDomain = result.url;
  try {
    const parsed = new URL(result.url);
    const path = parsed.pathname === "/" ? "" : parsed.pathname;
    displayDomain = `${parsed.hostname}${path}`;
  } catch {
    // Keep raw URL if unparseable
  }

  return (
    <article className="group relative p-4 sm:p-5 rounded-xl border border-zinc-200 dark:border-zinc-800/90 bg-white dark:bg-[#121215] hover:border-zinc-400 dark:hover:border-zinc-700 transition-all duration-150 flex flex-col gap-2.5 shadow-2xs">
      {/* Top Header: Rank + URL */}
      <div className="flex items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-xs font-semibold px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700/60 flex-shrink-0">
            #{result.rank}
          </span>
          <span className="font-mono text-xs text-zinc-500 dark:text-zinc-400 truncate max-w-md" title={result.url}>
            {displayDomain}
          </span>
        </div>

        <a
          href={result.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 transition-colors p-1"
          aria-label={`Open ${result.title} in new tab`}
          title="Open destination in new tab"
        >
          <ExternalLink className="w-3.5 h-3.5" />
        </a>
      </div>

      {/* Title */}
      <h3 className="text-base sm:text-lg font-semibold text-zinc-950 dark:text-zinc-50 leading-snug">
        <a
          href={result.url}
          target="_blank"
          rel="noopener noreferrer"
          className="hover:underline focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600 rounded"
        >
          {result.title}
        </a>
      </h3>

      {/* Snippet */}
      <p className="text-sm text-zinc-600 dark:text-zinc-300 leading-relaxed line-clamp-3">
        {result.snippet}
      </p>

      {/* Extracted Content Viewer (if available) */}
      {result.content && (
        <ContentViewer content={result.content} title={result.title} />
      )}

      {/* Bottom Metadata: Engines + RRF Score + Date */}
      <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-zinc-100 dark:border-zinc-800/60 mt-1">
        {/* Origin Engines */}
        <div className="flex flex-wrap items-center gap-1.5">
          {result.engines.map((eng) => (
            <span
              key={eng.name}
              className="inline-flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded-full border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/60 text-zinc-600 dark:text-zinc-400"
              title={`Discovered on ${eng.name} at SERP rank #${eng.position}`}
            >
              <span>{eng.name}</span>
              <span className="text-zinc-400 dark:text-zinc-500 font-semibold">#{eng.position}</span>
            </span>
          ))}
        </div>

        {/* Score & Date */}
        <div className="flex items-center gap-2 text-xs font-mono">
          {result.published_date && (
            <span
              className="hidden sm:inline-flex items-center gap-1 text-zinc-400 dark:text-zinc-500"
              title={`Publication date: ${result.published_date}`}
            >
              <Calendar className="w-3 h-3" />
              <span>{result.published_date}</span>
            </span>
          )}
          <span
            className="px-2 py-0.5 rounded-full text-[11px] font-mono bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium"
            title={`Reciprocal Rank Fusion score: ${result.score}`}
          >
            RRF {result.score.toFixed(4)}
          </span>
        </div>
      </div>
    </article>
  );
}
