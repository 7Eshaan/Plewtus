"use client";

import { useState } from "react";
import { Terminal, Copy, Check, Info } from "lucide-react";
import type {
  ProviderName,
  ExtractionMode,
  AdvancedOptions,
} from "@/types/search";

interface CodeSnippetsProps {
  query: string;
  providers: ProviderName[];
  extractionMode: ExtractionMode;
  advancedOptions: AdvancedOptions;
}

type SnippetLang = "curl" | "javascript" | "python";

export function CodeSnippets({
  query,
  providers,
  extractionMode,
  advancedOptions,
}: CodeSnippetsProps) {
  const [activeLang, setActiveLang] = useState<SnippetLang>("curl");
  const [copied, setCopied] = useState(false);

  // Construct request payload reflecting chosen options
  const q = query.trim() || "golang concurrency patterns";
  const payload: Record<string, unknown> = { query: q };

  if (advancedOptions.limit !== 50) {
    payload.limit = advancedOptions.limit;
  }
  if (providers.length < 3) {
    payload.providers = providers;
  }
  if (advancedOptions.freshness) {
    payload.freshness = advancedOptions.freshness;
  }
  if (advancedOptions.country.trim()) {
    payload.country = advancedOptions.country.trim().toLowerCase();
  }
  if (advancedOptions.language.trim()) {
    payload.language = advancedOptions.language.trim().toLowerCase();
  }
  if (advancedOptions.includeDomains.trim()) {
    payload.include_domains = advancedOptions.includeDomains
      .split(",")
      .map((d) => d.trim())
      .filter(Boolean);
  }
  if (advancedOptions.excludeDomains.trim()) {
    payload.exclude_domains = advancedOptions.excludeDomains
      .split(",")
      .map((d) => d.trim())
      .filter(Boolean);
  }
  if (extractionMode !== "none") {
    payload.extract = {
      enabled: true,
      mode: extractionMode,
      max_pages: advancedOptions.maxPages,
      max_chars_per_page: advancedOptions.maxCharsPerPage,
    };
  }

  const jsonPayloadString = JSON.stringify(payload, null, 2);

  // Generate language-specific code snippets using PLACEHOLDER key strictly
  const PLACEHOLDER_KEY = "YOUR_API_KEY";

  function getCode(lang: SnippetLang): string {
    switch (lang) {
      case "curl":
        return `curl -X POST http://localhost:8080/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${PLACEHOLDER_KEY}" \\
  -d '${JSON.stringify(payload, null, 2).replace(/'/g, "'\\''")}'`;

      case "javascript":
        return `// Execute Plewtus search with fetch (Node 18+ or browser)
const response = await fetch("http://localhost:8080/search", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer ${PLACEHOLDER_KEY}",
  },
  body: JSON.stringify(${jsonPayloadString}),
});

const data = await response.json();
console.log(\`Results: \${data.results.length} ranked in \${data.metadata.took_ms}ms\`);
for (const item of data.results) {
  console.log(\`#\${item.rank} [\${item.score.toFixed(4)}] \${item.title} -> \${item.url}\`);
}`;

      case "python":
        return `# Execute Plewtus search with python-requests
import json
import requests

url = "http://localhost:8080/search"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer ${PLACEHOLDER_KEY}",
}
payload = json.loads(${JSON.stringify(JSON.stringify(payload))})

response = requests.post(url, headers=headers, json=payload)
data = response.json()

print(f"Results: {len(data['results'])} ranked in {data['metadata']['took_ms']}ms")
for item in data["results"]:
    print(f"#{item['rank']} [{item['score']:.4f}] {item['title']} -> {item['url']}")`;
    }
  }

  const currentCode = getCode(activeLang);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(currentCode);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback if blocked
    }
  }

  return (
    <div className="flex-1 min-h-0 flex flex-col gap-2 text-left">
      {/* Header Bar */}
      <div className="flex items-center justify-between gap-2 flex-shrink-0 text-xs font-mono">
        <div className="flex items-center gap-2">
          <Terminal className="w-3.5 h-3.5 text-zinc-400" />
          <span className="font-mono text-xs font-semibold text-zinc-700 dark:text-zinc-300 uppercase tracking-wide">
            Generated API Request
          </span>
        </div>

        {/* Language Tabs & Copy Button */}
        <div className="flex items-center gap-1.5">
          <div className="inline-flex items-center p-0.5 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 font-mono text-xs">
            <button
              type="button"
              onClick={() => setActiveLang("curl")}
              className={`px-2 py-0.5 rounded transition-all cursor-pointer ${
                activeLang === "curl"
                  ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                  : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
              }`}
            >
              cURL
            </button>
            <button
              type="button"
              onClick={() => setActiveLang("javascript")}
              className={`px-2 py-0.5 rounded transition-all cursor-pointer ${
                activeLang === "javascript"
                  ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                  : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
              }`}
            >
              JavaScript
            </button>
            <button
              type="button"
              onClick={() => setActiveLang("python")}
              className={`px-2 py-0.5 rounded transition-all cursor-pointer ${
                activeLang === "python"
                  ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                  : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
              }`}
            >
              Python
            </button>
          </div>

          <button
            type="button"
            onClick={handleCopy}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:border-zinc-400 dark:hover:border-zinc-600 text-xs font-mono transition-colors cursor-pointer shadow-2xs"
            aria-label="Copy snippet to clipboard"
            title="Copy code snippet"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-500" />
                <span className="text-emerald-600 dark:text-emerald-400 font-semibold">Copied!</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Code Display Area: Left-aligned, scrollable */}
      <div className="flex-1 min-h-0 rounded-lg border border-zinc-200 dark:border-zinc-800/80 bg-[#0a0a0c] p-3 overflow-auto text-left font-mono text-xs leading-relaxed shadow-inner">
        <pre className="text-left font-mono text-xs text-zinc-300 selection:bg-zinc-800 whitespace-pre">{currentCode}</pre>
      </div>

      {/* Security Footer Notice */}
      <div className="flex items-center justify-between text-[10px] font-mono text-zinc-500 dark:text-zinc-400 pt-0.5 flex-shrink-0">
        <div className="flex items-center gap-1">
          <Info className="w-3 h-3 text-zinc-400" />
          <span>
            Key placeholder <code className="px-1 py-0.2 rounded bg-zinc-200/60 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-bold">{PLACEHOLDER_KEY}</code>. Demo keys are never exposed in client snippets.
          </span>
        </div>
        <span className="hidden sm:inline text-zinc-400 dark:text-zinc-500">
          Reflects active search state
        </span>
      </div>
    </div>
  );
}
