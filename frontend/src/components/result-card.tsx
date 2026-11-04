"use client";

import { ExternalLink, Calendar, Globe, Zap, BarChart2, FileText } from "lucide-react";
import { ContentViewer } from "./content-viewer";
import type { components } from "@/types/api";

type SearchResult = components["schemas"]["SearchResult"];

interface ResultCardProps {
  result: SearchResult;
}

function getResultIcon(domain: string, title: string) {
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
    <article className="group relative p-2 sm:p-2.5 rounded-lg border border-zinc-200/90 dark:border-zinc-800/90 bg-white dark:bg-[#121215] hover:border-zinc-300 dark:hover:border-zinc-700 transition-all flex flex-col gap-1 shadow-2xs">
      {/* Top Header: Icon + Title + Hostname */}
      <div className="flex items-center justify-between gap-2 text-xs">
        <div className="flex items-center gap-1.5 min-w-0 flex-1">
          <div className="w-4 h-4 rounded bg-zinc-900 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/60 flex items-center justify-center flex-shrink-0 shadow-2xs">
            {getResultIcon(displayDomain, result.title)}
          </div>
          <h3 className="text-xs font-semibold text-zinc-950 dark:text-zinc-100 truncate leading-snug">
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline hover:text-blue-600 dark:hover:text-blue-400 transition-colors focus:outline-none"
            >
              {result.title}
            </a>
          </h3>
        </div>

        <div className="flex items-center gap-1 flex-shrink-0">
          <span className="font-mono text-[10px] text-zinc-400 dark:text-zinc-500 truncate max-w-[120px]" title={result.url}>
            {displayDomain}
          </span>
          <a
            href={result.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 transition-colors opacity-0 group-hover:opacity-100 p-0.5"
            aria-label={`Open ${result.title} in new tab`}
            title="Open in new tab"
          >
            <ExternalLink className="w-2.5 h-2.5" />
          </a>
        </div>
      </div>

      {/* Snippet */}
      <p className="text-[11px] text-zinc-600 dark:text-zinc-400 leading-snug line-clamp-2 pl-5.5">
        {result.snippet}
      </p>

      {/* Extracted Content Viewer (if available) */}
      {result.content && (
        <div className="pl-5.5 pt-0.5">
          <ContentViewer content={result.content} title={result.title} />
        </div>
      )}

      {/* Bottom Metadata: Engines + RRF Score + Date */}
      <div className="flex flex-wrap items-center justify-between gap-1 pt-1 border-t border-zinc-100 dark:border-zinc-800/40 mt-0.5 text-[9px] font-mono pl-5.5">
        {/* Origin Engines */}
        <div className="flex flex-wrap items-center gap-1">
          {result.engines.map((eng) => (
            <span
              key={eng.name}
              className="inline-flex items-center gap-0.5 px-1 py-0.2 rounded bg-zinc-100 dark:bg-zinc-800/80 text-zinc-600 dark:text-zinc-400 border border-zinc-200/60 dark:border-zinc-700/50"
              title={`Discovered on ${eng.name} at rank #${eng.position}`}
            >
              <span>{eng.name}</span>
              <span className="text-zinc-400 dark:text-zinc-500 font-semibold">#{eng.position}</span>
            </span>
          ))}
        </div>

        {/* Score & Date */}
        <div className="flex items-center gap-1.5">
          {result.published_date && (
            <span
              className="hidden sm:inline-flex items-center gap-0.5 text-zinc-400 dark:text-zinc-500"
              title={`Publication date: ${result.published_date}`}
            >
              <Calendar className="w-2.5 h-2.5" />
              <span>{result.published_date}</span>
            </span>
          )}
          <span
            className="px-1 py-0.2 rounded text-[9px] bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium"
            title={`Reciprocal Rank Fusion score: ${result.score}`}
          >
            RRF {result.score.toFixed(4)}
          </span>
        </div>
      </div>
    </article>
  );
}
