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

**Protocol details:** the server negotiates MCP revisions `2025-06-18` (latest), `2025-03-26`, and `2024-11-05`. Tool calls run concurrently (up to 4 at a time), honor `notifications/cancelled`, and the process exits promptly on `SIGINT`/`SIGTERM` or when stdin closes. JSON-RPC batches (an array of messages on one line) are answered with one array of responses.

> [!IMPORTANT]
> MCP clients launch the compiled binary, not the source. After changing or pulling code, rebuild it: `go build -o bin/plewtus-mcp ./cmd/plewtus-mcp && go build -o bin/plewtus ./cmd/plewtus` (from `backend/`).

---

## Configuration for AI Clients

### 1. OpenAI Codex / ChatGPT CLI (`~/.codex/config.toml`)

Add to `~/.codex/config.toml`:

```toml
[mcp_servers.plewtus]
enabled = true
command = "/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp"
cwd = "/absolute/path/to/PlewtusV1"

[mcp_servers.plewtus.env]
PLEWTUS_CACHE_ENABLED = "true"
PLEWTUS_CACHE_TTL_SECONDS = "600"
```

### 2. ChatGPT Desktop (GUI)

Navigate to **Settings → MCP Servers → Connect to a custom MCP**:
- **Name:** `plewtus`
- **Type:** `STDIO`
- **Command to launch:** `/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp`
- **Working directory:** `/absolute/path/to/PlewtusV1`

### 3. Claude Desktop

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

### 4. Cursor / VS Code MCP Extensions

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
Searches DuckDuckGo, Bing, and Yahoo, removes duplicate and off-topic results, and ranks the rest by agreement between engines. Can also extract content from the top results.

**Parameters:**
- `query` (`string`, **required**): The search query.
- `limit` (`integer`, optional): Results to return (default `PLEWTUS_DEFAULT_LIMIT`, 10; max 50). Windows deeper than 10 make each engine fetch more pages, which is slower and best-effort.
- `offset` (`integer`, optional): Pagination offset (default 0); `offset + limit` cannot exceed 50.
- `providers` (`array<string>`, optional): Any of `duckduckgo`, `bing`, `yahoo`, `marginalia` (default: `PLEWTUS_DEFAULT_PROVIDERS`, i.e. the first three). Marginalia is an independent small-web index queried through its official API.
- `freshness` (`string`, optional): Recency filter (`"day"`, `"week"`, `"month"`, `"year"`).
- `include_domains` / `exclude_domains` (`array<string>`, optional): Domain allow/deny lists (subdomains included).
- `country` / `language` (`string`, optional): 2-letter ISO codes (e.g. `"us"`, `"en"`).
- `extract_mode` (`string`, optional): Extract page content as `"markdown"`, `"highlights"`, `"text"`, or `"metadata"`.
- `max_pages` (`integer`, optional): Number of top results to extract (default 3, max 10).
- `max_chars_per_page` (`integer`, optional): Characters per extracted page (default 4000, max 20000).
- `format` (`string`, optional): `"text"` (default) for a compact, numbered listing that is cheap in tokens, or `"json"` for the full REST API response.

### 2. `extract`
Fetches specific URLs and returns clean content, with SSRF protection and size limits. Each URL is reported separately; failures (blocked address, non-HTML content, HTTP errors) include the reason.

**Parameters:**
- `urls` (`array<string>`, **required**): HTTP/HTTPS URLs to fetch (maximum 10 per call).
- `mode` (`string`, optional): `"markdown"` (default), `"highlights"`, `"text"`, or `"metadata"`.
- `query` (`string`, optional): Focus for `"highlights"` mode.
- `max_chars` (`integer`, optional): Character limit per page (default 4000, maximum 20000).

Both tools are annotated `readOnlyHint: true` and `openWorldHint: true`.

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

Response emitted by Plewtus on `stdout` (text is abbreviated):
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Search results for \"go net/http middleware patterns\": 3 result(s) in 2140 ms · engines: bing, duckduckgo, yahoo\n\n1. Writing Middleware in Go\n   https://example.com/go-middleware\n   Found by: duckduckgo#1, yahoo#2\n   Middleware wraps an http.Handler...\n   Content (markdown):\n      # Writing Middleware in Go ..."
      }
    ]
  }
}
```
