"use client";

import { useState } from "react";
import { Copy, Check, Wrench, ShieldCheck, Zap } from "lucide-react";

type McpClientTab = "claude" | "cursor" | "cli";

export function McpSection() {
  const [activeTab, setActiveTab] = useState<McpClientTab>("claude");
  const [copied, setCopied] = useState(false);

  const CLAUDE_CONFIG = `{
  "mcpServers": {
    "plewtus": {
      "command": "/path/to/PlewtusV1/backend/bin/plewtus",
      "args": ["mcp"],
      "env": {
        "PLEWTUS_CACHE_ENABLED": "true",
        "PLEWTUS_CACHE_TTL_SECONDS": "600"
      }
    }
  }
}`;

  const CURSOR_CONFIG = `{
  "mcpServers": {
    "plewtus": {
      "command": "/path/to/PlewtusV1/backend/bin/plewtus-mcp"
    }
  }
}`;

  const CLI_EXAMPLE = `# Launch stdio server directly
./backend/bin/plewtus mcp

# Or send a JSON-RPC 2.0 tool invocation over stdin:
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"golang concurrency patterns","limit":3}}}' | ./backend/bin/plewtus mcp`;

  function getCurrentSnippet(): string {
    switch (activeTab) {
      case "claude":
        return CLAUDE_CONFIG;
      case "cursor":
        return CURSOR_CONFIG;
      case "cli":
        return CLI_EXAMPLE;
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(getCurrentSnippet());
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback
    }
  }

  return (
    <section id="mcp" className="w-full pt-8 pb-4 flex flex-col gap-6 scroll-mt-16">
      <div className="flex flex-col gap-1.5 text-center sm:text-left">
        <div className="inline-flex items-center gap-1.5 text-xs font-mono text-brand dark:text-brand-light uppercase tracking-wider font-semibold">
          <span>AI Agent Interoperability</span>
        </div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans">
          Native Model Context Protocol (MCP) Support
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400 max-w-2xl leading-relaxed">
          Plewtus includes a built-in stdio Model Context Protocol (MCP) server. Any MCP-compatible host—Claude Desktop, Cursor, Continue, or custom autonomous agent frameworks—can query the web directly via standard JSON-RPC 2.0.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-5 items-start">
        {/* Left column: Tool specifications */}
        <div className="lg:col-span-5 flex flex-col gap-3.5">
          {/* Tool 1: search */}
          <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-2 shadow-2xs">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Wrench className="w-3.5 h-3.5 text-emerald-500" />
                <code className="font-mono text-xs font-bold text-zinc-900 dark:text-zinc-100">
                  search
                </code>
              </div>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700/60">
                Tool
              </span>
            </div>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Searches DuckDuckGo, Bing, and Yahoo, filters weak query matches, combines relevance with engine consensus, and can extract page content.
            </p>
            <div className="pt-2 border-t border-zinc-100 dark:border-zinc-800/80 flex flex-wrap gap-1 text-[10px] font-mono text-zinc-500">
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">query (required)</span>
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">limit</span>
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">freshness</span>
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">extract_mode</span>
            </div>
          </div>

          {/* Tool 2: extract */}
          <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-2 shadow-2xs">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-3.5 h-3.5 text-brand" />
                <code className="font-mono text-xs font-bold text-zinc-900 dark:text-zinc-100">
                  extract
                </code>
              </div>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700/60">
                Tool
              </span>
            </div>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Fetches and converts webpage content into clean Markdown, plain text, or salient highlights with multi-layer SSRF guards (loopback, private RFC 1918, CGNAT rejection) and 2MB resource bounds.
            </p>
            <div className="pt-2 border-t border-zinc-100 dark:border-zinc-800/80 flex flex-wrap gap-1 text-[10px] font-mono text-zinc-500">
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">urls (required)</span>
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">mode</span>
              <span className="px-1.5 py-0.5 rounded bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">max_chars</span>
            </div>
          </div>
        </div>

        {/* Right column: Host Configuration Snippet */}
        <div className="lg:col-span-7 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] overflow-hidden shadow-2xs flex flex-col">
          <div className="px-4 py-3 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between bg-zinc-50/60 dark:bg-zinc-900/40">
            {/* Host Tabs */}
            <div className="inline-flex items-center p-0.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 font-mono text-xs">
              <button
                type="button"
                onClick={() => setActiveTab("claude")}
                className={`px-2.5 py-1 rounded-md transition-all cursor-pointer ${
                  activeTab === "claude"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                }`}
              >
                Claude Desktop
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("cursor")}
                className={`px-2.5 py-1 rounded-md transition-all cursor-pointer ${
                  activeTab === "cursor"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                }`}
              >
                Cursor / VS Code
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("cli")}
                className={`px-2.5 py-1 rounded-md transition-all cursor-pointer ${
                  activeTab === "cli"
                    ? "bg-white dark:bg-zinc-800 text-zinc-950 dark:text-zinc-50 font-medium shadow-2xs"
                    : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100"
                }`}
              >
                CLI Stdio
              </button>
            </div>

            {/* Copy Button */}
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 hover:border-zinc-400 dark:hover:border-zinc-500 text-xs font-mono transition-colors cursor-pointer"
              aria-label="Copy MCP config to clipboard"
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

          <div className="p-4 bg-zinc-950 text-zinc-200 overflow-x-auto text-xs font-mono leading-relaxed min-h-[220px]">
            <pre className="selection:bg-zinc-800">{getCurrentSnippet()}</pre>
          </div>

          <div className="px-4 py-2.5 border-t border-zinc-200 dark:border-zinc-800/80 bg-zinc-50/50 dark:bg-zinc-900/30 flex items-center justify-between text-[11px] font-mono text-zinc-500 dark:text-zinc-400">
            <span className="flex items-center gap-1.5">
              <Zap className="w-3 h-3 text-emerald-500" />
              <span>Strict stdio stream isolation: JSON-RPC on stdout, diagnostics on stderr</span>
            </span>
          </div>
        </div>
      </div>
    </section>
  );
}
