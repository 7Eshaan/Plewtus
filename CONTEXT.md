# Plewtus Project Context

## Overview
Plewtus is an AI-agent search API backend that aggregates search results across multiple search engines (DuckDuckGo, Bing, Yahoo, Mojeek) using metasearch architecture. It normalizes, deduplicates, and ranks results via Reciprocal Rank Fusion (RRF), with optional AI-friendly webpage extraction (clean text, highlights, Markdown).

## Environment
- **Go Version:** `go1.27.3 darwin/arm64`
- **Dependencies:** Standard library + `golang.org/x/net v0.62.0` (official Go sub-repository for HTML parsing).
- **Git State:** Managed exclusively by user (no automated git write commands).

## Provider Status Table
| Engine | Status | Quirks / Notes |
|---|---|---|
| DuckDuckGo | **Working** | Requires POST with `application/x-www-form-urlencoded` + `Origin`/`Referer` headers. GET triggers bot challenge modal (`HTTP 202`). Primary `/html/` with automatic fallback to `/lite/`. Ads (`.result--ad`, `.badge--ad`, `duckduckgo.com/y.js`) cleanly filtered. Supports `df` (freshness) and `kl` (region). |
| Bing | Pending (Phase 3) | Preliminary curl shows clean HTML SERP (`li.b_algo`, `h2 a`, `div.b_caption p`) with desktop User-Agent. |
| Yahoo | Pending (Phase 3) | Target plain-HTML SERP; preliminary curl noted HTTP 307 redirect with `YBV` cookie to `/_bv/v.gif`. |
| Mojeek | Candidate (Phase 3) | Plain-HTML search engine candidate; preliminary curl hit altcha challenge. |

## Phases Roadmap & Status
- [x] **Phase 1: Establish the backend** — Go module, config, server foundation, internal provider types, public AI-agent API schema, health endpoint. (Verified with live curl)
- [x] **Phase 2: Build the provider architecture** — Modular provider interface, provider registry, polite HTTP client, HTML DOM utilities, DuckDuckGo scraper with primary `/html/` and fallback `/lite/`. (Verified with fixtures & live queries)
- [ ] **Phase 3: Integrate additional search engines** — Bing & Yahoo scrapers.
- [ ] **Phase 4: Parallel search orchestration** — Concurrent dispatch, per-provider timeouts, partial response resilience.
- [ ] **Phase 5: Result normalization and deduplication** — Canonical URL normalization, tracker stripping, result merging.
- [ ] **Phase 6: Ranking and result fusion** — Reciprocal Rank Fusion (RRF).
- [ ] **Phase 7: AI-friendly content extraction** — Clean text, highlights, Markdown with SSRF & resource guards.
- [ ] **Phase 8: Search API features** — Query filters, freshness, domain constraints, content options.
- [ ] **Phase 9: Reliability and usage management** — In-memory caching, rate limiting, quotas, fallback.
- [ ] **Phase 10: Final integration and documentation** — End-to-end integration, OpenAPI spec, MCP server.

## Architectural Decisions
1. **Zero External Dependencies for Base Server:** Used Go standard library `net/http` (with modern Go 1.22+ pattern routing like `"GET /health"`, `"POST /search"`) and `log/slog` for structured logging. Minimal footprint.
2. **AI-Agent-First Schema Contract:**
   - Dedicated schema: echoed query, ranked results carrying `id`, `title`, `url`, `snippet`, `rank`, `score`, `engines` (recording discovering engine names and their original ranks), `published_date`, and optional `content`.
   - Honest metadata: execution time (`took_ms`), provider status (`queried`, `successful`, `failed` with granular error descriptions), and pipeline stats (`raw_count`, `deduped_count`).
3. **Decoupled Provider Architecture:**
   - `provider.Provider` interface: `Name() string`, `Search(ctx, req) (*model.ProviderResponse, error)`.
   - `provider.Registry`: Thread-safe provider registration and lookup.
   - `provider.HTTPClient`: Standard polite scraper wrapper with cookie jar, redirect preservation, and configurable timeouts.
   - `provider/htmlutil`: Reusable DOM traversal, class matching, and redirect parameter unwrapping (e.g. `uddg`).
4. **DuckDuckGo Dual-Endpoint Resilience:**
   - Direct GET requests to DuckDuckGo trigger bot anomaly detection (`anomaly-modal`).
   - Using form POST with `Origin` and `Referer` headers successfully yields full plain-HTML SERP.
   - Dual-endpoint failover: Scrapes `html.duckduckgo.com/html/` as primary; falls back automatically to `lite.duckduckgo.com/lite/` if challenged or throttled.
   - Strict ad filtration: Excludes sponsored links, `.result--ad`, `.badge--ad`, and tracking redirect URLs (`duckduckgo.com/y.js`, `bing.com/aclick`).

## Tests Performed
- `internal/config`: Default configs, environment variable overrides.
- `internal/model`: SearchRequest normalization, query validation, freshness/extract mode validation, SearchResponse JSON serialization.
- `internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight checks.
- `internal/provider`: Registry registration, thread safety, provider lookup.
- `internal/provider/htmlutil`: Node finding, attribute extraction, class matching, text extraction, URL unwrapping.
- `internal/provider/duckduckgo`:
  - Fixture tests against saved real HTML responses (`testdata/fixtures/duckduckgo_html.html` and `testdata/fixtures/duckduckgo_lite.html`).
  - Challenge/anomaly modal detection test.
  - Live query tests against DuckDuckGo:
    - `"golang concurrency"` (5 results returned, 774ms)
    - `"apple stock news"` (5 results returned, 708ms)
    - `"quantum computing"` (5 results returned, 781ms)
    - Verified titles, URLs, snippets, sequential ranks, and zero ad leakage.

## Next Step
Proceed to **Phase 3 — Integrate additional search engines**:
1. Inspect real responses and redirects for Bing and Yahoo.
2. Build and verify Bing scraper.
3. Build and verify Yahoo scraper (or assess Mojeek/alternative if blocked).
4. Save HTML fixtures for test suites.
