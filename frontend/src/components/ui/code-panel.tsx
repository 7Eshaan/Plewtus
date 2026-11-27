"use client";

import { useMemo, type ReactNode } from "react";
import { CopyButton } from "./copy-button";
import { handleArrowNav } from "@/lib/arrow-nav";
import { cn } from "@/lib/cn";

export type CodeLanguage = "bash" | "typescript" | "python" | "go" | "json" | "toml" | "yaml";

export interface CodeTab<T extends string> {
  value: T;
  label: string;
}

interface CodePanelProps<T extends string> {
  code: string;
  language: CodeLanguage;
  /** Text shown at the left of the title bar when there are no tabs (a file name, a status line). */
  caption?: ReactNode;
  tabs?: readonly CodeTab<T>[];
  activeTab?: T;
  onTabChange?: (value: T) => void;
  tabsLabel?: string;
  copyLabel: string;
  /** Tailwind max-height class for the scrolling code area. */
  maxHeight?: string;
}

const KEYWORDS: Record<CodeLanguage, readonly string[]> = {
  bash: ["curl", "echo"],
  typescript: ["import", "from", "type", "const", "await", "for", "of", "new", "return"],
  python: ["import", "from", "for", "in", "print", "True", "False", "None"],
  go: ["package", "import", "func", "var", "if", "return", "defer", "nil", "map", "any", "string", "true", "false", "for", "range"],
  json: ["true", "false", "null"],
  toml: ["true", "false"],
  yaml: ["true", "false"],
};

const COMMENT: Record<CodeLanguage, string> = {
  bash: "#[^\\n]*",
  python: "#[^\\n]*",
  typescript: "\\/\\/[^\\n]*",
  go: "\\/\\/[^\\n]*",
  json: "(?!)",
  toml: "#[^\\n]*",
  yaml: "#[^\\n]*",
};

type Token = { text: string; className?: string };

/** Syntax colors for the always-dark code surface (matching json-viewer.tsx). */
const STYLES = {
  comment: "text-code-on-surface-variant/70 italic",
  string: "text-emerald-300",
  key: "text-violet-300",
  number: "text-amber-300",
  keyword: "text-sky-300",
} as const;

/** A deliberately small highlighter: strings, comments, numbers, keywords and JSON keys. */
function tokenize(code: string, language: CodeLanguage): Token[] {
  const pattern = new RegExp(
    [
      `(${COMMENT[language]})`,
      `("(?:\\\\.|[^"\\\\\\n])*"|'(?:\\\\.|[^'\\\\])*'|\`(?:\\\\.|[^\`\\\\])*\`)(\\s*:)?`,
      `(?<![\\w:/.-])(\\d+(?:\\.\\d+)?)\\b`,
      `\\b(${KEYWORDS[language].join("|")})\\b`,
    ].join("|"),
    "g"
  );

  const tokens: Token[] = [];
  let last = 0;
  for (const match of code.matchAll(pattern)) {
    const index = match.index ?? 0;
    if (index > last) tokens.push({ text: code.slice(last, index) });
    const [whole, comment, string, colon, number] = match;
    if (comment) tokens.push({ text: whole, className: STYLES.comment });
    else if (string) {
      const isKey = colon !== undefined;
      tokens.push({ text: string, className: isKey ? STYLES.key : STYLES.string });
      if (colon) tokens.push({ text: colon });
    } else if (number) tokens.push({ text: whole, className: STYLES.number });
    else tokens.push({ text: whole, className: STYLES.keyword });
    last = index + whole.length;
  }
  if (last < code.length) tokens.push({ text: code.slice(last) });
  return tokens;
}

/** A code window (dark in both themes) with optional tabs and a copy button. */
export function CodePanel<T extends string>({
  code,
  language,
  caption,
  tabs,
  activeTab,
  onTabChange,
  tabsLabel,
  copyLabel,
  maxHeight = "max-h-[420px]",
}: CodePanelProps<T>) {
  const tokens = useMemo(() => tokenize(code, language), [code, language]);

  return (
    <div className="flex min-w-0 flex-col overflow-hidden rounded-corner-xl border border-code-outline bg-code-surface shadow-elevation-3">
      <div className="flex items-center justify-between gap-3 px-2.5 pt-2.5">
        {tabs && activeTab && onTabChange ? (
          <div
            role="tablist"
            aria-label={tabsLabel}
            onKeyDown={handleArrowNav}
            className="flex items-center gap-1 overflow-x-auto [scrollbar-width:none]"
          >
            {tabs.map((tab) => {
              const selected = tab.value === activeTab;
              return (
                <button
                  key={tab.value}
                  type="button"
                  role="tab"
                  aria-selected={selected}
                  tabIndex={selected ? 0 : -1}
                  onClick={() => onTabChange(tab.value)}
                  className={cn(
                    "state-layer focus-ring inline-flex h-8 shrink-0 cursor-pointer items-center rounded-corner-full px-3 text-label-lg whitespace-nowrap transition-colors motion-effects",
                    selected ? "bg-code-surface-high text-code-on-surface" : "text-code-on-surface-variant"
                  )}
                >
                  {tab.label}
                </button>
              );
            })}
          </div>
        ) : (
          <div className="truncate px-3 py-1.5 text-label-lg text-code-on-surface-variant">{caption}</div>
        )}
        <CopyButton size="sm" tone="code" getText={() => code} ariaLabel={copyLabel} />
      </div>

      <div className={cn("overflow-auto", maxHeight)}>
        <pre className="min-h-[260px] px-6 pb-7 pt-5 font-mono text-[14px] leading-[1.8] text-code-on-surface [tab-size:4] selection:bg-primary-container/60">
          <code>
            {tokens.map((token, i) =>
              token.className ? (
                <span key={i} className={token.className}>
                  {token.text}
                </span>
              ) : (
                token.text
              )
            )}
          </code>
        </pre>
      </div>
    </div>
  );
}
