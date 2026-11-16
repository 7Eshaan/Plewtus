"use client";

import { useState } from "react";
import { CodePanel, type CodeLanguage } from "./ui/code-panel";
import { SectionHeader } from "./ui/section-header";

type McpClientTab = "claude" | "cursor" | "codex" | "deepseek" | "cli";

const CLIENT_SNIPPETS: Record<McpClientTab, { label: string; language: CodeLanguage; code: string }> = {
  claude: {
    label: "Claude",
    language: "json",
    code: `{
  "mcpServers": {
    "plewtus": {
      "command": "/path/to/backend/bin/plewtus",
      "args": ["mcp"]
    }
  }
}`,
  },
  cursor: {
    label: "Cursor",
    language: "json",
    code: `{
  "mcpServers": {
    "plewtus": {
      "command": "/path/to/backend/bin/plewtus-mcp"
    }
  }
}`,
  },
  codex: {
    label: "Codex",
    language: "toml",
    code: `# ~/.codex/config.toml
[mcp_servers.plewtus]
command = "/path/to/backend/bin/plewtus-mcp"`,
  },
  deepseek: {
    label: "DeepSeek",
    language: "yaml",
    code: `# ~/.dsh/profiles/<profile>/cordis.patch.yml
- insert:
    - id: mcp-plewtus
      name: "@deepseek-ai/dsh-mcp-client"
      config:
        serverName: plewtus
        transport: stdio
        command: "/path/to/backend/bin/plewtus-mcp"`,
  },
  cli: {
    label: "Terminal",
    language: "bash",
    code: `# Build, then start the stdio server
go build -o bin/ ./cmd/...
./bin/plewtus mcp`,
  },
};

const TAB_ORDER: McpClientTab[] = ["claude", "cursor", "codex", "deepseek", "cli"];

const TOOLS = [
  { name: "search", body: "Search the web and get ranked, deduplicated results." },
  { name: "extract", body: "Read any page or PDF as clean Markdown." },
];

export function McpSection() {
  const [activeTab, setActiveTab] = useState<McpClientTab>("claude");
  const snippet = CLIENT_SNIPPETS[activeTab];

  return (
    <section id="mcp" className="w-full min-h-[calc(100vh-3.5rem)] min-h-[calc(100dvh-3.5rem)] py-20 flex flex-col justify-center scroll-mt-14">
      <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] gap-14 lg:gap-16 items-center">
        <div className="flex flex-col gap-12">
          <SectionHeader eyebrow="MCP server" title="Give your agent the web">
            Add Plewtus to Claude, Cursor, Codex, DeepSeek Harness or any MCP client. Your agent gets two tools.
          </SectionHeader>

          <ul className="flex flex-col gap-7">
            {TOOLS.map((tool) => (
              <li key={tool.name} className="flex flex-col gap-1.5 pl-5 border-l-2 border-brand/50">
                <span className="font-mono text-base font-medium text-zinc-950 dark:text-zinc-50">{tool.name}</span>
                <span className="text-[17px] text-zinc-600 dark:text-zinc-400">{tool.body}</span>
              </li>
            ))}
          </ul>
        </div>

        <CodePanel
          code={snippet.code}
          language={snippet.language}
          tabs={TAB_ORDER.map((value) => ({ value, label: CLIENT_SNIPPETS[value].label }))}
          activeTab={activeTab}
          onTabChange={setActiveTab}
          tabsLabel="MCP client"
          copyLabel="Copy MCP config to clipboard"
        />
      </div>
    </section>
  );
}
