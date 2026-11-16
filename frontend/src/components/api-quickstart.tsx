"use client";

import { useState } from "react";
import { CodePanel, type CodeLanguage } from "./ui/code-panel";
import { SectionHeader } from "./ui/section-header";

type Example = "curl" | "typescript" | "python" | "go" | "response";

const EXAMPLES: Record<Example, { label: string; language: CodeLanguage; code: string }> = {
  curl: {
    label: "cURL",
    language: "bash",
    code: `curl -X POST http://localhost:8080/search \\
  -H "Content-Type: application/json" \\
  -d '{
    "query": "raft consensus",
    "limit": 5,
    "extract": {
      "enabled": true,
      "mode": "markdown"
    }
  }'`,
  },
  typescript: {
    label: "TypeScript",
    language: "typescript",
    code: `const res = await fetch("http://localhost:8080/search", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ query: "raft consensus", limit: 5 }),
});

const { results } = await res.json();`,
  },
  python: {
    label: "Python",
    language: "python",
    code: `import requests

res = requests.post(
    "http://localhost:8080/search",
    json={"query": "raft consensus", "limit": 5},
)
results = res.json()["results"]`,
  },
  go: {
    label: "Go",
    language: "go",
    code: `body := strings.NewReader(\`{"query": "raft consensus", "limit": 5}\`)

res, err := http.Post("http://localhost:8080/search", "application/json", body)
if err != nil {
	log.Fatal(err)
}
defer res.Body.Close()`,
  },
  response: {
    label: "Response",
    language: "json",
    code: `{
  "query": "raft consensus",
  "results": [
    {
      "rank": 1,
      "title": "The Raft Consensus Algorithm",
      "url": "https://raft.github.io/",
      "snippet": "Raft is a consensus algorithm that is designed to be easy to understand...",
      "ranking_score": 0.99
    }
  ],
  "metadata": { "took_ms": 1420 }
}`,
  },
};

const ORDER: Example[] = ["curl", "typescript", "python", "go", "response"];

const POINTS = [
  "Works with any language that can send JSON",
  "No API key needed when self-hosted",
  "Add extract to get each page as Markdown",
];

export function ApiQuickstart() {
  const [active, setActive] = useState<Example>("curl");
  const example = EXAMPLES[active];

  return (
    <section id="quickstart" className="w-full min-h-[calc(100vh-3.5rem)] min-h-[calc(100dvh-3.5rem)] py-20 flex flex-col justify-center scroll-mt-14">
      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] gap-14 lg:gap-16 items-center">
        <div className="flex flex-col gap-12">
          <SectionHeader eyebrow="HTTP API" title="One request, every engine">
            Send a query to <span className="font-mono text-[0.9em] text-zinc-900 dark:text-zinc-200">POST /search</span>{" "}
            and get back one ranked list.
          </SectionHeader>

          <ul className="flex flex-col gap-4">
            {POINTS.map((point) => (
              <li key={point} className="flex items-center gap-3.5 text-[17px] text-zinc-600 dark:text-zinc-400">
                <span aria-hidden="true" className="w-2 h-2 rounded-full bg-brand/70 shrink-0" />
                {point}
              </li>
            ))}
          </ul>
        </div>

        <CodePanel
          code={example.code}
          language={example.language}
          tabs={ORDER.map((value) => ({ value, label: EXAMPLES[value].label }))}
          activeTab={active}
          onTabChange={setActive}
          tabsLabel="Example"
          copyLabel="Copy example"
        />
      </div>
    </section>
  );
}
