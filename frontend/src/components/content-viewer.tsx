"use client";

import { useId, useState } from "react";
import ReactMarkdown from "react-markdown";
import { ChevronDown, Sparkles, FileText, User, Clock, AlignLeft } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import { SegmentedControl } from "./ui/segmented-control";
import { cn } from "@/lib/cn";
import type { components } from "@/types/api";

type ExtractedContent = components["schemas"]["ExtractedContent"];

interface ContentViewerProps {
  content: ExtractedContent;
  title: string;
}

type ContentKind = "markdown" | "highlights" | "text";

const KIND_LABELS: Record<ContentKind, string> = {
  markdown: "Extracted Markdown",
  highlights: "Extracted highlights",
  text: "Extracted text",
};

/** Expandable panel with a result's extracted page content. */
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
    <div className="mt-1 overflow-hidden rounded-corner-sm bg-surface-container-high">
      <div className="flex items-center justify-between gap-2 pr-1">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          aria-controls={drawerId}
          className="state-layer focus-ring flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-corner-sm px-3 py-2 text-left"
        >
          <Icon className={cn("size-4 shrink-0", kind === "highlights" ? "text-tertiary" : "text-primary")} aria-hidden="true" />
          <span className="text-label-lg text-on-surface">{KIND_LABELS[kind]}</span>
          <span className="truncate text-label-md text-on-surface-variant">
            {sizeKb} KB{kind === "highlights" ? ` · ${content.highlights?.length} sentences` : ""}
            {content.truncated && content.total_chars
              ? ` · excerpt of ${(content.total_chars / 1000).toFixed(content.total_chars < 10_000 ? 1 : 0)}K chars`
              : ""}
          </span>
          <ChevronDown
            className={cn("ml-auto size-4 shrink-0 text-on-surface-variant transition-transform motion-spatial-fast", expanded && "rotate-180")}
            aria-hidden="true"
          />
        </button>
        <CopyButton getText={() => copyText} ariaLabel="Copy extracted content" size="sm" />
      </div>

      {expanded && (
        <div id={drawerId} className="flex flex-col gap-3 border-t border-outline-variant p-4">
          {kind === "markdown" && (
            <div className="flex items-center justify-between gap-2">
              <span className="truncate text-label-md text-on-surface-variant">Source: {content.title || title}</span>
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
              <div className="max-h-96 overflow-y-auto break-words pr-2 text-body-md text-on-surface">
                <ReactMarkdown
                  components={{
                    h1: ({ children }) => <h4 className="mb-1 mt-3 text-title-lg text-on-surface">{children}</h4>,
                    h2: ({ children }) => <h5 className="mb-1 mt-3 text-title-md text-on-surface">{children}</h5>,
                    h3: ({ children }) => <h6 className="mb-0.5 mt-2 text-title-sm text-on-surface">{children}</h6>,
                    h4: ({ children }) => <h6 className="mb-0.5 mt-2 text-title-sm text-on-surface">{children}</h6>,
                    p: ({ children }) => <p className="mb-2 text-on-surface-variant">{children}</p>,
                    ul: ({ children }) => <ul className="mb-2 list-disc space-y-1 pl-5 text-on-surface-variant">{children}</ul>,
                    ol: ({ children }) => <ol className="mb-2 list-decimal space-y-1 pl-5 text-on-surface-variant">{children}</ol>,
                    blockquote: ({ children }) => (
                      <blockquote className="mb-2 border-l-2 border-outline pl-3 italic text-on-surface-variant">{children}</blockquote>
                    ),
                    pre: ({ children }) => (
                      <pre className="my-2 overflow-x-auto rounded-corner-sm bg-code-surface p-3 font-mono text-[12px] text-code-on-surface [&_code]:border-0 [&_code]:bg-transparent [&_code]:p-0 [&_code]:text-inherit">
                        {children}
                      </pre>
                    ),
                    code: ({ children }) => (
                      <code className="rounded-corner-xs bg-surface-container-highest px-1 py-0.5 font-mono text-[0.85em] text-on-surface">
                        {children}
                      </code>
                    ),
                    table: ({ children }) => (
                      <div className="mb-2 overflow-x-auto">
                        <table className="border-collapse text-body-sm [&_td]:border [&_td]:border-outline-variant [&_td]:px-2 [&_td]:py-1 [&_th]:border [&_th]:border-outline-variant [&_th]:px-2 [&_th]:py-1">
                          {children}
                        </table>
                      </div>
                    ),
                    img: () => null,
                    a: ({ href, children }) => (
                      <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary underline underline-offset-2 hover:opacity-80">
                        {children}
                      </a>
                    ),
                  }}
                >
                  {content.markdown ?? ""}
                </ReactMarkdown>
              </div>
            ) : (
              <pre className="max-h-96 overflow-x-auto whitespace-pre-wrap rounded-corner-sm bg-code-surface p-3 font-mono text-[12px] text-code-on-surface">
                {content.markdown}
              </pre>
            ))}

          {kind === "highlights" && (
            <ul className="flex max-h-80 flex-col gap-2 overflow-y-auto pr-2">
              {content.highlights?.map((highlight, idx) => (
                <li
                  key={idx}
                  className="rounded-r-corner-sm border-l-2 border-tertiary bg-tertiary/8 py-1.5 pl-3 pr-2 text-body-md text-on-surface"
                >
                  {highlight}
                </li>
              ))}
            </ul>
          )}

          {kind === "text" && (
            <p className="max-h-80 overflow-y-auto whitespace-pre-line text-body-md text-on-surface-variant">{content.text}</p>
          )}

          <div className="flex flex-wrap items-center gap-3 border-t border-outline-variant pt-2 text-label-md text-on-surface-variant">
            {content.author && (
              <span className="inline-flex items-center gap-1">
                <User className="size-3.5" aria-hidden="true" />
                {content.author}
              </span>
            )}
            {content.extracted_at && (
              <span className="inline-flex items-center gap-1">
                <Clock className="size-3.5" aria-hidden="true" />
                <time dateTime={content.extracted_at}>Extracted {new Date(content.extracted_at).toLocaleTimeString()}</time>
              </span>
            )}
            {content.description && (
              <span className="hidden max-w-md truncate sm:inline" title={content.description}>
                {content.description}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
