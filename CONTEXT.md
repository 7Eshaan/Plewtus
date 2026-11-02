# Plewtus Project Context

## Overview
Plewtus is an AI-agent search API backend that aggregates search results across multiple search engines (DuckDuckGo, Bing, Yahoo, Mojeek) using metasearch architecture. It normalizes, deduplicates, and ranks results via Reciprocal Rank Fusion (RRF), with optional AI-friendly webpage extraction (clean text, highlights, Markdown).

## Environment
- **Go Version:** `go1.27.3 darwin/arm64`
- **Dependencies:** Standard library only (Go 1.22+ `net/http` routing, `log/slog`). Zero external dependencies so far.
- **Git State:** Managed exclusively by user (no automated git write commands).

## Provider Status Table
| Engine | Status | Quirks / Notes |
|---|---|---|
| DuckDuckGo | Pending (Phase 2) | Target endpoints: `html.duckduckgo.com/html` & `/lite` |
| Bing | Pending (Phase 3) | Target plain-HTML SERP with polite User-Agent |
| Yahoo | Pending (Phase 3) | Target plain-HTML SERP with polite User-Agent |
| Mojeek | Candidate (Phase 3) | Plain-HTML search engine, fallback candidate |

## Phases Roadmap & Status
- [x] **Phase 1: Establish the backend** — Go module, config, server foundation, internal provider types, public AI-agent API schema, health endpoint. (Verified with live curl)
- [ ] **Phase 2: Build the provider architecture** — Modular provider interface & DuckDuckGo scraper.
- [ ] **Phase 3: Integrate additional search engines** — Bing & Yahoo scrapers.
- [ ] **Phase 4: Parallel search orchestration** — Concurrent dispatch, per-provider timeouts, partial response resilience.
- [ ] **Phase 5: Result normalization and deduplication** — Canonical URL normalization, tracker stripping, result merging.
- [ ] **Phase 6: Ranking and result fusion** — Reciprocal Rank Fusion (RRF).
- [ ] **Phase 7: AI-friendly content extraction** — Clean text, highlights, Markdown with SSRF & resource guards.
- [ ] **Phase 8: Search API features** — Query filters, freshness, domain constraints, content options.
- [ ] **Phase 9: Reliability and usage management** — In-memory caching, rate limiting, quotas, fallback.
- [ ] **Phase 10: Final integration and documentation** — End-to-end integration, OpenAPI spec, MCP server.

## Architectural Decisions
1. **Zero External Dependencies for Base Server:** Used Go standard library `net/http` (with modern Go 1.22+ pattern routing like `"GET /health"`, `"POST /search"`) and `log/slog` for structured logging. Minimal footprint, zero supply chain risk.
2. **AI-Agent-First Schema Contract:**
   - Dedicated schema: echoed query, ranked results carrying `id`, `title`, `url`, `snippet`, `rank`, `score`, `engines` (recording discovering engine names and their original ranks), `published_date`, and optional `content`.
   - Honest metadata: execution time (`took_ms`), provider status (`queried`, `successful`, `failed` with granular error descriptions), and pipeline stats (`raw_count`, `deduped_count`).
3. **Decoupled Provider Interface:** Scrapers implement `provider.Provider` (`Search(ctx, req) (*model.ProviderResponse, error)`) returning `RawSearchResult` items, keeping scraping isolated from normalization and ranking.

## Tests Performed
- `go test -v ./...`:
  - `internal/config`: Default configs, environment variable overrides.
  - `internal/model`: SearchRequest normalization, query validation, freshness/extract mode validation, SearchResponse JSON serialization.
  - `internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight checks.
- Real binary live test:
  - Compiled `bin/plewtus`.
  - Executed server on `:8080`.
  - Live `curl -i http://localhost:8080/health` verified (Status 200 OK, valid JSON).
  - Live `curl -i -X POST http://localhost:8080/search` verified (valid JSON contract and 400 Bad Request on empty query).

## Next Step
Proceed to **Phase 2 — Build the provider architecture**:
1. Implement DuckDuckGo HTML scraper using `html.duckduckgo.com/html` and `/lite`.
2. Inspect live response markup, selectors, and redirects.
3. Test against live queries until extraction of title, url, and snippet is consistent.
