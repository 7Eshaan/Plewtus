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
- [x] **Phase 5: Result normalization and deduplication** — Canonical URL normalization, tracker stripping, result merging, engine discovery preservation. (Verified with pipeline tests, orchestrator tests, and live curl)
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
   - Avoids wasteful HTTP redirect roundtrips and protects agent privacy.
5. **Parallel Search Orchestration:**
   - `orchestrator.Orchestrator`: Dispatches concurrent goroutines for each requested provider with bounded per-provider contexts.
   - Per-goroutine panic recovery prevents individual scraper panics from affecting the process.
   - Partial response resilience: if some providers fail or time out, successful results are preserved and failures are recorded in `metadata.providers.failed`.
   - Context cancellation resilience: drains available responses and safely terminates without leaking goroutines.
6. **URL Normalization & Canonicalization (`internal/pipeline`):**
   - Strips tracking query parameters (`utm_*`, `gclid`, `fbclid`, `msclkid`, `twclid`, `guccounter`, `ref`, etc.) while strictly preserving semantic parameters.
   - Deterministic query parameter sorting (`a=1&b=2`).
   - Normalizes schemes to lowercase https, cleans paths via `path.Clean`, removes trailing fragments (`#...`), and canonicalizes default index files (`/index.html`).
   - Two-tier representation: a clean display URL (`https://...`) and a canonical comparison key (stripping `www.`, scheme variance, and non-root trailing slashes) for robust identity matching.
7. **Multi-Engine Consensus & Metadata Merging:**
   - Retains all discovering engines and their original SERP ranks in `SearchResult.Engines`.
   - Intelligent metadata selection: prefers non-truncated titles and richer, more informative snippets; preserves published dates.
   - Consensus-aware ordering before Phase 6 RRF: pages confirmed by more engines rank higher.

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
- `internal/pipeline`:
  - `TestNormalizeURL_TrackingParameters`: Stripping UTM, click IDs, referral tokens, deterministic query sorting.
  - `TestNormalizeURL_CanonicalMatching`: Canonical key matching across schemes, ports, www prefixes, trailing slashes, fragments, index files.
  - `TestDeduplicate_IdenticalURLsAcrossEngines`: Deduplicating identical URLs across DuckDuckGo, Bing, and Yahoo into a single result with 3 discovering engine sources.
  - `TestDeduplicate_EngineConsensusRanking`: Verifying multi-engine consensus ranking (trio > duo > solo).
  - `TestDeduplicate_TitleTruncationSelection`: Automatic selection of non-truncated titles.
  - `TestDeduplicate_LimitEnforcement`: Preserving accurate `dedupedCount` before limit truncation.
- `internal/orchestrator`: Parallel execution, per-provider timeouts, partial response resilience, unknown provider handling, context cancellation, limit interleaving, deduplication integration, live multi-engine queries.
- `cmd/plewtus`: Full binary build and live curl test against `http://localhost:8889/search` returning deduplicated results with honest `raw_count` and `deduped_count`.

## Next Step
Proceed to **Phase 6 — Ranking and result fusion**:
1. Implement Reciprocal Rank Fusion (RRF) algorithm: \( \text{score}(d) = \sum_{m \in M} \frac{1}{k + r_m(d)} \) with standard smoothing parameter \( k = 60 \).
2. Decouple ranking logic into a dedicated fusion module so future re-ranking algorithms or ML models can be plugged in seamlessly.
3. Compute honest fused scores and final ranked ordering across all discovering engines.
4. Verify RRF with algorithmic unit tests, edge-case tests, and live multi-provider queries.
