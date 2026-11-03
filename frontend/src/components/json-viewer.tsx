"use client";

import { useState } from "react";
import { FileJson, Copy, Check, Download, Layers } from "lucide-react";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];

interface JsonViewerProps {
  response: SearchResponse | null;
  loading: boolean;
}

export function JsonViewer({ response, loading }: JsonViewerProps) {
  const [copied, setCopied] = useState(false);

  if (loading) {
    return (
      <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] p-6 flex flex-col gap-3 animate-pulse">
        <div className="w-48 h-4 rounded bg-zinc-200 dark:bg-zinc-800" />
        <div className="w-full h-64 rounded bg-zinc-100 dark:bg-zinc-900/60" />
      </div>
    );
  }

  if (!response) {
    return (
      <div className="py-16 px-4 rounded-xl border border-dashed border-zinc-300 dark:border-zinc-800 flex flex-col items-center justify-center text-center gap-3 bg-zinc-50/40 dark:bg-zinc-900/20">
        <div className="w-10 h-10 rounded-full bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex items-center justify-center text-zinc-500">
          <FileJson className="w-5 h-5" />
        </div>
        <div className="flex flex-col gap-1 max-w-sm">
          <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            No API response to inspect yet
          </h3>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            Submit a search query above to view the full, real JSON response returned by the Plewtus metasearch engine.
          </p>
        </div>
      </div>
    );
  }

  const jsonString = JSON.stringify(response, null, 2);
  const sizeKb = (jsonString.length / 1024).toFixed(1);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(jsonString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback
    }
  }

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
    <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] overflow-hidden shadow-2xs flex flex-col">
      {/* Header */}
      <div className="px-4 py-3 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between bg-zinc-50/60 dark:bg-zinc-900/40">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <FileJson className="w-4 h-4 text-emerald-500" />
            <span className="font-mono text-xs font-semibold text-zinc-900 dark:text-zinc-100 uppercase tracking-wide">
              API JSON Response
            </span>
          </div>

          <div className="flex items-center gap-1.5 text-[11px] font-mono text-zinc-500">
            <span className="px-1.5 py-0.2 rounded bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-semibold">
              200 OK
            </span>
            <span>•</span>
            <span>{sizeKb} KB</span>
            <span>•</span>
            <span>{response.metadata.took_ms} ms</span>
            <span>•</span>
            <span className="inline-flex items-center gap-1">
              <Layers className="w-3 h-3 text-zinc-400" />
              <span>{response.results.length} items</span>
            </span>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={handleDownload}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 hover:border-zinc-400 dark:hover:border-zinc-500 text-xs font-mono transition-colors cursor-pointer"
            title="Download JSON file"
            aria-label="Download JSON response"
          >
            <Download className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Download</span>
          </button>

          <button
            type="button"
            onClick={handleCopy}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 hover:border-zinc-400 dark:hover:border-zinc-500 text-xs font-mono transition-colors cursor-pointer"
            title="Copy JSON to clipboard"
            aria-label="Copy JSON response"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-500" />
                <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy JSON</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Code Display Area */}
      <div className="relative p-4 bg-zinc-950 text-zinc-200 overflow-x-auto text-xs font-mono leading-relaxed max-h-[500px]">
        <pre className="selection:bg-zinc-800">{jsonString}</pre>
      </div>

      {/* Footer Info */}
      <div className="px-4 py-2 border-t border-zinc-200 dark:border-zinc-800/80 bg-zinc-50/50 dark:bg-zinc-900/30 flex items-center justify-between text-[11px] font-mono text-zinc-500 dark:text-zinc-400">
        <span>
          Echoed query: &ldquo;{response.query}&rdquo;
        </span>
        <span>
          Providers: {response.metadata.providers.successful.join(", ")}
        </span>
      </div>
    </div>
  );
}
