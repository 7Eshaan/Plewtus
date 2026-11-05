# Plewtus

Plewtus is a lightweight, high-performance metasearch API backend built in Go specifically for AI agents and LLM applications.

Instead of operating an expensive web-scale crawler and search index, Plewtus aggregates search results from multiple external search engines (DuckDuckGo, Bing, Yahoo) in parallel. It unwraps tracking URLs, cleans ads, deduplicates canonical destinations, fuses rankings via **Reciprocal Rank Fusion (RRF)**, and provides optional **SSRF-safe webpage extraction** (clean Markdown, text, and query highlights).

Plewtus is free, cost-efficient, and runs with **zero external dependencies** outside of the Go standard library and `golang.org/x/net` for HTML parsing.

---

## Key Features

- **Multi-Engine Parallel Metasearch:** Queries DuckDuckGo, Bing, and Yahoo concurrently with per-provider bounded timeouts and partial response resilience.
- **Direct Search Engine Redirect Unwrapping:** Decodes base64 click URLs from Bing (`/ck/a?!...&u=a1...`), unescapes Yahoo redirect paths (`/RU=...`), and unwraps DuckDuckGo tokens (`/l/?uddg=...`), protecting agent privacy and skipping redundant HTTP hops.
- **URL Normalization & Tracker Stripping:** Strips advertising tokens (`utm_*`, `gclid`, `fbclid`, `msclkid`, `twclid`, etc.) and canonicalizes URLs across scheme variants, `www.` prefixes, default index files, and trailing slashes.
- **Reciprocal Rank Fusion (RRF):** Merges multi-engine rankings into a unified scale-invariant score using standard RRF smoothing ($k=60$), prioritizing multi-engine consensus over single-engine noise.
- **AI-Friendly Webpage Content Extraction:** Safe DOM cleaning and conversion into clean Markdown, query highlights, or plain text with SSRF defenses (blocking private/loopback/link-local/CGNAT IPs, DNS rebinding, non-HTML MIME types, and oversized responses).
- **In-Memory LRU Response Cache:** Caches queries and extraction outputs with deterministic key hashing and TTL expiration, dramatically cutting latency and upstream requests.
- **Usage Management & Configurable Free Tier:** Token-bucket rate limiting and daily request quotas for both unauthenticated clients (free IP tier) and authenticated API keys.
- **Circuit Breaking & Provider Fallback:** Protects upstream search engines during bot challenges or outages with `Closed`, `Open`, and `Half-Open` state management and automatic fallback to healthy engines.
- **Model Context Protocol (MCP) Server:** Native JSON-RPC 2.0 stdio server ready for direct plug-and-play integration with Claude Desktop, Cursor, and custom agent harnesses.
- **OpenAPI 3.1.0 Specification:** Complete machine-readable API spec included.

---

## Architecture Overview

```
                          AI Agent / HTTP Client / MCP Host
                                        │
                                        ▼
                     ┌─────────────────────────────────────┐
                     │          Server & Middleware        │
                     │  (CORS, Usage Limits, Auth Guard)   │
                     └──────────────────┬──────────────────┘
                                        │
                                        ▼
                     ┌─────────────────────────────────────┐
                     │          In-Memory Cache            │
                     │  (SHA-256 Key Hashing, LRU & TTL)   │
                     └──────────────────┬──────────────────┘
                                        │ (Cache Miss)
                                        ▼
                     ┌─────────────────────────────────────┐
                     │        Search Orchestrator          │
                     │ (Per-Provider Circuit Breakers)     │
                     └──────┬───────────┬───────────┬──────┘
                            │           │           │
                    ┌───────▼──┐   ┌────▼─────┐  ┌──▼───────┐
                    │DuckDuckGo│   │   Bing   │  │  Yahoo   │
                    └───────┬──┘   └────┬─────┘  └──┬───────┘
                            │           │           │
                            └───────────┼───────────┘
                                        ▼
                     ┌─────────────────────────────────────┐
                     │         Pipeline Processing         │
                     │   - Tracking Parameter Stripping    │
                     │   - Canonical URL Deduplication     │
                     │   - Reciprocal Rank Fusion (RRF)    │
                     └──────────────────┬──────────────────┘
                                        │
                                        ▼
                     ┌─────────────────────────────────────┐
                     │      Safe Webpage Extractor         │
                     │  (SSRF Guard, HTML Pruning, MD)     │
                     └──────────────────┬──────────────────┘
                                        │
                                        ▼
                               Unified Response
```

---

## Quick Start

To run both the backend and frontend as containers, follow the root [Docker Compose guide](../README.md#run-the-full-stack-with-docker).

### 1. Build

Requires Go 1.22+ (tested on `go1.27.3 darwin/arm64`). From the `backend/` directory:

```bash
cd backend
go build -o bin/plewtus ./cmd/plewtus
go build -o bin/plewtus-mcp ./cmd/plewtus-mcp
```

### 2. Run the HTTP Server

```bash
./bin/plewtus
```

Server starts on `http://localhost:8080` by default.

### 3. Test with curl

**Simple Search:**
```bash
curl -s -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"query": "golang concurrency patterns", "limit": 3}'
```

**Search with Markdown Webpage Extraction:**
```bash
curl -s -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "go standard library net/http",
    "limit": 2,
    "extract": {
      "enabled": true,
      "mode": "markdown",
      "max_pages": 2
    }
  }'
```

---

## Using as an MCP Server (Model Context Protocol)

Plewtus can be connected directly to AI agents such as **Claude Desktop** or **Cursor**.

### Option A: Subcommand
```bash
./bin/plewtus mcp
```

### Option B: Dedicated Binary
```bash
./bin/plewtus-mcp
```

### Claude Desktop Configuration (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "plewtus": {
      "command": "/absolute/path/to/PlewtusV1/backend/bin/plewtus",
      "args": ["mcp"]
    }
  }
}
```

See [docs/MCP.md](docs/MCP.md) for full configuration instructions and tool schemas.

---

## Documentation

- **[API Reference](docs/API.md):** Complete HTTP contract, parameters, response shape, headers, and error codes.
- **[Configuration Guide](docs/CONFIGURATION.md):** Environment variables reference, timeouts, quotas, tiers, and circuit breaker settings.
- **[MCP Guide](docs/MCP.md):** Integration guide for Claude Desktop, Cursor, and MCP tools.
- **[OpenAPI 3.1.0 Specification](docs/openapi.yaml):** Machine-readable API schema ([YAML](docs/openapi.yaml) and [JSON](docs/openapi.json)).

---

## Testing

Run all unit, integration, and end-to-end tests:

```bash
go test -count=1 ./...
```

---

## License

MIT License.
