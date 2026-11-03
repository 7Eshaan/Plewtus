"use client";

import { Clock, Cpu, Layers, Zap, AlertTriangle, CheckCircle2 } from "lucide-react";
import type { components } from "@/types/api";

type SearchMetadata = components["schemas"]["SearchMetadata"];

interface MetadataStripProps {
  metadata: SearchMetadata;
}

export function MetadataStrip({ metadata }: MetadataStripProps) {
  const queriedCount = metadata.providers.queried?.length ?? 0;
  const successfulCount = metadata.providers.successful?.length ?? 0;
  const failedList = metadata.providers.failed ?? [];
  const failedCount = failedList.length;

  const enginesSummary =
    failedCount === 0
      ? `All ${queriedCount} engines responded`
      : `Results from ${successfulCount} of ${queriedCount} engines`;

  return (
    <div className="w-full flex flex-wrap items-center justify-between gap-3 px-3.5 py-2.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50/80 dark:bg-zinc-900/50 text-xs font-mono text-zinc-600 dark:text-zinc-400">
      {/* Left side metrics */}
      <div className="flex flex-wrap items-center gap-4">
        {/* Execution time */}
        <div className="flex items-center gap-1.5" title={`Total request time: ${metadata.took_ms} ms`}>
          <Clock className="w-3.5 h-3.5 text-zinc-400 dark:text-zinc-500" />
          <span>
            <strong className="text-zinc-900 dark:text-zinc-200 font-semibold">
              {metadata.took_ms < 1000 ? `${metadata.took_ms} ms` : `${(metadata.took_ms / 1000).toFixed(2)} s`}
            </strong>
          </span>
        </div>

        {/* Engine status summary */}
        <div className="flex items-center gap-1.5" title={`Engines: ${metadata.providers.successful.join(", ")}`}>
          <Cpu className="w-3.5 h-3.5 text-zinc-400 dark:text-zinc-500" />
          <span className="text-zinc-900 dark:text-zinc-200 font-medium">
            {enginesSummary}
          </span>
        </div>

        {/* Deduplication stats */}
        <div
          className="flex items-center gap-1.5"
          title={`Raw results scraped: ${metadata.stats.raw_count}, unique after deduplication: ${metadata.stats.deduped_count}`}
        >
          <Layers className="w-3.5 h-3.5 text-zinc-400 dark:text-zinc-500" />
          <span>
            <strong className="text-zinc-900 dark:text-zinc-200 font-semibold">
              {metadata.stats.deduped_count}
            </strong>{" "}
            unique ({metadata.stats.raw_count} raw)
          </span>
        </div>
      </div>

      {/* Right side engine badges & cache indicator */}
      <div className="flex flex-wrap items-center gap-2">
        {/* Successful engines */}
        {metadata.providers.successful.map((engine) => (
          <span
            key={engine}
            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
            title={`Successfully queried ${engine}`}
          >
            <CheckCircle2 className="w-2.5 h-2.5" />
            <span>{engine}</span>
          </span>
        ))}

        {/* Failed engines with granular failure message */}
        {failedList.map((fail) => (
          <span
            key={fail.engine}
            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20 cursor-help"
            title={`${fail.engine} failed: ${fail.error}`}
          >
            <AlertTriangle className="w-2.5 h-2.5" />
            <span>{fail.engine}</span>
          </span>
        ))}

        {/* Cache status */}
        {metadata.cached && (
          <span
            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20"
            title="Served instantly from in-memory cache"
          >
            <Zap className="w-2.5 h-2.5" />
            <span>Cached</span>
          </span>
        )}
      </div>

      {/* Active applied filters if any */}
      {metadata.filters && (metadata.filters.freshness || (metadata.filters.include_domains && metadata.filters.include_domains.length > 0) || (metadata.filters.exclude_domains && metadata.filters.exclude_domains.length > 0) || metadata.filters.country || metadata.filters.language) && (
        <div className="w-full pt-2 border-t border-zinc-200/60 dark:border-zinc-800/60 flex flex-wrap items-center gap-1.5 text-[11px] font-mono text-zinc-500">
          <span className="text-zinc-400 dark:text-zinc-500">Active filters:</span>
          {metadata.filters.freshness && (
            <span className="px-1.5 py-0.2 rounded bg-zinc-200/60 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
              freshness:{metadata.filters.freshness}
            </span>
          )}
          {metadata.filters.include_domains?.map((d) => (
            <span key={d} className="px-1.5 py-0.2 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
              +{d}
            </span>
          ))}
          {metadata.filters.exclude_domains?.map((d) => (
            <span key={d} className="px-1.5 py-0.2 rounded bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
              -{d}
            </span>
          ))}
          {metadata.filters.country && (
            <span className="px-1.5 py-0.2 rounded bg-zinc-200/60 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 uppercase">
              geo:{metadata.filters.country}
            </span>
          )}
          {metadata.filters.language && (
            <span className="px-1.5 py-0.2 rounded bg-zinc-200/60 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 uppercase">
              lang:{metadata.filters.language}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
