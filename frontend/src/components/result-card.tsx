"use client";

import { useState } from "react";
import { ExternalLink, Calendar, Globe, AlertTriangle } from "lucide-react";
import { ContentViewer } from "./content-viewer";
import { chipClasses } from "./ui/chip";
import { PROVIDER_LABELS, type ProviderName } from "@/types/search";
import type { components } from "@/types/api";

type SearchResult = components["schemas"]["SearchResult"];

interface ResultCardProps {
  result: SearchResult;
}

/** Site icon, served through our own /api/favicon proxy (see that route for why). */
function Favicon({ domain }: { domain: string }) {
  const [failed, setFailed] = useState(false);

  if (failed || !domain) {
    return <Globe className="size-4 text-on-surface-variant" aria-hidden="true" />;
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element -- tiny icons from our own route; next/image adds nothing here
    <img
      src={`/api/favicon?domain=${encodeURIComponent(domain)}`}
      alt=""
      width={18}
      height={18}
      loading="lazy"
      onError={() => setFailed(true)}
      className="size-[18px] object-contain"
    />
  );
}

/** One search result as an M3 card: site, title, snippet, page content, and where it was found. */
export function ResultCard({ result }: ResultCardProps) {
  let displayDomain = result.url;
  try {
    displayDomain = new URL(result.url).hostname.replace(/^www\./, "");
  } catch {
    // Keep the raw URL if it cannot be parsed.
  }

  return (
    <article className="group flex flex-col gap-2 rounded-corner-md bg-surface-container p-3 transition-colors motion-effects hover:bg-surface-container-high sm:p-4">
      {/* Site and title */}
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-surface-container-highest">
          <Favicon key={displayDomain} domain={displayDomain} />
        </div>
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex min-w-0 items-center gap-1.5 text-body-sm text-on-surface-variant">
            <span className="tabular-nums" aria-label={`Rank ${result.rank}`}>
              {result.rank}.
            </span>
            <span className="truncate" title={result.url}>
              {displayDomain}
            </span>
            <ExternalLink className="size-3 shrink-0 opacity-0 transition-opacity motion-effects group-hover:opacity-100" aria-hidden="true" />
          </div>
          <h3 className="line-clamp-2 text-title-md text-on-surface sm:line-clamp-1" title={result.title}>
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              className="focus-ring rounded-corner-xs transition-colors motion-effects hover:text-primary hover:underline"
            >
              {result.title}
            </a>
          </h3>
        </div>
      </div>

      <p className="line-clamp-2 text-body-md text-on-surface-variant sm:pl-11">
        {result.snippet || <span className="italic opacity-70">No snippet available</span>}
      </p>

      {result.content && (
        <div className="sm:pl-11">
          <ContentViewer content={result.content} title={result.title} />
        </div>
      )}
      {!result.content && result.content_error && (
        <p className="inline-flex items-start gap-1.5 text-body-sm text-error sm:pl-11">
          <AlertTriangle className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
          <span>Page content unavailable: {result.content_error}</span>
        </p>
      )}

      {/* Engines that found the result, date, score */}
      <div className="flex flex-wrap items-center justify-between gap-2 pt-1 sm:pl-11">
        <ul className="flex flex-wrap items-center gap-1" aria-label="Found by">
          {result.engines.map((eng) => (
            <li
              key={eng.name}
              className={chipClasses({ interactive: false, dense: true })}
              title={`Found on ${eng.name} at position ${eng.position}`}
            >
              <span>{PROVIDER_LABELS[eng.name as ProviderName] ?? eng.name}</span>
              <span className="opacity-70 tabular-nums">#{eng.position}</span>
            </li>
          ))}
        </ul>

        <div className="flex items-center gap-3 text-body-sm text-on-surface-variant">
          {result.published_date && (
            <span className="hidden items-center gap-1 sm:inline-flex" title={`Publication date: ${result.published_date}`}>
              <Calendar className="size-3.5" aria-hidden="true" />
              <span>{result.published_date}</span>
            </span>
          )}
          <span
            className="rounded-corner-full bg-primary-container px-2 py-0.5 text-label-md text-on-primary-container tabular-nums"
            title={`Ranking score ${result.ranking_score.toFixed(3)}: engine agreement (RRF ${result.score.toFixed(4)}) with a small boost for query relevance (${result.relevance_score.toFixed(2)})`}
          >
            score {result.ranking_score.toFixed(2)}
          </span>
        </div>
      </div>
    </article>
  );
}
