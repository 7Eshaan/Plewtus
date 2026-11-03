# Model Context Protocol (MCP) Guide

Plewtus includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server. This enables AI agent environments (such as Claude Desktop, Cursor, Continue, Antigravity, and custom LLM agent harnesses) to use Plewtus's search and extraction capabilities directly over `stdio`.

---

## Running the MCP Server

Plewtus can be launched as an MCP stdio server in two equivalent ways:

1. **Subcommand on primary binary:**
   ```bash
   ./bin/plewtus mcp
   ```

2. **Dedicated MCP binary:**
   ```bash
   ./bin/plewtus-mcp
   ```

> [!NOTE]
> All diagnostic logs in MCP mode are automatically routed to `stderr`. The `stdout` stream is strictly reserved for clean JSON-RPC 2.0 messages.

---

## Configuration for AI Clients

### 1. Claude Desktop

Add Plewtus to your `claude_desktop_config.json`:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "plewtus": {
      "command": "/absolute/path/to/PlewtusV1/backend/bin/plewtus",
      "args": ["mcp"],
      "env": {
        "PLEWTUS_CACHE_ENABLED": "true",
        "PLEWTUS_CACHE_TTL_SECONDS": "600"
      }
    }
  }
}
```

### 2. Cursor / VS Code MCP Extensions

Add to your MCP settings:

```json
{
  "mcpServers": {
    "plewtus": {
      "command": "/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp"
    }
  }
}
```

---

## Available MCP Tools

### 1. `search`
Search the web across multiple search engines with automatic consensus deduplication, Reciprocal Rank Fusion (RRF), and optional content extraction.

**Parameters:**
- `query` (`string`, **required**): The search query.
- `limit` (`integer`, optional): Max results to return (default 10, max 50).
- `offset` (`integer`, optional): Pagination offset (default 0).
- `freshness` (`string`, optional): Recency filter (`"day"`, `"week"`, `"month"`, `"year"`).
- `include_domains` (`array<string>`, optional): Whitelist of hostnames/domains.
- `exclude_domains` (`array<string>`, optional): Blacklist of hostnames/domains.
- `country` (`string`, optional): 2-letter ISO country code (e.g. `"us"`).
- `language` (`string`, optional): 2-letter ISO language code (e.g. `"en"`).
- `extract_mode` (`string`, optional): Webpage extraction mode (`"markdown"`, `"highlights"`, `"text"`, `"metadata"`).

### 2. `extract`
Extract clean text, Markdown, or highlights from one or more target URLs with SSRF protection and resource capping.

**Parameters:**
- `urls` (`array<string>`, **required**): Array of HTTP/HTTPS URLs to scrape and extract.
- `mode` (`string`, optional): Extraction format: `"markdown"`, `"highlights"`, `"text"`, or `"metadata"` (default `"markdown"`).
- `max_chars` (`integer`, optional): Character limit per extracted page (default 4000).

---

## Example JSON-RPC Interaction

Request sent by AI client on `stdin`:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "search",
    "arguments": {
      "query": "go net/http middleware patterns",
      "limit": 3,
      "extract_mode": "markdown"
    }
  }
}
```

Response emitted by Plewtus on `stdout`:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\n  \"query\": \"go net/http middleware patterns\",\n  \"results\": [...],\n  \"metadata\": {...}\n}"
      }
    ]
  }
}
```
