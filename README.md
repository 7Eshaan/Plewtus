<div align="center">

# Plewtus

### Federated Metasearch & Extraction Engine for AI Agents

**Zero search API keys required.** Query multiple search engines concurrently, deduplicate URLs, rank results with Reciprocal Rank Fusion (RRF), safely extract full page content into LLM-ready Markdown, and connect directly to AI agents via the Model Context Protocol (MCP).

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.27.3-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="Next.js" src="https://img.shields.io/badge/Next.js-16-black?style=flat-square&logo=next.js">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black">
  <img alt="Protocol" src="https://img.shields.io/badge/MCP-Protocol-8A2BE2?style=flat-square">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white">
  <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square">
</p>

[Quickstart](#quickstart) • [Architecture](#architecture) • [MCP Integration](#model-context-protocol-mcp-integration) • [Search & Extraction](#search--page-extraction) • [Configuration](#configuration-reference)

</div>

---

## Key Capabilities

- **Zero Search API Keys Required** — Built-in resilient scrapers for DuckDuckGo, Bing, and Yahoo with automatic query escaping and anti-bot mitigation.
- **Concurrent Search** — Fan-out queries across all providers in parallel (typically 1–2.5 s uncached, milliseconds from cache) with per-engine timeouts, circuit breakers, a cut-off for one hung engine, and graceful fallback.
- **Consensus ranking** — Deduplicates by canonical URL, removes off-topic results (including decoy result pages some engines serve to bots), then ranks with Reciprocal Rank Fusion ($k=60$): agreement between engines decides the order, and query relevance only separates near-ties.
- **On-The-Fly Web Extraction** — Automatically strips boilerplate, adverts, and scripts, converting target web pages into clean Markdown, Highlights, or plain text.
- **Native Model Context Protocol (MCP)** — Seamless stdio MCP integration with ChatGPT Desktop, OpenAI Codex CLI, Claude Desktop, Cursor, and local LLM harnesses (DeepSeek Harness, Ollama).
- **Production-Grade Reliability** — Full SSRF protection (loopback and private IP blocking), thread-safe in-memory TTL caching, token bucket rate limits, and zero external database dependencies.

---

## Architecture

```mermaid
flowchart TD
    subgraph Clients["Clients & AI Agents"]
        Agent["AI Agent (ChatGPT / Codex / Claude / Cursor / DSH)"]
        Playground["Next.js Web Playground (Port 3000)"]
        APIClient["Direct REST Client (cURL / SDK)"]
    end

    subgraph PlewtusEngine["Plewtus Core Engine (Go · Port 8080)"]
        Router{"Protocol Gateway"}
        Cache[("In-Memory TTL Cache")]
        Breakers["Circuit Breakers & Concurrency Limiters"]
        
        subgraph Providers["Concurrent Search Providers"]
            DDG["DuckDuckGo Scraper"]
            Bing["Bing Scraper"]
            Yahoo["Yahoo Scraper"]
        end

        Fusion["URL Deduplication & Reciprocal Rank Fusion (RRF)"]
        
        subgraph Extractor["Safe Content Extractor"]
            SSRF["SSRF & Private IP Guard"]
            Sanitize["HTML Sanitizer & Readability Engine"]
            Formatter["Markdown / Highlights Formatter"]
        end
    end

    Agent -- "MCP (stdio JSON-RPC)" --> Router
    Playground -- "REST Proxy" --> Router
    APIClient -- "POST /search" --> Router

    Router <--> Cache
    Router --> Breakers
    Breakers --> DDG & Bing & Yahoo
    DDG & Bing & Yahoo --> Fusion
    Fusion --> SSRF --> Sanitize --> Formatter
    Formatter --> Router
```

---

## Quickstart

### Option 1: Docker Compose (Recommended)

Run the full stack (Go Search API + Next.js Playground) with a single command:

```bash
# 1. Clone repository
git clone https://github.com/7Eshaan/Plewtus.git
cd Plewtus

# 2. Configure environment
cp .env.example .env

# 3. Start services
docker compose up --build -d
```

| Service | Address | Description |
| :--- | :--- | :--- |
| **Playground UI** | [http://localhost:3000](http://localhost:3000) | Interactive web interface with live JSON inspector |
| **Search API** | [http://localhost:8080](http://localhost:8080) | High-performance Go REST API |
| **Health Check** | [http://localhost:8080/healthz](http://localhost:8080/healthz) | Liveness and readiness probe |

---

### Option 2: Native Binary / Local Development

```bash
# Run backend (reads backend/.env if present; real env vars take precedence)
cd backend
cp .env.example .env         # defines the demo API key the frontend uses
go run ./cmd/plewtus serve   # or just: go run ./cmd/plewtus

# In another terminal, run frontend
cd frontend
cp .env.example .env.local   # points at http://localhost:8080 with the same demo key
npm install
npm run dev
```

The frontend authenticates to the backend with `PLEWTUS_API_KEY`, which must appear in the backend's `PLEWTUS_API_KEYS`; the two example files are already paired.

---

## Model Context Protocol (MCP) Integration

Plewtus includes a native [Model Context Protocol](https://modelcontextprotocol.io/) stdio server, turning Plewtus into a private real-time search engine for any LLM.

### Compile the MCP Binary
```bash
cd backend
go build -o bin/plewtus-mcp ./cmd/plewtus-mcp
```

Rebuild after pulling changes: MCP clients launch this binary, not the source.

### 1. ChatGPT Desktop (GUI)
Under **Settings → MCP Servers → Connect to a custom MCP**:
- **Name:** `plewtus`
- **Type:** `STDIO`
- **Command to launch:** `/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp`
- **Working directory:** `/absolute/path/to/PlewtusV1`

### 2. OpenAI Codex / ChatGPT CLI (`~/.codex/config.toml`)
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

### 3. Claude Desktop
Add to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "plewtus": {
      "command": "/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp",
      "env": {
        "PLEWTUS_CACHE_ENABLED": "true",
        "PLEWTUS_CACHE_TTL_SECONDS": "600"
      }
    }
  }
}
```

### 4. Cursor / VS Code
Add to your workspace `.cursor/mcp.json` or MCP Extension config:
```json
{
  "mcpServers": {
    "plewtus": {
      "command": "/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp"
    }
  }
}
```

### 5. DeepSeek Harness (`cordis.patch.yml`)
```yaml
- id: mcp-plewtus
  name: "@deepseek-ai/dsh-mcp-client"
  config:
    serverName: plewtus
    transport: stdio
    command: "/absolute/path/to/PlewtusV1/backend/bin/plewtus-mcp"
```

### Available MCP Tools
- **`search`**: Multi-engine search with provider selection, freshness and domain filters, and optional extraction of the top pages (`markdown`, `highlights`, `text`, `metadata`). Returns a compact text listing by default, or full JSON with `format: "json"`.
- **`extract`**: SSRF-safe content extraction from up to 10 URLs, with a per-URL error when a page can't be read.

See [backend/docs/MCP.md](backend/docs/MCP.md) for all parameters.

---

## Search & Page Extraction

Execute a search with automated Markdown extraction:

```bash
curl -sS http://localhost:8080/search \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $PLEWTUS_API_KEY" \
  -d '{
    "query": "golang concurrency patterns",
    "limit": 3,
    "extract": {
      "enabled": true,
      "mode": "markdown",
      "max_pages": 2
    }
  }'
```

### Response Preview

```json
{
  "query": "golang concurrency patterns",
  "results": [
    {
      "id": "res_1",
      "title": "Go Concurrency Patterns: Pipelines and cancellation",
      "url": "https://go.dev/blog/pipelines",
      "snippet": "Go's concurrency primitives make it easy to construct streaming data pipelines...",
      "score": 0.032787,
      "ranking_score": 0.6667,
      "engines": [
        {"name": "duckduckgo", "position": 1},
        {"name": "yahoo", "position": 1}
      ],
      "content": {
        "markdown": "# Go Concurrency Patterns: Pipelines\n\nGo provides primitives that facilitate building concurrent pipelines...",
        "author": "Sameer Ajmani",
        "extracted_at": "2026-11-06T16:30:14Z"
      }
    }
  ],
  "metadata": {
    "took_ms": 1820,
    "providers": {
      "queried": ["duckduckgo", "bing", "yahoo"],
      "successful": ["bing", "duckduckgo", "yahoo"]
    }
  }
}
```

### Extraction Modes

| Mode | Output Format | Recommended Use Case |
| :--- | :--- | :--- |
| `markdown` | Clean GitHub Flavored Markdown | Deep synthesis and technical documentation |
| `highlights` | High-signal excerpt sentences | Fast fact verification and lightweight context |
| `text` | Cleaned plain readable text | General classification or summarization |
| `metadata` | Title, description, author, publication date | Source vetting and cataloging |

---

## Configuration Reference

Plewtus is configured via environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PLEWTUS_PORT` (or `PORT`) | `8080` | HTTP listening port for the REST API |
| `PLEWTUS_AUTH_REQUIRED` | `false` | When `true`, rejects requests lacking a valid API key |
| `PLEWTUS_API_KEYS` | `""` | Comma-delimited list of authorized keys with tiers (e.g. `key1:free,key2:pro`) |
| `PLEWTUS_DEFAULT_PROVIDERS` | `duckduckgo,bing,yahoo` | Default providers queried per search (`marginalia` is also available) |
| `PLEWTUS_MARGINALIA_API_KEY` | `public` | Marginalia Search API key; `public` is shared and heavily rate limited |
| `PLEWTUS_DEFAULT_LIMIT` | `10` | Results returned when `limit` is omitted |
| `PLEWTUS_CACHE_ENABLED` | `true` | Enables the in-memory LRU search cache |
| `PLEWTUS_CACHE_TTL_SECONDS` | `600` | Search result cache expiration in seconds |
| `PLEWTUS_DEFAULT_TIMEOUT_MS` | `6000` | Max duration per upstream search provider request |
| `PLEWTUS_EXTRACT_TIMEOUT_MS` | `8000` | Time budget for page extraction (each page is capped at 5 s) |
| `PLEWTUS_STRAGGLER_GRACE_MS` | `1500` | How long the last engine may take once the others have answered (`0` disables) |
| `PLEWTUS_FREE_TIER_RATE_LIMIT` | `10` | Free tier requests allowed per minute |
| `PLEWTUS_FREE_TIER_DAILY_QUOTA` | `100` | Free tier requests allowed per UTC day |

The full list is in [backend/docs/CONFIGURATION.md](backend/docs/CONFIGURATION.md).

---

## Repository Structure

```text
PlewtusV1/
├── backend/
│   ├── cmd/
│   │   ├── plewtus/         # Unified server & CLI binary
│   │   └── plewtus-mcp/     # Dedicated stdio MCP server binary
│   ├── internal/
│   │   ├── app/             # Wiring shared by the HTTP and MCP servers
│   │   ├── cache/           # Thread-safe in-memory LRU cache with TTL
│   │   ├── config/          # Environment configuration
│   │   ├── extractor/       # SSRF-safe fetcher, HTML cleaner & Markdown converter
│   │   ├── mcp/             # Model Context Protocol JSON-RPC 2.0 server
│   │   ├── model/           # Request and response types
│   │   ├── orchestrator/    # Concurrent provider dispatch and response assembly
│   │   ├── pipeline/        # URL normalization, deduplication, relevance, RRF ranking
│   │   ├── provider/        # DuckDuckGo, Bing, Yahoo scrapers + Marginalia API client
│   │   ├── reliability/     # Circuit breakers
│   │   ├── server/          # HTTP routes and middleware
│   │   └── usage/           # API keys, rate limits, daily quotas
│   └── docs/                # OpenAPI specification and API manuals
├── frontend/
│   ├── src/app/             # Next.js 16 App Router UI & API proxy
│   └── src/components/      # Playground, filters, and JSON viewer
└── docker-compose.yml       # Runs the API and the playground together
```

---

## License

Distributed under the Apache License 2.0. See [LICENSE](LICENSE) for more information.
