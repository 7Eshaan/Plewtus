"use client";

import { useState } from "react";
import { Terminal, Info } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import { SegmentedControl } from "./ui/segmented-control";
import { buildSearchRequest } from "@/lib/build-request";
import {
  EXAMPLE_QUERIES,
  type ProviderName,
  type ExtractionMode,
  type AdvancedOptions,
} from "@/types/search";

interface CodeSnippetsProps {
  query: string;
  providers: ProviderName[];
  extractionMode: ExtractionMode;
  advancedOptions: AdvancedOptions;
}

type SnippetLang = "curl" | "javascript" | "python";

/** Renders a JSON-compatible value as a Python literal (True/False/None). */
function toPython(value: unknown, indent = 0): string {
  const pad = "    ".repeat(indent + 1);
  const end = "    ".repeat(indent);
  if (value === null || value === undefined) return "None";
  if (typeof value === "boolean") return value ? "True" : "False";
  if (typeof value === "number") return String(value);
  if (typeof value === "string") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map((v) => toPython(v, indent + 1)).join(", ")}]`;
  const entries = Object.entries(value as Record<string, unknown>);
  if (entries.length === 0) return "{}";
  return `{\n${entries.map(([k, v]) => `${pad}${JSON.stringify(k)}: ${toPython(v, indent + 1)},`).join("\n")}\n${end}}`;
}

export function CodeSnippets({
  query,
  providers,
  extractionMode,
  advancedOptions,
}: CodeSnippetsProps) {
  const [activeLang, setActiveLang] = useState<SnippetLang>("curl");

  const payload = buildSearchRequest(query.trim() || EXAMPLE_QUERIES[1], providers, extractionMode, advancedOptions);
  const jsonPayloadString = JSON.stringify(payload, null, 2);

  // Generate language-specific code snippets using PLACEHOLDER key strictly
  const PLACEHOLDER_KEY = "YOUR_API_KEY";

  function getCode(lang: SnippetLang): string {
    switch (lang) {
      case "curl":
        return `curl -X POST http://localhost:8080/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${PLACEHOLDER_KEY}" \\
  -d '${jsonPayloadString.replace(/'/g, "'\\''")}'`;

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
  console.log(\`#\${item.rank} [\${item.ranking_score.toFixed(2)}] \${item.title} -> \${item.url}\`);
}`;

      case "python":
        return `# Execute Plewtus search with python-requests
import requests

url = "http://localhost:8080/search"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer ${PLACEHOLDER_KEY}",
}
payload = ${toPython(payload)}

response = requests.post(url, headers=headers, json=payload, timeout=30)
response.raise_for_status()
data = response.json()

print(f"Results: {len(data['results'])} ranked in {data['metadata']['took_ms']}ms")
for item in data["results"]:
    print(f"#{item['rank']} [{item['ranking_score']:.2f}] {item['title']} -> {item['url']}")`;
    }
  }

  const currentCode = getCode(activeLang);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 text-left">
      {/* Header bar */}
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Terminal className="size-4 text-primary" aria-hidden="true" />
          <span className="text-title-sm text-on-surface">Generated API request</span>
        </div>

        {/* Language choice and copy button */}
        <div className="flex items-center gap-2">
          <SegmentedControl
            ariaLabel="Snippet language"
            size="sm"
            value={activeLang}
            onChange={setActiveLang}
            options={[
              { value: "curl", label: "cURL" },
              { value: "javascript", label: "JavaScript" },
              { value: "python", label: "Python" },
            ]}
          />
          <CopyButton getText={() => currentCode} ariaLabel="Copy snippet to clipboard" />
        </div>
      </div>

      {/* Always-dark code surface, in both themes */}
      <div className="min-h-0 flex-1 overflow-auto rounded-corner-md bg-code-surface p-4">
        <pre className="whitespace-pre font-mono text-[12px] leading-relaxed text-code-on-surface selection:bg-primary-container/60">
          {currentCode}
        </pre>
      </div>

      {/* Security notice */}
      <div className="flex shrink-0 items-center justify-between gap-2 text-body-sm text-on-surface-variant">
        <div className="flex items-center gap-1.5">
          <Info className="size-3.5 shrink-0" aria-hidden="true" />
          <span>
            Key placeholder{" "}
            <code className="rounded-corner-xs bg-surface-container-highest px-1 py-0.5 font-mono text-on-surface">{PLACEHOLDER_KEY}</code>.
            Demo keys are never exposed in client snippets.
          </span>
        </div>
        <span className="hidden shrink-0 sm:inline">Reflects the current search options</span>
      </div>
    </div>
  );
}
