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
| DuckDuckGo | **Working** | Requires POST with `application/x-www-form-urlencoded` + `Origin`/`Referer` headers. GET triggers bot challenge modal (`HTTP 202`). Primary `/html/` with automatic fallback to `/lite/`. Ads (`.result--ad`, `.badge--ad`, `duckduckgo.com/y.js`) cleanly filtered. Supports `df` (freshness) and `kl` (region). Temporary bot challenges detected cleanly without crash. |
| Bing | **Working** | Plain-HTML SERP via GET `https://www.bing.com/search`. Organic results in `<li class="b_algo">`. Click URLs (`/ck/a?!...&u=a1<base64>`) are automatically base64-unwrapped to clean destination URLs. Date in `.news_dt` extracted when available. Ads in `.b_ad` filtered out. |
| Yahoo | **Working** | Plain-HTML SERP via GET `https://search.yahoo.com/search`. Requires cookie jar to handle initial 307 redirect (`YBV` cookie). Redirect URLs (`r.search.yahoo.com/.../RU=<encoded_url>/...`) are automatically unwrapped to clean destination URLs. Up to 3 retries with exponential backoff and idle connection resets for transient ATS reverse-proxy EOFs. |
| Mojeek | **Cut** | Evaluated and rejected. Returns 403 Forbidden with default User-Agent and an Altcha JS-based CAPTCHA challenge (`/js/page_specific/challenge.js`) when spoofing desktop User-Agent. Cut to prevent brittle failures and eliminate dead code. |

## Phases Roadmap & Status
- [x] **Phase 1: Establish the backend** — Go module, config, server foundation, internal provider types, public AI-agent API schema, health endpoint. (Verified with live curl)
- [x] **Phase 2: Build the provider architecture** — Modular provider interface, provider registry, polite HTTP client, HTML DOM utilities, DuckDuckGo scraper with primary `/html/` and fallback `/lite/`. (Verified with fixtures & live queries)
- [x] **Phase 3: Integrate additional search engines** — Bing & Yahoo scrapers with automated URL unwrapping and ad filtering; Mojeek evaluated and cut due to Altcha captcha. (Verified with fixtures & live queries)
- [x] **Phase 4: Parallel search orchestration** — Concurrent dispatch, per-provider timeouts, partial response resilience, server integration. (Verified with unit tests, live integration, and live curl)
- [ ] **Phase 5: Result normalization and deduplication** — Canonical URL normalization, tracker stripping, result merging, engine discovery preservation.
- [ ] **Phase 6: Ranking and result fusion** — Reciprocal Rank Fusion (RRF).
- [ ] **Phase 7: AI-friendly content extraction** — Clean text, highlights, Markdown with SSRF & resource guards.
- [ ] **Phase 8: Search API features** — Query filters, freshness, domain constraints, content options.
- [ ] **Phase 9: Reliability and usage management** — In-memory caching, rate limiting, quotas, fallback.
- [ ] **Phase 10: Final integration and documentation** — End-to-end integration, OpenAPI spec, MCP server.

## Architectural Decisions
1. **Zero External Dependencies for Base Server:** Used Go standard library `net/http` (with Go 1.22+ pattern routing like `"GET /health"`, `"POST /search"`) and `log/slog` for structured logging. Minimal footprint.
2. **AI-Agent-First Schema Contract:**
   - Dedicated schema: echoed query, ranked results carrying `id`, `title`, `url`, `snippet`, `rank`, `score`, `engines` (recording discovering engine names and their original ranks), `published_date`, and optional `content`.
   - Honest metadata: execution time (`took_ms`), provider status (`queried`, `successful`, `failed` with granular error descriptions), and pipeline stats (`raw_count`, `deduped_count`).
3. **Decoupled Provider Architecture:**
   - `provider.Provider` interface: `Name() string`, `Search(ctx, req) (*model.ProviderResponse, error)`.
   - `provider.Registry`: Thread-safe provider registration and lookup.
   - `provider.HTTPClient`: Standard polite scraper wrapper with cookie jar, redirect preservation, configurable timeouts, and transport connection recycling.
   - `provider/htmlutil`: Reusable DOM traversal, class matching, and search engine redirect parameter unwrapping.
4. **Direct URL Unwrapping Across Search Engines:**
   - Bing wraps external links in `/ck/a?!...&u=a1<base64>`. Decoded via base64 URL unpadding directly in `htmlutil.UnwrapURL`.
   - Yahoo wraps external links in `/RU=<encoded_url>/`. Unescaped directly in `htmlutil.UnwrapURL`.
   - DuckDuckGo wraps external links in `/l/?uddg=<encoded_url>`. Unescaped directly in `htmlutil.UnwrapURL`.
   - Completely avoids wasteful HTTP redirect roundtrips and protects agent privacy.
5. **Parallel Search Orchestration Design:**
   - `orchestrator.Orchestrator`: Dispatches concurrent goroutines for each requested provider with bounded per-provider contexts.
   - Per-goroutine panic recovery prevents individual scraper panics from affecting the process.
   - Partial response resilience: if some providers fail or time out, successful results are preserved and failures are recorded in `metadata.providers.failed`.
   - Context cancellation resilience: drains available responses and safely terminates without leaking goroutines.
   - Fair round-robin interleaving prior to Phase 6 RRF ranking.
6. **Server Integration via Searcher Interface:** `server.Server` receives a `Searcher` interface, enabling fast unit testing with mock searchers alongside live end-to-end orchestrator tests.

## Tests Performed
- `internal/config`: Default configs, environment variable overrides.
- `internal/model`: SearchRequest normalization, query validation, freshness/extract mode validation, SearchResponse JSON serialization.
- `internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight checks, searcher error & timeout handling (`500`, `408`), live server integration test.
- `internal/provider`: Registry registration, thread safety, provider lookup.
- `internal/provider/htmlutil`: Node finding, attribute extraction, class matching, text extraction, URL unwrapping for direct, protocol-relative, DDG, Bing, and Yahoo links.
- `internal/provider/duckduckgo`: Fixtures, challenge detection, live queries.
- `internal/provider/bing`: Fixtures, challenge detection, live queries.
- `internal/provider/yahoo`: Fixtures, challenge detection, live queries with ATS EOF retry.
- `internal/provider/all`: Registry initialization with all active providers.
- `internal/orchestrator`:
  - `TestOrchestrator_ParallelExecution`: 3 providers running in ~100ms instead of 300ms sequential.
  - `TestOrchestrator_PerProviderTimeout`: Slow provider cancelled at 60ms while fast provider succeeds.
  - `TestOrchestrator_PartialResponseResilience`: Working provider succeeds while rate-limited and panicking providers are reported in `metadata.providers.failed`.
  - `TestOrchestrator_UnknownProvider`: Missing providers recorded as failed without erroring the request.
  - `TestOrchestrator_AllProvidersFail`: Returns empty results slice with all errors recorded.
  - `TestOrchestrator_ContextCancelled`: Handles parent cancellation cleanly without goroutine leak.
  - `TestOrchestrator_InterleavingAndLimit`: Round-robin rank interleaving and limit truncation.
  - `TestOrchestrator_LiveEngines`: Live parallel query to DuckDuckGo, Bing, and Yahoo simultaneously in 1.17s.
- `cmd/plewtus`: Full binary build and live curl test against `http://localhost:8888/search` returning real results from all 3 engines.

## Next Step
Proceed to **Phase 5 — Result normalization and deduplication**:
1. Implement URL canonicalization and normalization (strip tracking params like UTM, fbclid, gclid, normalize schemes, lowercasing domains, removing fragments and trailing slashes).
2. Deduplicate results referring to identical canonical URLs across providers.
3. Merge duplicate result metadata (preserve all discovering engines in `SearchResult.Engines` with their original ranks, combine snippets or pick the most informative snippet).
4. Integrate deduplication into the search processing pipeline.
