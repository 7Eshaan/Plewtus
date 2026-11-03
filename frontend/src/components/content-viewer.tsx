"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import { ChevronDown, Copy, Check, Sparkles, FileText, User, Clock } from "lucide-react";
import type { components } from "@/types/api";

type ExtractedContent = components["schemas"]["ExtractedContent"];

interface ContentViewerProps {
  content: ExtractedContent;
  title: string;
}

export function ContentViewer({ content, title }: ContentViewerProps) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);
  const [viewMode, setViewMode] = useState<"rendered" | "raw">("rendered");

  const hasMarkdown = Boolean(content.markdown && content.markdown.trim().length > 0);
  const hasHighlights = Boolean(content.highlights && content.highlights.length > 0);
  const hasText = Boolean(content.text && content.text.trim().length > 0);

  if (!hasMarkdown && !hasHighlights && !hasText) {
    return null;
  }

  const primaryContentText = content.markdown || content.text || content.highlights?.join("\n\n") || "";
  const charCount = primaryContentText.length;
  const sizeKb = (charCount / 1024).toFixed(1);

  async function handleCopy(e: React.MouseEvent) {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(primaryContentText);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback if clipboard API is blocked
    }
  }

  return (
    <div className="mt-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/40 overflow-hidden transition-all duration-150">
      {/* Accordion Header */}
      <div
        role="button"
        tabIndex={0}
        onClick={() => setExpanded(!expanded)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            setExpanded(!expanded);
          }
        }}
        className="w-full px-3.5 py-2.5 flex items-center justify-between text-left hover:bg-zinc-100/70 dark:hover:bg-zinc-800/50 transition-colors select-none cursor-pointer"
        aria-expanded={expanded}
      >
        <div className="flex items-center gap-2">
          {hasMarkdown ? (
            <FileText className="w-3.5 h-3.5 text-zinc-500 dark:text-zinc-400" />
          ) : (
            <Sparkles className="w-3.5 h-3.5 text-emerald-500" />
          )}
          <span className="text-xs font-mono font-medium text-zinc-900 dark:text-zinc-100">
            {hasMarkdown ? "Extracted Markdown" : "Extracted Highlights"}
          </span>
          <span className="text-[11px] font-mono text-zinc-400 dark:text-zinc-500">
            ({sizeKb} KB{hasHighlights ? ` • ${content.highlights?.length} chunks` : ""})
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* Copy Button */}
          <button
            type="button"
            onClick={handleCopy}
            className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-mono border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 hover:border-zinc-400 dark:hover:border-zinc-500 transition-colors cursor-pointer"
            aria-label="Copy extracted content"
            title="Copy extracted content"
          >
            {copied ? (
              <>
                <Check className="w-3 h-3 text-emerald-500" />
                <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3 h-3" />
                <span>Copy as Markdown</span>
              </>
            )}
          </button>

          <ChevronDown
            className={`w-3.5 h-3.5 text-zinc-400 dark:text-zinc-500 transition-transform duration-150 ${
              expanded ? "rotate-180" : ""
            }`}
          />
        </div>
      </div>

      {/* Expanded Drawer */}
      {expanded && (
        <div className="p-4 border-t border-zinc-200 dark:border-zinc-800 bg-white/60 dark:bg-zinc-950/40 flex flex-col gap-3">
          {/* View mode toggle (Rendered vs Raw Markdown) */}
          {hasMarkdown && (
            <div className="flex items-center justify-between text-xs pb-2 border-b border-zinc-100 dark:border-zinc-800/80">
              <span className="font-mono text-[11px] text-zinc-400 dark:text-zinc-500 truncate max-w-sm">
                Document source: {title}
              </span>
              <div className="inline-flex items-center p-0.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 font-mono text-[11px]">
                <button
                  type="button"
                  onClick={() => setViewMode("rendered")}
                  className={`px-2 py-0.5 rounded transition-all cursor-pointer ${
                    viewMode === "rendered"
                      ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-50 shadow-2xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  Rendered
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("raw")}
                  className={`px-2 py-0.5 rounded transition-all cursor-pointer ${
                    viewMode === "raw"
                      ? "bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-50 shadow-2xs font-medium"
                      : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
                  }`}
                >
                  Raw Markdown
                </button>
              </div>
            </div>
          )}

          {/* Content Body */}
          {hasMarkdown ? (
            viewMode === "rendered" ? (
              <div className="text-xs sm:text-sm text-zinc-800 dark:text-zinc-200 max-h-96 overflow-y-auto pr-2 leading-relaxed space-y-3 prose-zinc dark:prose-invert">
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h1 className="text-base font-bold mt-2 mb-1 text-zinc-950 dark:text-zinc-50">{children}</h1>,
                    h2: ({ children }) => <h2 className="text-sm font-semibold mt-2 mb-1 text-zinc-900 dark:text-zinc-100">{children}</h2>,
                    h3: ({ children }) => <h3 className="text-xs font-semibold mt-1.5 mb-0.5 text-zinc-900 dark:text-zinc-200">{children}</h3>,
                    p: ({ children }) => <p className="mb-2 leading-relaxed text-zinc-700 dark:text-zinc-300">{children}</p>,
                    ul: ({ children }) => <ul className="list-disc pl-4 space-y-1 mb-2 text-zinc-700 dark:text-zinc-300">{children}</ul>,
                    ol: ({ children }) => <ol className="list-decimal pl-4 space-y-1 mb-2 text-zinc-700 dark:text-zinc-300">{children}</ol>,
                    li: ({ children }) => <li>{children}</li>,
                    code: ({ children }) => (
                      <code className="font-mono text-[11px] px-1 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 border border-zinc-200 dark:border-zinc-700/60">
                        {children}
                      </code>
                    ),
                    pre: ({ children }) => (
                      <pre className="p-3 rounded-md bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 overflow-x-auto text-[11px] font-mono text-zinc-900 dark:text-zinc-100 my-2">
                        {children}
                      </pre>
                    ),
                    a: ({ href, children }) => (
                      <a href={href} target="_blank" rel="noopener noreferrer" className="text-zinc-900 dark:text-zinc-100 underline underline-offset-2 hover:opacity-80">
                        {children}
                      </a>
                    ),
                  }}
                >
                  {content.markdown || ""}
                </ReactMarkdown>
              </div>
            ) : (
              <pre className="p-3 rounded-md bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 overflow-x-auto text-[11px] font-mono text-zinc-800 dark:text-zinc-200 max-h-96 whitespace-pre-wrap">
                {content.markdown}
              </pre>
            )
          ) : hasHighlights ? (
            <div className="flex flex-col gap-2 max-h-80 overflow-y-auto pr-2">
              {content.highlights?.map((highlight, idx) => (
                <div
                  key={idx}
                  className="pl-3 py-1 border-l-2 border-emerald-500/70 text-xs text-zinc-700 dark:text-zinc-300 leading-relaxed font-sans bg-emerald-500/5 rounded-r-md"
                >
                  {highlight}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-zinc-700 dark:text-zinc-300 leading-relaxed whitespace-pre-line max-h-80 overflow-y-auto">
              {content.text}
            </p>
          )}

          {/* Footer Metadata */}
          <div className="flex flex-wrap items-center gap-3 pt-2 border-t border-zinc-100 dark:border-zinc-800/80 text-[11px] font-mono text-zinc-400 dark:text-zinc-500">
            {content.author && (
              <span className="inline-flex items-center gap-1">
                <User className="w-3 h-3" />
                <span>{content.author}</span>
              </span>
            )}
            {content.extracted_at && (
              <span className="inline-flex items-center gap-1">
                <Clock className="w-3 h-3" />
                <span>Extracted {new Date(content.extracted_at).toLocaleTimeString()} UTC</span>
              </span>
            )}
            {content.description && (
              <span className="truncate max-w-md hidden sm:inline" title={content.description}>
                Meta: {content.description}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
