# Plewtus Project Context

## Overview
Plewtus is an AI-agent search API backend that aggregates search results across multiple search engines (DuckDuckGo, Bing, Yahoo) using metasearch architecture. It normalizes, deduplicates, and ranks results via Reciprocal Rank Fusion (RRF), with optional AI-friendly webpage extraction (clean text, highlights, Markdown), in-memory response caching, token-bucket rate limiting, configurable free-tier quotas, provider circuit breaking, and native Model Context Protocol (MCP) support.

## Environment & Repository Layout
- **Go Version:** `go1.27.3 darwin/arm64`
- **Dependencies:** Standard library + `golang.org/x/net v0.62.0` (official Go sub-repository for HTML parsing).
- **Git State:** Managed exclusively by user (no automated git write commands).
- **Repository Layout:**
  - Root: `CONTEXT.md`, `.gitignore`, `backend/`, `frontend/`
  - `backend/`: Contains the complete Go backend codebase, module configuration, tests, and documentation. All Go commands (`go build ./...`, `go vet ./...`, `go test ./...`, and server execution) run from inside `backend/`.
  - `frontend/`: Empty placeholder directory (`frontend/.gitkeep`) ready for frontend development.
- **Key File Paths:**
  - Server entrypoint: `backend/cmd/plewtus/main.go`
  - MCP stdio server entrypoint: `backend/cmd/plewtus-mcp/main.go`
  - HTTP Server & Handlers: `backend/internal/server/`
  - Search Orchestrator: `backend/internal/orchestrator/orchestrator.go`
  - Pipeline (Normalization, Dedup, RRF): `backend/internal/pipeline/`
  - Search Engine Scrapers: `backend/internal/provider/`
  - Webpage Extractor: `backend/internal/extractor/`
  - In-Memory LRU Cache: `backend/internal/cache/`
  - Usage & Quota Management: `backend/internal/usage/`
  - Circuit Breakers: `backend/internal/reliability/`
  - End-to-End Tests: `backend/test/`
  - Test Fixtures: `backend/testdata/fixtures/`
  - Documentation & OpenAPI Spec: `backend/docs/` (`openapi.yaml`, `openapi.json`, `API.md`, `CONFIGURATION.md`, `MCP.md`)
  - Go Module: `backend/go.mod`, `backend/go.sum`
  - Environment Example: `backend/.env.example`

## Provider Status Table
| Engine     | Status      | Quirks / Notes                                                                                                                                                                                                                                                                                                                                                                                                      |
| ---------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DuckDuckGo | **Working** | Requires POST with `application/x-www-form-urlencoded` + `Origin`/`Referer` headers. Intermittent bot challenge modal (`HTTP 202` with `anomaly-modal` / `anomaly.js`) is automatically detected by `parser.isChallengeResponse()`, safely tripping the circuit breaker without crashing or blocking user queries. Primary `/html/` with automatic fallback to `/lite/`. Ads cleanly filtered. Supports `df` (freshness) and `kl` (region). |
| Bing       | **Working** | Plain-HTML SERP via GET `https://www.bing.com/search`. Organic results in `<li class="b_algo">`. Click URLs (`/ck/a?!...&u=a1<base64>`) are automatically base64-unwrapped to clean destination URLs. Date in `.news_dt` extracted when available. Ads in `.b_ad` filtered out. Language defaults to `setlang=en` and region to `cc=US` when omitted to prevent IP-localized noise from hijacking general technical queries. |
| Yahoo      | **Working** | Plain-HTML SERP via GET `https://search.yahoo.com/search`. Requires cookie jar to handle initial 307 redirect (`YBV` cookie). Redirect URLs (`r.search.yahoo.com/.../RU=<encoded_url>/...`) are automatically unwrapped to clean destination URLs. Language defaults to `vl=lang_en` and region to `vc=us`. Retries with exponential backoff on transient ATS reverse-proxy 500/EOF errors.                                                    |
| Mojeek     | **Cut**     | Evaluated and rejected. Returns 403 Forbidden with default User-Agent and an Altcha JS-based CAPTCHA challenge (`/js/page_specific/challenge.js`) when spoofing desktop User-Agent. Cut to prevent brittle failures and eliminate dead code.                                                                                                                                                                        |

## Phases Roadmap & Status
- [x] **Phase 1: Establish the backend** — Go module, config, server foundation, internal provider types, public AI-agent API schema, health endpoint. (Verified with live curl)
- [x] **Phase 2: Build the provider architecture** — Modular provider interface, provider registry, polite HTTP client, HTML DOM utilities, DuckDuckGo scraper with primary `/html/` and fallback `/lite/`. (Verified with fixtures & live queries)
- [x] **Phase 3: Integrate additional search engines** — Bing & Yahoo scrapers with automated URL unwrapping and ad filtering; Mojeek evaluated and cut due to Altcha captcha. (Verified with fixtures & live queries)
- [x] **Phase 4: Parallel search orchestration** — Concurrent dispatch, per-provider timeouts, partial response resilience, server integration. (Verified with unit tests, live integration, and live curl)
- [x] **Phase 5: Result normalization and deduplication** — Canonical URL normalization, tracker stripping, result merging, engine discovery preservation. (Verified with pipeline tests, orchestrator tests, and live curl)
- [x] **Phase 6: Ranking and result fusion** — Reciprocal Rank Fusion (RRF) with standard $k=60$, decoupled Ranker interface, transparent score computation. (Verified with formula tests, consensus tests, and live curl)
- [x] **Phase 7: AI-friendly content extraction** — SSRF & resource guards (loopback, private, link-local, CGNAT, DNS rebinding, redirect validation, 2MB size limit, HTML MIME check), DOM cleaning, extraction modes (markdown, highlights, text, metadata), developer limits (max_pages, max_chars_per_page), orchestrator integration. (Verified with unit tests, live extraction, live integration, and live curl)
- [x] **Phase 8: Search API features** — Query filters, freshness mapping (day, week, month, year), domain inclusion & exclusion with Boolean OR queries and pipeline post-filtering, ISO country/language settings, offset & limit pagination, transparent metadata.filters reflection. (Verified with unit tests, live integration, and live curl)
- [x] **Phase 9: Reliability and usage management** — In-memory LRU cache with TTL and deterministic key hashing, token bucket rate limiting (per API key and client IP), daily request quotas with UTC reset, configurable free usage tier, API key authentication, provider circuit breaking (Closed/Open/Half-Open) and fallback to alternative healthy engines. (Verified with unit tests, server integration, and live curl)
- [x] **Phase 10: Final integration and documentation** — End-to-end full lifecycle integration tests, Model Context Protocol (MCP) stdio server (`internal/mcp`, `cmd/plewtus-mcp`, `plewtus mcp`), complete OpenAPI 3.1.0 specifications (`docs/openapi.yaml`, `docs/openapi.json`), comprehensive documentation (`README.md`, `docs/API.md`, `docs/CONFIGURATION.md`, `docs/MCP.md`, `.env.example`). (Verified with unit tests, E2E tests, and live MCP stdio sessions)
- [x] **Repository Restructure** — Restructured repository into `backend/` and `frontend/` folders at root. All Go files, tests, documentation, and configuration moved into `backend/`. `frontend/` initialized with `.gitkeep`. (Verified with `go build ./...`, `go vet ./...`, `go test ./...`, and live curl `/health` from within `backend/`)

## Architectural Decisions
1. **Zero External Dependencies for Base Server:** Used Go standard library `net/http` (with Go 1.22+ pattern routing like `"GET /health"`, `"POST /search"`) and `log/slog` for structured logging. Minimal footprint.
2. **AI-Agent-First Schema Contract:**
   - Dedicated schema: echoed query, ranked results carrying `id`, `title`, `url`, `snippet`, `rank`, `score`, `engines` (recording discovering engine names and their original ranks), `published_date`, and optional `content`.
   - Honest metadata: execution time (`took_ms`), provider status (`queried`, `successful`, `failed` with granular error descriptions), pipeline stats (`raw_count`, `deduped_count`), active `filters`, and `cached` status.
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
   - Two-tier representation: clean display URL (`https://...`) and invariant canonical comparison key (stripping `www.`, scheme variance, and non-root trailing slashes) for robust identity matching.
7. **Multi-Engine Consensus & Metadata Merging:**
   - Retains all discovering engines and their original SERP ranks in `SearchResult.Engines`.
   - Metadata enhancement: prefers non-truncated titles and richer snippets; preserves published dates.
8. **Decoupled Ranking & Reciprocal Rank Fusion (RRF):**
   - `Ranker` interface decouples ranking algorithms from deduplication and orchestrator logic, allowing custom or chained rerankers (`MultiRanker`).
   - `RRFRanker`: Computes \( \text{score}(d) = \sum_{m \in M} \frac{1}{k + r_m(d)} \) with standard smoothing constant \( k = 60 \). Scores are rounded to 6 decimal places for clean consumption.
   - Scale-invariant: avoids comparing incomparable raw relevance scores from different engines.
   - Deterministic tie-breaking: RRF score descending, engine count descending, best original position ascending, stable discovery order.
9. **Secure Webpage Extraction & AI Content Synthesis (`internal/extractor`):**
   - SSRF Defense: Rejects private IPv4 (RFC 1918), IPv6 unique-local, link-local / cloud metadata (`169.254.169.254`), loopback (`127.0.0.0/8`, `::1`), CGNAT (`100.64.0.0/10`), broadcast, and IPv4-mapped IPv6. `SafeDialContext` resolves DNS and dials safe public IPs directly, preventing DNS rebinding. `SafeCheckRedirect` re-validates each redirect hop (max 5).
   - Resource & MIME Guards: Caps response body reading at 2 MB via `io.LimitReader`. Validates HTML MIME types (`text/html`, `application/xhtml+xml`) and rejects non-HTML binaries/PDFs.
   - DOM Cleaning & Conversion: Prunes noise elements (`<script>`, `<style>`, `<nav>`, `<footer>`, `<aside>`, `class*="ad"`, `class*="cookie"`, etc.), targets primary content (`<main>`, `<article>`), extracts metadata (author, description, date), and converts into clean Markdown, plain text, or query-weighted highlights.
   - Resource Bounding: Strictly honors `max_pages` (fetching only top N results up to the limit) and truncates cleanly at sentence/word boundaries with `max_chars_per_page`. Individual page errors are handled gracefully without aborting the search.
10. **Query Constraints, Domain Isolation, and Pagination (`internal/pipeline`, `internal/provider`):**
    - Freshness: Mapped across all three providers (DuckDuckGo: `d/w/m/y`, Bing: `ex1:"ez1"/"ez2"/"ez3"/"ez5"`, Yahoo: `1d/1w/1m/1y`).
    - Dual-Layer Domain Controls: Multi-domain inclusion grouped via Boolean OR in upstream queries (`(site:a.com OR site:b.com)`), while `pipeline.matchesDomainFilters` enforces 100% strict hostname/subdomain compliance post-scraping.
    - Federated Metasearch Pagination: Upstream scrapers pull `limit + offset` candidates into the unified RRF fusion, and `DeduplicateWithOptions` slices `[offset : offset+limit]`, ensuring stable global ranking across pages.
    - Transparent Metadata: Echoes active search constraints in `metadata.filters` (`freshness`, `include_domains`, `exclude_domains`, `country`, `language`, `offset`, `limit`).
11. **Multi-Tier Usage Management, In-Memory Caching & Circuit Breaking (`internal/cache`, `internal/usage`, `internal/reliability`):**
    - In-Memory LRU Cache: Canonical query + filter + extract key hashing with SHA-256. Deep-cloned cache retrieval with transparent `metadata.cached: true` flag. Configurable TTL and max entry bounds.
    - Usage Management & Configurable Free Tier: Token bucket rate limiting and UTC daily quota tracking per identity (`api_key:<token>` or anonymous `ip:<addr>`). Standard rate limit and quota response headers (`X-RateLimit-*`, `X-Quota-*`, `Retry-After`). 401 Unauthorized on invalid/missing keys when auth is required, 429 Too Many Requests on rate limit or daily quota exhaustion.
    - Provider Circuit Breaking & Fallback: Automatic failure tracking per upstream provider with Closed, Open, and Half-Open probe states. Degraded providers fast-fail during cooldown without network latency; orchestrator automatically queries alternative healthy registered engines when primary providers trip.
12. **Native Model Context Protocol (MCP) Support (`internal/mcp`, `cmd/plewtus-mcp`):**
    - Direct AI Agent Integration: Exposes `search` and `extract` tools over standard JSON-RPC 2.0 stdio protocol (`protocolVersion: 2024-11-05`), compatible out-of-the-box with Claude Desktop, Cursor, and custom agent frameworks.
    - Stream Isolation: Enforces strict separation between stdout (JSON-RPC messages only) and stderr (all diagnostic logs), preventing parse errors in agent hosts.
    - Dual-Binary & Subcommand Invocations: Available as both a dedicated binary (`backend/bin/plewtus-mcp`) and a CLI subcommand (`backend/bin/plewtus mcp`).

## Tests Performed
- `backend/internal/config`: Default configs, environment variable overrides for cache, usage, tiers, API keys, and circuit breakers.
- `backend/internal/cache`: Key determinism across order permutations, deep copy isolation, TTL expiration, LRU eviction at max capacity, disabled caching on TTL=0, concurrent access.
- `backend/internal/reliability`: State transitions (Closed -> Open -> Half-Open -> Closed / Open), cooldown expiration, trial probe serialization, concurrency, manager health lookups and reset.
- `backend/internal/usage`: Free tier assignment, client IP extraction (`X-Forwarded-For`, `X-Real-IP`, `RemoteAddr`), token bucket rate limit consumption and refill, daily quota exhaustion and midnight UTC reset, API key authentication (`Authorization: Bearer` and `X-API-Key`), 401 Unauthorized and 429 Too Many Requests errors.
- `backend/internal/model`: SearchRequest normalization, query validation, freshness/country/language ISO validation, extract mode validation, SearchResponse JSON serialization with `cached` metadata.
- `backend/internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight and header exposition (`X-RateLimit-*`, `X-Quota-*`, `Retry-After`), searcher error & timeout handling (`500`, `408`), rate limit headers and 429 enforcement (`TestServer_RateLimitingHeadersAnd429`), daily quota 429 enforcement (`TestServer_DailyQuotaExceeded429`), authentication 401 enforcement (`TestServer_Authentication401`), cache integration test (`TestServer_CacheIntegration`), live server integration test, live extraction integration test, live search filters integration test.
- `backend/internal/mcp`: Initialize handshake, ping, notifications/initialized, tools/list schema validation, tools/call execution for `search` and `extract`, parameter error handling, and unknown method errors.
- `backend/internal/provider`: Registry registration, thread safety, provider lookup, query builder tests.
- `backend/internal/provider/htmlutil`: Node finding, attribute extraction, class matching, text extraction, URL unwrapping for direct, protocol-relative, DDG, Bing, and Yahoo links.
- `backend/internal/provider/duckduckgo`: Fixtures, challenge detection, live queries.
- `backend/internal/provider/bing`: Fixtures, challenge detection, live queries.
- `backend/internal/provider/yahoo`: Fixtures, challenge detection, live queries with ATS EOF retry.
- `backend/internal/provider/all`: Registry initialization with all active providers.
- `backend/internal/pipeline`: URL normalization, tracking parameter stripping, canonical key matching, multi-engine consensus deduplication, title truncation selection, domain whitelist/blacklist filtering, offset & limit slicing, RRF ranker formula, consensus tie-breaking, custom k smoothing, multi-ranker chains.
- `backend/internal/extractor`: SSRF defense, URL validation, safe redirect checks, HTML pruning, Markdown conversion, highlights extraction, clean boundary truncation, MIME guard, batch extraction, live web extraction.
- `backend/internal/orchestrator`: Parallel execution, per-provider timeouts, partial response resilience, unknown provider handling, context cancellation, limit interleaving, deduplication integration, content extraction integration, metadata filters integration, offset pagination, cache hit integration, circuit breaker fast-failure, circuit breaker fallback, live multi-engine queries.
- `backend/test`:
  - `TestE2E_FullPipelineLifecycle`: End-to-end simulation covering authentication (401 without key), multi-provider dispatch with tracking parameter stripping, multi-engine consensus deduplication, RRF rank fusion score computation, content extraction, rate limit/quota header exposition, and subsequent cache hit.
  - `TestE2E_CircuitBreakerAndFallback`: Simulates multiple provider failures tripping circuit breaker to open, fast-failing during cooldown, and executing fallback to healthy registered engine.
  - `TestOpenAPISpecification_ValidJSON`: Validates that `backend/docs/openapi.json` is well-formed OpenAPI 3.1.0 schema with `/search` and `/health` paths.
- `backend/cmd/plewtus` & `backend/cmd/plewtus-mcp`:
  - Built binaries: `backend/bin/plewtus` and `backend/bin/plewtus-mcp`.
  - Verified live MCP stdio handshake (`initialize` and `tools/list`).
  - Executed live `tools/call` for `search` against DuckDuckGo, Bing, and Yahoo with multi-engine consensus output formatted cleanly over stdio.
  - Verified live HTTP server execution and curl to `http://localhost:8890/health` returning 200 OK after repository restructure.

## Next Step
**Repository restructure complete.**
The repository is split into `backend/` and `frontend/`. All backend verification checks passed.
The next step is the **frontend**.
