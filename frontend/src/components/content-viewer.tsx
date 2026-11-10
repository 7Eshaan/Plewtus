"use client";

import { useId, useState } from "react";
import ReactMarkdown from "react-markdown";
import { ChevronDown, Sparkles, FileText, User, Clock, AlignLeft } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import { SegmentedControl } from "./ui/segmented-control";
import type { components } from "@/types/api";

type ExtractedContent = components["schemas"]["ExtractedContent"];

interface ContentViewerProps {
  content: ExtractedContent;
  title: string;
}

type ContentKind = "markdown" | "highlights" | "text";

const KIND_LABELS: Record<ContentKind, string> = {
  markdown: "Extracted Markdown",
  highlights: "Extracted Highlights",
  text: "Extracted Text",
};

export function ContentViewer({ content, title }: ContentViewerProps) {
  const [expanded, setExpanded] = useState(false);
  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");
  const drawerId = useId();

  const kind: ContentKind | null = content.markdown?.trim()
    ? "markdown"
    : content.highlights?.length
      ? "highlights"
      : content.text?.trim()
        ? "text"
        : null;
  if (!kind) return null;

  const copyText =
    kind === "markdown"
      ? content.markdown ?? ""
      : kind === "highlights"
        ? (content.highlights ?? []).map((h) => `- ${h}`).join("\n")
        : content.text ?? "";
  const sizeKb = (new Blob([copyText]).size / 1024).toFixed(1);
  const Icon = kind === "markdown" ? FileText : kind === "highlights" ? Sparkles : AlignLeft;

  return (
    <div className="mt-2 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/40 overflow-hidden">
      <div className="flex items-center justify-between gap-2 pr-2 hover:bg-zinc-100/70 dark:hover:bg-zinc-800/50 transition-colors">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          aria-controls={drawerId}
          className="flex-1 min-w-0 px-3 py-2 flex items-center gap-2 text-left cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-brand/60"
        >
          <Icon className={`w-3.5 h-3.5 shrink-0 ${kind === "highlights" ? "text-emerald-500" : "text-zinc-500 dark:text-zinc-400"}`} />
          <span className="text-xs font-mono font-medium text-zinc-900 dark:text-zinc-100">{KIND_LABELS[kind]}</span>
          <span className="text-[11px] font-mono text-zinc-500 truncate">
            ({sizeKb} KB{kind === "highlights" ? ` · ${content.highlights?.length} sentences` : ""})
          </span>
          <ChevronDown
            className={`ml-auto w-3.5 h-3.5 shrink-0 text-zinc-400 dark:text-zinc-500 transition-transform ${expanded ? "rotate-180" : ""}`}
          />
        </button>
        <CopyButton getText={() => copyText} ariaLabel="Copy extracted content" size="sm" />
      </div>

      {expanded && (
        <div id={drawerId} className="p-4 border-t border-zinc-200 dark:border-zinc-800 bg-white/60 dark:bg-zinc-950/40 flex flex-col gap-3">
          {kind === "markdown" && (
            <div className="flex items-center justify-between gap-2 text-xs pb-2 border-b border-zinc-100 dark:border-zinc-800/80">
              <span className="font-mono text-[11px] text-zinc-500 truncate">Source: {content.title || title}</span>
              <SegmentedControl
                ariaLabel="Markdown view"
                size="sm"
                value={viewMode}
                onChange={setViewMode}
                options={[
                  { value: "rendered", label: "Rendered" },
                  { value: "raw", label: "Raw" },
                ]}
              />
            </div>
          )}

          {kind === "markdown" &&
            (viewMode === "rendered" ? (
              <div className="text-xs sm:text-sm text-zinc-800 dark:text-zinc-200 max-h-96 overflow-y-auto pr-2 leading-relaxed break-words">
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h4 className="text-base font-bold mt-3 mb-1 text-zinc-950 dark:text-zinc-50">{children}</h4>,
                    h2: ({ children }) => <h5 className="text-sm font-semibold mt-3 mb-1 text-zinc-900 dark:text-zinc-100">{children}</h5>,
                    h3: ({ children }) => <h6 className="text-xs font-semibold mt-2 mb-0.5 text-zinc-900 dark:text-zinc-200">{children}</h6>,
                    h4: ({ children }) => <h6 className="text-xs font-semibold mt-2 mb-0.5 text-zinc-900 dark:text-zinc-200">{children}</h6>,
                    p: ({ children }) => <p className="mb-2 text-zinc-700 dark:text-zinc-300">{children}</p>,
                    ul: ({ children }) => <ul className="list-disc pl-4 space-y-1 mb-2 text-zinc-700 dark:text-zinc-300">{children}</ul>,
                    ol: ({ children }) => <ol className="list-decimal pl-4 space-y-1 mb-2 text-zinc-700 dark:text-zinc-300">{children}</ol>,
                    blockquote: ({ children }) => (
                      <blockquote className="border-l-2 border-zinc-300 dark:border-zinc-700 pl-3 italic text-zinc-600 dark:text-zinc-400 mb-2">{children}</blockquote>
                    ),
                    pre: ({ children }) => (
                      <pre className="p-3 rounded-md bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 overflow-x-auto text-[11px] font-mono text-zinc-900 dark:text-zinc-100 my-2 [&_code]:border-0 [&_code]:bg-transparent [&_code]:p-0">
                        {children}
                      </pre>
                    ),
                    code: ({ children }) => (
                      <code className="font-mono text-[11px] px-1 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 border border-zinc-200 dark:border-zinc-700/60">
                        {children}
                      </code>
                    ),
                    table: ({ children }) => (
                      <div className="overflow-x-auto mb-2">
                        <table className="text-[11px] border-collapse [&_td]:border [&_th]:border [&_td]:border-zinc-200 [&_th]:border-zinc-200 dark:[&_td]:border-zinc-800 dark:[&_th]:border-zinc-800 [&_td]:px-2 [&_th]:px-2 [&_td]:py-1 [&_th]:py-1">
                          {children}
                        </table>
                      </div>
                    ),
                    img: () => null,
                    a: ({ href, children }) => (
                      <a href={href} target="_blank" rel="noopener noreferrer" className="text-brand dark:text-brand-light underline underline-offset-2 hover:opacity-80">
                        {children}
                      </a>
                    ),
                  }}
                >
                  {content.markdown ?? ""}
                </ReactMarkdown>
              </div>
            ) : (
              <pre className="p-3 rounded-md bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 overflow-x-auto text-[11px] font-mono text-zinc-800 dark:text-zinc-200 max-h-96 whitespace-pre-wrap">
                {content.markdown}
              </pre>
            ))}

          {kind === "highlights" && (
            <ul className="flex flex-col gap-2 max-h-80 overflow-y-auto pr-2">
              {content.highlights?.map((highlight, idx) => (
                <li
                  key={idx}
                  className="pl-3 py-1 border-l-2 border-emerald-500/70 text-xs text-zinc-700 dark:text-zinc-300 leading-relaxed bg-emerald-500/5 rounded-r-md"
                >
                  {highlight}
                </li>
              ))}
            </ul>
          )}

          {kind === "text" && (
            <p className="text-xs text-zinc-700 dark:text-zinc-300 leading-relaxed whitespace-pre-line max-h-80 overflow-y-auto">{content.text}</p>
          )}

          <div className="flex flex-wrap items-center gap-3 pt-2 border-t border-zinc-100 dark:border-zinc-800/80 text-[11px] font-mono text-zinc-500">
            {content.author && (
              <span className="inline-flex items-center gap-1">
                <User className="w-3 h-3" />
                {content.author}
              </span>
            )}
            {content.extracted_at && (
              <span className="inline-flex items-center gap-1">
                <Clock className="w-3 h-3" />
                <time dateTime={content.extracted_at}>Extracted {new Date(content.extracted_at).toLocaleTimeString()}</time>
              </span>
            )}
            {content.description && (
              <span className="truncate max-w-md hidden sm:inline" title={content.description}>
                {content.description}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
