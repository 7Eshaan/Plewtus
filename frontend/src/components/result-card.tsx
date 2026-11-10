"use client";

import { useState } from "react";
import { ExternalLink, Calendar, Globe, AlertTriangle } from "lucide-react";
import { ContentViewer } from "./content-viewer";
import type { components } from "@/types/api";

type SearchResult = components["schemas"]["SearchResult"];

interface ResultCardProps {
  result: SearchResult;
}

/** Site icon, served through our own /api/favicon proxy (see that route for why). */
function Favicon({ domain }: { domain: string }) {
  const [failed, setFailed] = useState(false);

  if (failed || !domain) {
    return <Globe className="w-3 h-3 text-zinc-400" aria-hidden="true" />;
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element -- tiny icons from our own route; next/image adds nothing here
    <img
      src={`/api/favicon?domain=${encodeURIComponent(domain)}`}
      alt=""
      width={14}
      height={14}
      loading="lazy"
      onError={() => setFailed(true)}
      className="w-3.5 h-3.5 object-contain rounded-xs"
    />
  );
}

export function ResultCard({ result }: ResultCardProps) {
  let displayDomain = result.url;
  try {
    displayDomain = new URL(result.url).hostname.replace(/^www\./, "");
  } catch {
    // Keep the raw URL if it cannot be parsed.
  }

  return (
    <article className="group relative p-3 sm:p-3.5 rounded-xl border border-zinc-200/90 dark:border-zinc-800/90 bg-white dark:bg-[#121215] hover:border-brand/30 dark:hover:border-brand/30 transition-colors flex flex-col gap-1.5 shadow-2xs">
      {/* Header: icon, rank, title; the domain drops below the title on narrow screens. */}
      <div className="flex items-start gap-2 text-xs">
        <div className="w-5 h-5 rounded-md bg-zinc-100 dark:bg-zinc-800 border border-zinc-200/80 dark:border-zinc-700/60 flex items-center justify-center shrink-0 shadow-2xs overflow-hidden">
          <Favicon key={displayDomain} domain={displayDomain} />
        </div>
        <span className="font-mono text-[10px] leading-5 text-zinc-400 dark:text-zinc-500 tabular-nums shrink-0" aria-label={`Rank ${result.rank}`}>
          {result.rank}.
        </span>
        <div className="min-w-0 flex-1 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-x-2.5">
          <h3 className="text-xs sm:text-sm font-semibold text-zinc-950 dark:text-zinc-100 leading-5 line-clamp-2 sm:line-clamp-1 min-w-0" title={result.title}>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline hover:text-brand dark:hover:text-brand-light transition-colors focus-visible:outline-none focus-visible:underline"
            >
              {result.title}
            </a>
          </h3>
          <span className="inline-flex items-center gap-1 font-mono text-[11px] sm:text-xs text-zinc-400 dark:text-zinc-500 min-w-0 sm:shrink-0" title={result.url}>
            <span className="truncate sm:max-w-[150px]">{displayDomain}</span>
            <ExternalLink className="w-3 h-3 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
          </span>
        </div>
      </div>

      <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed line-clamp-2 pl-7">
        {result.snippet || <span className="italic text-zinc-400">No snippet available</span>}
      </p>

      {result.content && (
        <div className="pl-7 pt-0.5">
          <ContentViewer content={result.content} title={result.title} />
        </div>
      )}
      {!result.content && result.content_error && (
        <p className="pl-7 inline-flex items-start gap-1.5 text-[11px] font-mono text-amber-700 dark:text-amber-400/90">
          <AlertTriangle className="w-3 h-3 mt-0.5 shrink-0" aria-hidden="true" />
          <span>Page content unavailable: {result.content_error}</span>
        </p>
      )}

      {/* Footer: engines that found the result, date, score. */}
      <div className="flex flex-wrap items-center justify-between gap-1.5 pt-1.5 border-t border-zinc-100 dark:border-zinc-800/40 text-[10px] font-mono pl-7">
        <div className="flex flex-wrap items-center gap-1">
          {result.engines.map((eng) => (
            <span
              key={eng.name}
              className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/60 dark:border-zinc-700/50"
              title={`Found on ${eng.name} at position ${eng.position}`}
            >
              <span>{eng.name}</span>
              <span className="text-zinc-400 dark:text-zinc-500 font-semibold">#{eng.position}</span>
            </span>
          ))}
        </div>

        <div className="flex items-center gap-2">
          {result.published_date && (
            <span className="hidden sm:inline-flex items-center gap-1 text-zinc-400 dark:text-zinc-500" title={`Publication date: ${result.published_date}`}>
              <Calendar className="w-3 h-3" />
              <span>{result.published_date}</span>
            </span>
          )}
          <span
            className="px-1.5 py-0.5 rounded text-[10px] bg-brand/10 text-brand dark:text-brand-light border border-brand/20 font-medium tabular-nums"
            title={`Ranking score ${result.ranking_score.toFixed(3)}: engine agreement (RRF ${result.score.toFixed(4)}) with a small boost for query relevance (${result.relevance_score.toFixed(2)})`}
          >
            score {result.ranking_score.toFixed(2)}
          </span>
        </div>
      </div>
    </article>
  );
}
