"use client";

import { useMemo, type ReactNode } from "react";
import { CopyButton } from "./copy-button";
import { handleArrowNav } from "@/lib/arrow-nav";

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

const STYLES = {
  comment: "text-zinc-500 italic",
  string: "text-emerald-200/80",
  key: "text-violet-200",
  number: "text-amber-200/80",
  keyword: "text-sky-300/90",
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

/** A dark code window with optional text tabs and a copy button. */
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
    <div className="rounded-2xl border border-white/[0.08] bg-[#0b0b0e] overflow-hidden shadow-[0_20px_60px_-30px_rgba(0,0,0,0.6)] flex flex-col min-w-0">
      <div className="flex items-center justify-between gap-3 px-2.5 pt-2.5">
        {tabs && activeTab && onTabChange ? (
          <div role="tablist" aria-label={tabsLabel} onKeyDown={handleArrowNav} className="flex items-center gap-0.5 overflow-x-auto [scrollbar-width:none]">
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
                  className={`px-2.5 py-1.5 rounded-lg text-[14px] whitespace-nowrap transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 ${
                    selected ? "bg-white/[0.08] text-zinc-100" : "text-zinc-500 hover:text-zinc-300"
                  }`}
                >
                  {tab.label}
                </button>
              );
            })}
          </div>
        ) : (
          <div className="px-3 py-1.5 text-[13px] text-zinc-400 truncate">{caption}</div>
        )}
        <div className="[&_button]:border-transparent [&_button]:bg-transparent [&_button]:text-zinc-500 [&_button:hover]:text-zinc-200 [&_button:hover]:border-transparent">
          <CopyButton size="sm" getText={() => code} ariaLabel={copyLabel} />
        </div>
      </div>

      <div className={`overflow-auto ${maxHeight}`}>
        <pre className="px-6 pt-5 pb-7 text-[14px] leading-[1.8] min-h-[260px] font-mono [tab-size:4] text-zinc-300 selection:bg-brand/40">
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
