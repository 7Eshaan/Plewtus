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
- [x] **Phase 6: Ranking and result fusion** — Reciprocal Rank Fusion (RRF) with standard $k=60$, decoupled Ranker interface, transparent score computation. (Verified with formula tests, consensus tests, and live curl)
- [x] **Phase 7: AI-friendly content extraction** — SSRF & resource guards (loopback, private, link-local, CGNAT, DNS rebinding, redirect validation, 2MB size limit, HTML MIME check), DOM cleaning, extraction modes (markdown, highlights, text, metadata), developer limits (max_pages, max_chars_per_page), orchestrator integration. (Verified with unit tests, live extraction, live integration, and live curl)
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

## Tests Performed
- `internal/config`: Default configs, environment variable overrides.
- `internal/model`: SearchRequest normalization, query validation, freshness/extract mode validation, SearchResponse JSON serialization.
- `internal/server`: Health handler (`/health`, `/healthz`), Search handler contract validation, bad JSON handling, CORS preflight checks, searcher error & timeout handling (`500`, `408`), live server integration test, live extraction integration test (`POST /search` with `extract.enabled`).
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
  - `TestDeduplicate_EngineConsensusRanking`: Multi-engine consensus ranking (trio > duo > solo).
  - `TestDeduplicate_TitleTruncationSelection`: Automatic selection of non-truncated titles.
  - `TestDeduplicate_LimitEnforcement`: Preserving accurate `dedupedCount` before limit truncation.
  - `TestRRFRanker_Formula`: Verified exact formula output for \( k=60 \).
  - `TestRRFRanker_MultiEngineConsensusBeatsSingleHighRank`: Confirmed 2-engine #2 consensus outranks single-engine #1.
  - `TestRRFRanker_TieBreaking`: Deterministic stable ordering on identical scores.
  - `TestRRFRanker_CustomK`: Configurable smoothing parameter.
  - `TestRRFRanker_MultiRankerChain`: Chained RRF ranker with secondary reranker.
  - `TestRRFRanker_EdgeCases`: Empty inputs, non-positive positions, duplicate engine sources.
- `internal/extractor`:
  - `TestIsBlockedIP`: Comprehensive IP verification (loopback, RFC 1918 private, link-local, cloud metadata, CGNAT, IPv6 local, IPv4-mapped IPv6, public IPs).
  - `TestValidateURL`: Scheme restrictions (http/https only), rejection of file/ftp/data/javascript URIs and local/internal hostnames.
  - `TestSafeCheckRedirect`: Redirect count capping, blocking redirects to private IPs and non-HTTP protocols.
  - `TestParseHTML_MetadataAndPruning`: Extraction of title, description, author, publication date, and elimination of ads/scripts/banners/sidebars.
  - `TestToMarkdown`: Headings, paragraphs, bold, italics, lists, code fences, blockquotes, links, and tables.
  - `TestExtractHighlights`: Scoring, sentence filtering, and boilerplate exclusion.
  - `TestTruncateCleanly`: Clean sentence and word boundary truncation.
  - `TestFetcher_MIMETypeGuard`: Rejection of non-HTML (PDF) responses and acceptance of HTML.
  - `TestExtractor_ExtractBatch`: `max_pages` enforcement, error resilience, and mode formatting.
  - `TestLiveWebpageExtraction`: Live Markdown and highlights extraction against `https://go.dev`.
- `internal/orchestrator`: Parallel execution, per-provider timeouts, partial response resilience, unknown provider handling, context cancellation, limit interleaving, deduplication integration, content extraction integration (`TestOrchestrator_ContentExtractionIntegration`), live multi-engine queries.
- `cmd/plewtus`: Full binary build and live curl test against `http://localhost:8895/search` verifying RRF results enriched with Markdown, Highlights, and Text content.

## Next Step
Proceed to **Phase 8 — Search API features**:
1. Implement and verify search query filters and options:
   - Freshness filters across search engines (`day`, `week`, `month`, `year`).
   - Domain inclusion (`include_domains`) and exclusion (`exclude_domains`) constraints.
   - Country and language settings (`country`, `language`).
   - Provider selection validation (`providers`).
   - Limit controls and pagination semantics.
2. Extend search response metadata to reflect applied query filters transparently.
3. Test against both fixtures and live engine queries.
