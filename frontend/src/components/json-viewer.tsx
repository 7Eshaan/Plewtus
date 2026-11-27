"use client";

import { useMemo } from "react";
import { FileJson, Download } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import { Button } from "./ui/button";
import { chipClasses } from "./ui/chip";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

interface JsonViewerProps {
  response: SearchResponse | null;
  /** When set, the error body is shown instead of the last successful response. */
  error: ErrorResponse | null;
  loading: boolean;
}

/** Syntax colors for the always-dark code surface (both themes). */
const SYNTAX = {
  key: "text-violet-300",
  string: "text-emerald-300",
  number: "text-amber-300",
  boolean: "text-fuchsia-300",
  null: "text-rose-300",
} as const;

function highlightJson(json: string): string {
  const escaped = json.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

  return escaped.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    (match) => {
      let cls: string = SYNTAX.number;
      if (/^"/.test(match)) {
        cls = /:$/.test(match) ? SYNTAX.key : SYNTAX.string;
      } else if (/true|false/.test(match)) {
        cls = SYNTAX.boolean;
      } else if (/null/.test(match)) {
        cls = SYNTAX.null;
      }
      return `<span class="${cls}">${match}</span>`;
    }
  );
}

export function JsonViewer({ response, error, loading }: JsonViewerProps) {
  const payload = error ?? response;
  const jsonString = useMemo(() => (payload ? JSON.stringify(payload, null, 2) : ""), [payload]);
  const highlightedHtml = useMemo(() => (jsonString ? highlightJson(jsonString) : ""), [jsonString]);

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 animate-pulse flex-col gap-3" aria-hidden="true">
        <div className="flex items-center justify-between">
          <div className="h-6 w-40 rounded-corner-xs bg-surface-container-highest" />
          <div className="h-8 w-40 rounded-corner-full bg-surface-container-highest" />
        </div>
        <div className="flex-1 rounded-corner-md bg-code-surface" />
      </div>
    );
  }

  if (!payload) {
    return (
      <div className="my-auto flex min-h-0 flex-1 flex-col items-center justify-center gap-3 rounded-corner-lg border border-dashed border-outline-variant px-4 py-12 text-center">
        <div className="flex size-10 items-center justify-center rounded-full bg-surface-container-highest text-on-surface-variant">
          <FileJson className="size-5" aria-hidden="true" />
        </div>
        <p className="text-body-md text-on-surface-variant">Submit a search query above to inspect real JSON output</p>
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
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2 text-label-md text-on-surface-variant">
          <span className={chipClasses({ interactive: false, dense: true, selected: !error, tone: error ? "error" : "default" })}>
            {error ? (error.code > 0 ? `HTTP ${error.code}` : "No response") : "200 OK"}
          </span>
          <span>{sizeKb} KB</span>
          {!error && response && (
            <>
              <span aria-hidden="true">·</span>
              <span>{response.metadata.took_ms} ms</span>
              <span aria-hidden="true">·</span>
              <span>{response.results.length} results</span>
            </>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outlined" size="xs" icon={<Download />} onClick={handleDownload} aria-label="Download JSON response">
            <span className="hidden sm:inline">Download</span>
          </Button>
          <CopyButton getText={() => jsonString} ariaLabel="Copy JSON response" />
        </div>
      </div>

      {/* Always-dark code surface, in both themes */}
      <div className="min-h-0 flex-1 overflow-auto rounded-corner-md bg-code-surface p-4">
        <pre
          className="whitespace-pre font-mono text-[12px] leading-relaxed text-code-on-surface selection:bg-primary-container/60"
          dangerouslySetInnerHTML={{ __html: highlightedHtml }}
        />
      </div>
    </div>
  );
}
