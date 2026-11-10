"use client";

import { useMemo } from "react";
import { FileJson, Download, Layers } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

interface JsonViewerProps {
  response: SearchResponse | null;
  /** When set, the error body is shown instead of the last successful response. */
  error: ErrorResponse | null;
  loading: boolean;
}

function highlightJson(json: string): string {
  const escaped = json
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

  return escaped.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    (match) => {
      let cls = "text-amber-400 dark:text-amber-300"; // number
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = "text-sky-400 dark:text-sky-300 font-medium"; // key
        } else {
          cls = "text-emerald-400 dark:text-emerald-300"; // string
        }
      } else if (/true|false/.test(match)) {
        cls = "text-purple-400 dark:text-purple-300 font-semibold"; // boolean
      } else if (/null/.test(match)) {
        cls = "text-rose-400 dark:text-rose-300 font-semibold"; // null
      }
      return `<span class="${cls}">${match}</span>`;
    }
  );
}

export function JsonViewer({ response, error, loading }: JsonViewerProps) {
  const payload = error ?? response;
  const jsonString = useMemo(() => {
    return payload ? JSON.stringify(payload, null, 2) : "";
  }, [payload]);

  const highlightedHtml = useMemo(() => {
    return jsonString ? highlightJson(jsonString) : "";
  }, [jsonString]);

  if (loading) {
    return (
      <div className="flex-1 min-h-0 flex flex-col gap-2 p-2 animate-pulse text-left">
        <div className="flex items-center justify-between">
          <div className="w-32 h-4 rounded bg-zinc-200 dark:bg-zinc-800" />
          <div className="w-24 h-4 rounded bg-zinc-200 dark:bg-zinc-800" />
        </div>
        <div className="flex-1 rounded-lg bg-zinc-100 dark:bg-zinc-900/60" />
      </div>
    );
  }

  if (!payload) {
    return (
      <div className="flex-1 min-h-0 py-12 px-4 rounded-xl border border-dashed border-zinc-200 dark:border-zinc-800 flex flex-col items-center justify-center text-center gap-2 text-left my-auto">
        <div className="w-8 h-8 rounded-lg bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex items-center justify-center text-zinc-400">
          <FileJson className="w-4 h-4" />
        </div>
        <p className="text-xs font-mono text-zinc-500">
          Submit a search query above to inspect real JSON output
        </p>
      </div>
    );
  }

  const sizeKb = (jsonString.length / 1024).toFixed(1);

  function handleDownload() {
    const blob = new Blob([jsonString], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `plewtus-response-${Date.now()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex-1 min-h-0 flex flex-col gap-2 text-left">
      {/* Top Controls Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-shrink-0 text-xs font-mono">
        {/* Left: Status & Metrics */}
        <div className="flex items-center gap-2 text-[11px] text-zinc-500 dark:text-zinc-400">
          {error ? (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
              {error.code > 0 ? `HTTP ${error.code}` : "No response"}
            </span>
          ) : (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
              200 OK
            </span>
          )}
          <span>{sizeKb} KB</span>
          {!error && response && (
            <>
              <span>•</span>
              <span>{response.metadata.took_ms} ms</span>
              <span>•</span>
              <span className="inline-flex items-center gap-1">
                <Layers className="w-3 h-3 text-zinc-400" />
                <span>{response.results.length} results</span>
              </span>
            </>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={handleDownload}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:border-zinc-400 dark:hover:border-zinc-600 text-xs font-mono transition-colors cursor-pointer shadow-2xs"
            title="Download JSON file"
            aria-label="Download JSON response"
          >
            <Download className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Download</span>
          </button>

          <CopyButton getText={() => jsonString} ariaLabel="Copy JSON response" />
        </div>
      </div>

      {/* Code Display Area: Left-aligned, syntax highlighted, scrollable */}
      <div className="flex-1 min-h-0 rounded-lg border border-zinc-200 dark:border-zinc-800/80 bg-[#0a0a0c] p-3 overflow-auto text-left font-mono text-xs leading-relaxed shadow-inner">
        <pre
          className="text-left font-mono text-xs text-zinc-300 selection:bg-zinc-800 whitespace-pre"
          dangerouslySetInnerHTML={{ __html: highlightedHtml }}
        />
      </div>
    </div>
  );
}
