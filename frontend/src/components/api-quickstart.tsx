"use client";

import { useState } from "react";
import { Code2, Globe, Cpu, ShieldCheck, Sparkles, FileJson } from "lucide-react";
import { CopyButton } from "./ui/copy-button";
import { SegmentedControl } from "./ui/segmented-control";

type CodeLang = "curl" | "typescript" | "python" | "go";

const CODE_EXAMPLES: Record<CodeLang, { label: string; code: string }> = {
  curl: {
    label: "cURL",
    code: `curl -X POST http://localhost:8080/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "query": "distributed consensus raft",
    "limit": 10,
    "providers": ["duckduckgo", "bing", "yahoo"],
    "freshness": "month",
    "extract": {
      "enabled": true,
      "mode": "markdown"
    }
  }'`,
  },
  typescript: {
    label: "TypeScript",
    code: `import type { components } from "./types/api";

type SearchRequest = components["schemas"]["SearchRequest"];
type SearchResponse = components["schemas"]["SearchResponse"];

const payload: SearchRequest = {
  query: "distributed consensus raft",
  limit: 10,
  providers: ["duckduckgo", "bing", "yahoo"],
  extract: {
    enabled: true,
    mode: "markdown",
  },
};

const res = await fetch("http://localhost:8080/search", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY",
  },
  body: JSON.stringify(payload),
});

const data: SearchResponse = await res.json();
console.log(\`Aggregated \${data.results.length} results in \${data.metadata.took_ms}ms\`);
for (const item of data.results) {
  console.log(\`#\${item.rank} [\${item.ranking_score.toFixed(2)}] \${item.title} -> \${item.url}\`);
}`,
  },
  python: {
    label: "Python",
    code: `import requests

url = "http://localhost:8080/search"
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer YOUR_API_KEY",
}
payload = {
    "query": "distributed consensus raft",
    "limit": 10,
    "providers": ["duckduckgo", "bing", "yahoo"],
    "freshness": "month",
    "extract": {
        "enabled": True,
        "mode": "markdown",
    },
}

response = requests.post(url, headers=headers, json=payload)
data = response.json()

print(f"Aggregated {len(data['results'])} results in {data['metadata']['took_ms']}ms:")
for r in data["results"]:
    print(f"#{r['rank']} [{r['ranking_score']:.2f}] {r['title']} -> {r['url']}")`,
  },
  go: {
    label: "Go",
    code: `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	payload := map[string]any{
		"query":     "distributed consensus raft",
		"limit":     10,
		"providers": []string{"duckduckgo", "bing", "yahoo"},
		"extract": map[string]any{
			"enabled": true,
			"mode":    "markdown",
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "http://localhost:8080/search", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer YOUR_API_KEY")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Search query: %v\\n", result["query"])
}`,
  },
};

const SAMPLE_RESPONSE = `{
  "query": "distributed consensus raft",
  "results": [
    {
      "id": "res_1",
      "rank": 1,
      "score": 0.048918,
      "relevance_score": 1,
      "ranking_score": 0.994667,
      "title": "In Search of an Understandable Consensus Algorithm",
      "url": "https://raft.github.io/",
      "snippet": "Raft is a consensus algorithm for managing a replicated log...",
      "engines": [
        { "name": "duckduckgo", "position": 1 },
        { "name": "bing", "position": 1 },
        { "name": "yahoo", "position": 2 }
      ],
      "published_date": "2024-03-15",
      "content": {
        "title": "Raft Consensus Algorithm",
        "markdown": "# Raft\\n\\nRaft is a consensus algorithm...",
        "extracted_at": "2026-11-05T09:00:00Z"
      }
    }
  ],
  "metadata": {
    "total_results": 1,
    "took_ms": 1420,
    "providers": {
      "queried": ["duckduckgo", "bing", "yahoo"],
      "successful": ["bing", "duckduckgo", "yahoo"],
      "failed": [],
      "timing_ms": { "bing": 410, "duckduckgo": 780, "yahoo": 1390 }
    },
    "stats": {
      "raw_count": 3,
      "deduped_count": 1,
      "relevance_rejected_count": 0,
      "domain_rejected_count": 0,
      "invalid_url_count": 0,
      "duplicate_count": 2,
      "provider_discarded_count": 0
    },
    "timestamp": "2026-11-05T09:00:00Z"
  }
}`;

export function ApiQuickstart() {
  const [activeLang, setActiveLang] = useState<CodeLang>("curl");
  const [activeTab, setActiveTab] = useState<"request" | "response">("request");

  return (
    <section id="quickstart" className="w-full pt-8 pb-4 flex flex-col gap-6 scroll-mt-16">
      {/* Header */}
      <div className="flex flex-col gap-1.5 text-center sm:text-left">
        <div className="inline-flex items-center gap-1.5 text-xs font-mono text-brand dark:text-brand-light uppercase tracking-wider font-semibold">
          <span>API Quickstart &amp; Integration</span>
        </div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans">
          Direct HTTP &amp; Code Integration
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400 max-w-2xl leading-relaxed">
          Plewtus exposes a unified JSON endpoint. Send a single POST request to query DuckDuckGo, Bing, and Yahoo simultaneously, with canonical deduplication, consensus ranking, and optional content extraction.
        </p>
      </div>

      {/* Feature Highlights Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs font-mono">
        <div className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-1 shadow-2xs">
          <div className="flex items-center gap-1.5 text-zinc-900 dark:text-zinc-100 font-semibold">
            <Globe className="w-3.5 h-3.5 text-brand dark:text-brand-light" />
            <span>POST /search</span>
          </div>
          <span className="text-[11px] text-zinc-500">Standard REST API</span>
        </div>

        <div className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-1 shadow-2xs">
          <div className="flex items-center gap-1.5 text-zinc-900 dark:text-zinc-100 font-semibold">
            <Cpu className="w-3.5 h-3.5 text-brand dark:text-brand-light" />
            <span>Parallel Dispatch</span>
          </div>
          <span className="text-[11px] text-zinc-500">DDG, Bing &amp; Yahoo</span>
        </div>

        <div className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-1 shadow-2xs">
          <div className="flex items-center gap-1.5 text-zinc-900 dark:text-zinc-100 font-semibold">
            <Sparkles className="w-3.5 h-3.5 text-brand dark:text-brand-light" />
            <span>Consensus ranking</span>
          </div>
          <span className="text-[11px] text-zinc-500">RRF with a small relevance boost</span>
        </div>

        <div className="p-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-1 shadow-2xs">
          <div className="flex items-center gap-1.5 text-zinc-900 dark:text-zinc-100 font-semibold">
            <ShieldCheck className="w-3.5 h-3.5 text-brand dark:text-brand-light" />
            <span>SSRF Protection</span>
          </div>
          <span className="text-[11px] text-zinc-500">Safe page extraction</span>
        </div>
      </div>

      {/* Code Editor & Response Showcase */}
      <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] overflow-hidden shadow-2xs flex flex-col">
        {/* Editor Toolbar */}
        <div className="px-4 py-3 border-b border-zinc-200 dark:border-zinc-800 flex flex-wrap items-center justify-between gap-3 bg-zinc-50/60 dark:bg-zinc-900/40">
          <SegmentedControl
            ariaLabel="Example"
            value={activeTab}
            onChange={setActiveTab}
            options={[
              {
                value: "request",
                label: (
                  <>
                    <Code2 className="w-3.5 h-3.5 text-brand dark:text-brand-light" />
                    API Request
                  </>
                ),
              },
              {
                value: "response",
                label: (
                  <>
                    <FileJson className="w-3.5 h-3.5 text-emerald-500" />
                    Response JSON
                  </>
                ),
              },
            ]}
          />

          <div className="flex flex-wrap items-center gap-2">
            {activeTab === "request" && (
              <SegmentedControl
                ariaLabel="Language"
                value={activeLang}
                onChange={setActiveLang}
                options={(Object.keys(CODE_EXAMPLES) as CodeLang[]).map((lang) => ({
                  value: lang,
                  label: CODE_EXAMPLES[lang].label,
                }))}
              />
            )}
            <CopyButton
              getText={() => (activeTab === "request" ? CODE_EXAMPLES[activeLang].code : SAMPLE_RESPONSE)}
              ariaLabel="Copy snippet to clipboard"
            />
          </div>
        </div>

        {/* Code Box */}
        <div className="p-4 bg-zinc-950 text-zinc-200 overflow-x-auto text-xs font-mono leading-relaxed min-h-[260px]">
          <pre className="selection:bg-brand/30">
            {activeTab === "request" ? CODE_EXAMPLES[activeLang].code : SAMPLE_RESPONSE}
          </pre>
        </div>

        {/* Specs Footer */}
        <div className="px-4 py-2.5 bg-zinc-50/40 dark:bg-zinc-900/20 border-t border-zinc-200 dark:border-zinc-800/80 flex flex-wrap items-center justify-between gap-3 text-[11px] font-mono text-zinc-500">
          <div className="flex items-center gap-2">
            <span className="text-zinc-400 dark:text-zinc-500">Endpoint:</span>
            <code className="text-zinc-800 dark:text-zinc-200 font-semibold">POST http://localhost:8080/search</code>
          </div>
          <div className="flex items-center gap-3">
            <span>Auth: Bearer &lt;key&gt; (Optional)</span>
            <span>•</span>
            <span>Content: application/json</span>
          </div>
        </div>
      </div>
    </section>
  );
}
