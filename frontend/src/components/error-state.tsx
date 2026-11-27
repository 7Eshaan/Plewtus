import { AlertCircle, RotateCcw } from "lucide-react";
import type { components } from "@/types/api";

type ErrorResponse = components["schemas"]["ErrorResponse"];

interface ErrorStateProps {
  error: ErrorResponse;
  onRetry?: () => void;
}

/** A failed search, shown on the M3 error container. */
export function ErrorState({ error, onRetry }: ErrorStateProps) {
  const isRateLimit = error.code === 429 || error.error === "rate_limit_exceeded" || error.error === "quota_exceeded";
  const isBackendUnavailable = error.error === "backend_unavailable" || error.error === "backend_timeout" || error.code === 502;
  const isValidation = error.code === 400;

  return (
    <div
      role="alert"
      className="flex flex-col items-start justify-between gap-4 rounded-corner-md bg-error-container p-4 text-on-error-container sm:flex-row sm:items-center sm:p-5"
    >
      <div className="flex items-start gap-3">
        <AlertCircle className="mt-0.5 size-5 shrink-0" aria-hidden="true" />
        <div className="flex flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-title-sm">
              {isRateLimit
                ? "Rate limit reached"
                : isBackendUnavailable
                  ? "Search backend unavailable"
                  : isValidation
                    ? "Invalid search"
                    : "Search failed"}
            </span>
            <code className="rounded-corner-xs bg-on-error-container/10 px-1.5 py-0.5 font-mono text-label-sm">{error.error}</code>
          </div>
          <p className="max-w-xl text-body-md">{error.message}</p>
          {isBackendUnavailable && (
            <p className="text-body-sm opacity-80">The backend may be starting up or temporarily unreachable. Wait a moment and retry.</p>
          )}
        </div>
      </div>

      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="state-layer focus-ring inline-flex h-8 shrink-0 cursor-pointer items-center gap-1.5 rounded-[16px] border border-on-error-container/40 pl-2.5 pr-3 text-label-lg transition-[border-radius] motion-spatial-fast active:rounded-corner-sm [&_svg]:size-[18px]"
        >
          <RotateCcw aria-hidden="true" />
          Retry
        </button>
      )}
    </div>
  );
}
