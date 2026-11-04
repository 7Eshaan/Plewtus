"use client";

import { AlertCircle, RotateCcw } from "lucide-react";
import type { components } from "@/types/api";

type ErrorResponse = components["schemas"]["ErrorResponse"];

interface ErrorStateProps {
  error: ErrorResponse;
  onRetry?: () => void;
}

export function ErrorState({ error, onRetry }: ErrorStateProps) {
  const isRateLimit = error.code === 429 || error.error === "rate_limit_exceeded";
  const isBackendUnavailable = error.code === 502 || error.error === "backend_unavailable";

  return (
    <div className="p-4 sm:p-5 rounded-xl border border-rose-500/20 dark:border-rose-500/30 bg-rose-50/50 dark:bg-rose-950/15 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
      <div className="flex items-start gap-3">
        <div className="w-8 h-8 rounded-full bg-rose-100 dark:bg-rose-900/40 text-rose-600 dark:text-rose-400 flex items-center justify-center flex-shrink-0 mt-0.5 sm:mt-0">
          <AlertCircle className="w-4 h-4" />
        </div>
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-rose-950 dark:text-rose-200">
              {isRateLimit
                ? "Rate Limit Exceeded"
                : isBackendUnavailable
                ? "Backend Offline"
                : "Search Failed"}
            </span>
            <span className="font-mono text-xs px-1.5 py-0.2 rounded bg-rose-200/50 dark:bg-rose-900/40 text-rose-700 dark:text-rose-300">
              {error.error}
            </span>
          </div>
          <p className="text-xs text-rose-800 dark:text-rose-300/80 leading-relaxed max-w-xl">
            {error.message}
          </p>
          {isBackendUnavailable && (
            <p className="text-[11px] font-mono text-rose-600 dark:text-rose-400 mt-1">
              Start the Go backend with: PLEWTUS_PORT=8890 PLEWTUS_API_KEYS=pro_live_agent123:pro ./bin/plewtus
            </p>
          )}
        </div>
      </div>

      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-rose-300 dark:border-rose-700/60 bg-white dark:bg-rose-950/40 hover:bg-rose-50 dark:hover:bg-rose-900/60 text-xs font-medium text-rose-900 dark:text-rose-200 transition-colors flex-shrink-0 cursor-pointer shadow-2xs"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          <span>Retry</span>
        </button>
      )}
    </div>
  );
}
