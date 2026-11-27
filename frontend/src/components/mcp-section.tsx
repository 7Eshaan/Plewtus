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
    label: "DeepSeek Harness",
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
    <section
      id="mcp"
      className="flex min-h-[calc(100vh-4rem)] min-h-[calc(100dvh-4rem)] w-full scroll-mt-16 flex-col justify-center py-20"
    >
      <div className="grid grid-cols-1 items-center gap-12 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)] lg:gap-16">
        <div className="flex flex-col gap-10">
          <SectionHeader eyebrow="MCP server" title="Give your agent the web">
            Add Plewtus to Claude, Cursor, Codex, DeepSeek Harness or any MCP client. Your agent gets two tools.
          </SectionHeader>

          <ul className="flex flex-col gap-3">
            {TOOLS.map((tool) => (
              <li key={tool.name} className="flex flex-col gap-1.5 rounded-corner-lg bg-surface-container-low/80 p-5 backdrop-blur-sm">
                <code className="font-mono text-title-md text-primary">{tool.name}</code>
                <span className="text-body-lg text-on-surface-variant">{tool.body}</span>
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
