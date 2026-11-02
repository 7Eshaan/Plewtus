# Plewtus Project Context

## Overview
Plewtus is an AI-agent search API backend that aggregates search results across multiple search engines (DuckDuckGo, Bing, Yahoo) using metasearch architecture. It normalizes, deduplicates, and ranks results via Reciprocal Rank Fusion (RRF), with optional AI-friendly webpage extraction (clean text, highlights, Markdown).

## Environment
- **Go Version:** `go1.27.3 darwin/arm64`
- **Dependencies:** Standard library + `golang.org/x/net v0.62.0` (official Go sub-repository for HTML parsing).
- **Git State:** Managed exclusively by user (no automated git write commands).

## Provider Status Table
| Engine | Status | Quirks / Notes |
|---|---|---|
| DuckDuckGo | **Working** | Requires POST with `application/x-www-form-urlencoded` + `Origin`/`Referer` headers. GET triggers bot challenge modal (`HTTP 202`). Primary `/html/` with automatic fallback to `/lite/`. Ads (`.result--ad`, `.badge--ad`, `duckduckgo.com/y.js`) cleanly filtered. Supports `df` (freshness) and `kl` (region). |
| Bing | **Working** | Plain-HTML SERP via GET `https://www.bing.com/search`. Organic results in `<li class="b_algo">`. Click URLs (`/ck/a?!...&u=a1<base64>`) are automatically base64-unwrapped to clean destination URLs. Date in `.news_dt` extracted when available. Ads in `.b_ad` filtered out. |
| Yahoo | **Working** | Plain-HTML SERP via GET `https://search.yahoo.com/search`. Requires cookie jar to handle initial 307 redirect (`YBV` cookie). Redirect URLs (`r.search.yahoo.com/.../RU=<encoded_url>/...`) are automatically unwrapped to clean destination URLs. Built-in single retry for transient redirect EOFs. |
| Mojeek | **Cut** | Evaluated and rejected. Returns 403 Forbidden with default User-Agent and an Altcha JS-based CAPTCHA challenge (`/js/page_specific/challenge.js`) when spoofing desktop User-Agent. Cut to prevent brittle failures and eliminate dead code. |

## Phases Roadmap & Status
- [x] **Phase 1: Establish the backend** — Go module, config, server foundation, internal provider types, public AI-agent API schema, health endpoint. (Verified with live curl)
- [x] **Phase 2: Build the provider architecture** — Modular provider interface, provider registry, polite HTTP client, HTML DOM utilities, DuckDuckGo scraper with primary `/html/` and fallback `/lite/`. (Verified with fixtures & live queries)
- [x] **Phase 3: Integrate additional search engines** — Bing & Yahoo scrapers with automated URL unwrapping and ad filtering; Mojeek evaluated and cut due to Altcha captcha. (Verified with fixtures & live queries)
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
   - `provider/htmlutil`: Reusable DOM traversal, class matching, and search engine redirect parameter unwrapping.
4. **Direct URL Unwrapping Across Search Engines:**
   - Bing wraps external links in `/ck/a?!...&u=a1<base64>`. Decoded via base64 URL unpadding directly in `htmlutil.UnwrapURL`.
   - Yahoo wraps external links in `/RU=<encoded_url>/`. Unescaped directly in `htmlutil.UnwrapURL`.
   - DuckDuckGo wraps external links in `/l/?uddg=<encoded_url>`. Unescaped directly in `htmlutil.UnwrapURL`.
   - This completely avoids wasteful HTTP redirect roundtrips and protects agent privacy.
5. **Engine Evaluation Discipline:** Mojeek was inspected with real requests and found to enforce JS-dependent Altcha captchas and 403s. In adherence to project guidelines, it was cut rather than leaving brittle or dead code in the repository.

## Tests Performed
- `internal/config`: Default configs, environment variable overrides.
- `internal/model`: SearchRequest normalization, query validation, freshness/extract mode validation, SearchResponse JSON serialization.
- `internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight checks.
- `internal/provider`: Registry registration, thread safety, provider lookup.
- `internal/provider/htmlutil`: Node finding, attribute extraction, class matching, text extraction, URL unwrapping for direct, protocol-relative, DDG, Bing, and Yahoo links.
- `internal/provider/duckduckgo`:
  - Fixture tests against saved real HTML responses (`testdata/fixtures/duckduckgo_html.html` and `testdata/fixtures/duckduckgo_lite.html`).
  - Challenge/anomaly modal detection test.
  - Live query tests against DuckDuckGo: `"golang concurrency"`, `"apple stock news"`, `"quantum computing"`.
- `internal/provider/bing`:
  - Fixture tests against saved real HTML response (`testdata/fixtures/bing.html`).
  - Challenge detection test.
  - Live query tests against Bing: `"golang concurrency"`, `"apple stock news"`, `"quantum computing"`.
- `internal/provider/yahoo`:
  - Fixture tests against saved real HTML response (`testdata/fixtures/yahoo.html`).
  - Challenge detection test.
  - Live query tests against Yahoo: `"golang concurrency"`, `"apple stock news"`, `"quantum computing"`.
- `internal/provider/all`: Complete registry initialization with all 3 active providers.

## Next Step
Proceed to **Phase 4 — Parallel search orchestration**:
1. Implement the search orchestrator (`internal/orchestrator/orchestrator.go`).
2. Concurrent dispatch across selected providers with per-provider timeouts.
3. Partial response resilience (individual provider failures do not fail the request).
4. Integrate with `POST /search` server handler.
