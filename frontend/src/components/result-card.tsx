"use client";

import { useState } from "react";
import { ExternalLink, Calendar, Globe, Zap, BarChart2, FileText } from "lucide-react";
import { ContentViewer } from "./content-viewer";
import type { components } from "@/types/api";

type SearchResult = components["schemas"]["SearchResult"];

interface ResultCardProps {
  result: SearchResult;
}

function FallbackIcon({ domain, title }: { domain: string; title: string }) {
  const t = (domain + " " + title).toLowerCase();
  if (t.includes("benchmark") || t.includes("rank") || t.includes("chart") || t.includes("stats") || t.includes("trend")) {
    return <BarChart2 className="w-3 h-3 text-blue-400" />;
  }
  if (t.includes("ai") || t.includes("model") || t.includes("spec") || t.includes("weight") || t.includes("llm")) {
    return <Zap className="w-3 h-3 text-amber-400" />;
  }
  if (t.includes("paper") || t.includes("doc") || t.includes("news") || t.includes("weekly") || t.includes("article")) {
    return <FileText className="w-3 h-3 text-emerald-400" />;
  }
  return <Globe className="w-3 h-3 text-sky-400" />;
}

function Favicon({ domain, title }: { domain: string; title: string }) {
  const [hasError, setHasError] = useState(false);

  if (hasError || !domain) {
    return <FallbackIcon domain={domain} title={title} />;
  }

  const faviconUrl = `https://www.google.com/s2/favicons?domain=${encodeURIComponent(domain)}&sz=32`;

  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={faviconUrl}
      alt=""
      width={14}
      height={14}
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setHasError(true)}
      className="w-3.5 h-3.5 object-contain rounded-xs"
    />
  );
}

export function ResultCard({ result }: ResultCardProps) {
  // Format display domain cleanly
  let displayDomain = result.url;
  try {
    const parsed = new URL(result.url);
    displayDomain = parsed.hostname.replace(/^www\./, "");
  } catch {
    // Keep raw URL if unparseable
  }

  return (
    <article className="group relative p-3 sm:p-3.5 rounded-xl border border-zinc-200/90 dark:border-zinc-800/90 bg-white dark:bg-[#121215] hover:border-brand/30 dark:hover:border-brand/30 transition-all flex flex-col gap-1.5 shadow-2xs">
      {/* Top Header: Icon + Title + Hostname */}
      <div className="flex items-center justify-between gap-2.5 text-xs">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <div className="w-5 h-5 rounded-md bg-zinc-100 dark:bg-zinc-800 border border-zinc-200/80 dark:border-zinc-700/60 flex items-center justify-center flex-shrink-0 shadow-2xs overflow-hidden">
            <Favicon key={displayDomain} domain={displayDomain} title={result.title} />
          </div>
          <span className="font-mono text-[10px] text-zinc-400 dark:text-zinc-500 tabular-nums shrink-0" aria-label={`Rank ${result.rank}`}>
            {result.rank}.
          </span>
          <h3 className="text-xs sm:text-sm font-semibold text-zinc-950 dark:text-zinc-100 truncate leading-snug" title={result.title}>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline hover:text-brand dark:hover:text-brand-light transition-colors focus-visible:outline-none focus-visible:underline"
            >
              {result.title}
            </a>
          </h3>
        </div>

        <div className="flex items-center gap-1.5 flex-shrink-0">
          <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500 truncate max-w-[140px]" title={result.url}>
            {displayDomain}
          </span>
          <a
            href={result.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 transition-opacity opacity-0 group-hover:opacity-100 focus-visible:opacity-100 p-0.5"
            aria-label={`Open ${result.title} in new tab`}
            title="Open in new tab"
          >
            <ExternalLink className="w-3 h-3" />
          </a>
        </div>
      </div>

      {/* Snippet */}
      <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed line-clamp-2 pl-7">
        {result.snippet || <span className="italic text-zinc-400">No snippet available</span>}
      </p>

      {/* Extracted Content Viewer (if available) */}
      {result.content && (
        <div className="pl-7 pt-0.5">
          <ContentViewer content={result.content} title={result.title} />
        </div>
      )}

      {/* Bottom Metadata: Engines + RRF Score + Date */}
      <div className="flex flex-wrap items-center justify-between gap-1.5 pt-1.5 border-t border-zinc-100 dark:border-zinc-800/40 text-[10px] font-mono pl-7">
        {/* Origin Engines */}
        <div className="flex flex-wrap items-center gap-1">
          {result.engines.length > 1 && (
            <span className="text-zinc-400 dark:text-zinc-500 mr-0.5">{result.engines.length} engines:</span>
          )}
          {result.engines.map((eng) => (
            <span
              key={eng.name}
              className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/60 dark:border-zinc-700/50"
              title={`Discovered on ${eng.name} at rank #${eng.position}`}
            >
              <span>{eng.name}</span>
              <span className="text-zinc-400 dark:text-zinc-500 font-semibold">#{eng.position}</span>
            </span>
          ))}
        </div>

        {/* Score & Date */}
        <div className="flex items-center gap-2">
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
            className="px-1.5 py-0.5 rounded text-[10px] bg-brand/10 text-brand dark:text-brand-light border border-brand/20 font-medium tabular-nums"
            title={`Ranking score ${result.ranking_score.toFixed(3)} = 0.8 × relevance ${result.relevance_score.toFixed(3)} + 0.2 × consensus (RRF ${result.score.toFixed(4)})`}
          >
            score {result.ranking_score.toFixed(2)}
          </span>
        </div>
      </div>
    </article>
  );
}
