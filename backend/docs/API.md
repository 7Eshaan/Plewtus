# Plewtus Search API Reference

Plewtus is an AI-agent metasearch API that aggregates search results across multiple search engines (DuckDuckGo, Bing, Yahoo), deduplicates destination URLs, removes off-topic results, ranks with Reciprocal Rank Fusion (RRF), and provides optional safe webpage content extraction.

---

## Base URL

```
http://localhost:8080
```

---

## Authentication & Usage Tiers

Plewtus supports two modes of access:

1. **Configurable Free Usage Tier (Default):**
   - Unauthenticated requests are allowed by default without requiring an API key.
   - Rate limits and daily quotas are tracked per client IP or API key. The backend uses the socket peer by default and only honors `X-Forwarded-For` when the socket peer matches `PLEWTUS_TRUSTED_PROXIES`.
   - Default: 10 requests per minute, 100 requests per UTC day.

2. **API Key Authentication:**
   - Clients supply an API key using either of the following headers:
     ```http
     Authorization: Bearer <api_key>
     ```
     or
     ```http
     X-API-Key: <api_key>
     ```
   - Valid keys unlock higher tier limits (e.g., Pro Tier: 60 req/min, 2000 req/day).
   - If `PLEWTUS_AUTH_REQUIRED=true` is configured, requests without a valid API key are rejected with `401 Unauthorized`.

---

## Response Headers

Every `/search` response includes usage and rate limit status headers:

| Header | Description |
|---|---|
| `X-RateLimit-Limit` | Maximum requests permitted per minute in the active tier |
| `X-RateLimit-Remaining` | Remaining request tokens in the current token bucket |
| `X-RateLimit-Reset` | Estimated seconds until the rate limit bucket completely refills |
| `X-Quota-Limit` | Daily request quota limit |
| `X-Quota-Remaining` | Remaining requests for the current UTC calendar day |
| `X-Quota-Reset` | Seconds until daily quota resets (00:00:00 UTC) |
| `Retry-After` | Included on `429 Too Many Requests`; specifies seconds to wait |

---

## Endpoints

### 1. `POST /search`

Executes federated search across multiple search engines with deduplication, relevance filtering, and consensus ranking.

How results are processed:

1. **Normalize & deduplicate** — redirect wrappers are unwrapped, tracking parameters and fragments stripped, and identical pages merged (keeping every engine and its position). The returned `url` is otherwise left as the engine reported it; scheme, `www.`, trailing slashes, index files, and parameter order are ignored only when deciding whether two results are the same page.
2. **Drop off-topic results** — results whose title/snippet do not cover the query (`relevance_rejected_count`) are removed. Matching ignores search operators (`-term`, `site:`, quotes) and common inflections ("laptops" matches "laptop"). If *nothing* matches, the candidates are kept so short or misspelled queries still get an answer.
3. **Discard poisoned engine responses** — search engines sometimes serve unrelated decoy pages to automated clients. An engine's whole page is discarded (`provider_discarded_count`), and the engine reported under `metadata.providers.failed`, when it is off-topic and either another engine returned on-topic results or the query has three or more terms that the page barely touches. Repeated discards open that engine's circuit breaker.
4. **Rank** — results are ordered by `ranking_score = normalized RRF × (0.98 + 0.02 × relevance_score)`. `score` is the raw RRF value (`Σ 1/(60 + position)` over engines); it is normalized against a result that every responding engine ranks first. Agreement between engines and their own ordering decide the ranking; relevance only separates results the engines rank equally.

#### Request Headers

```http
Content-Type: application/json
Authorization: Bearer <token>   # Optional (Free tier used if omitted)
```

#### Request Body (`SearchRequest`)

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `query` | `string` | **Yes** | — | Search query to execute (at most 500 characters). |
| `limit` | `integer` | No | `10` | Max ranked results to return (max `50`). See *Result depth* below. |
| `offset` | `integer` | No | `0` | Offset for slicing results. `offset + limit` must be at most `50`. When `limit` is omitted it shrinks to fit the window. |
| `providers` | `[]string` | No | `["duckduckgo", "bing", "yahoo"]` | Search engines to query: `duckduckgo`, `bing`, `yahoo`, `marginalia` (opt-in). Unknown names are rejected with `400`. Marginalia supports only the `year` freshness window; other windows report it as unsupported for that search. |
| `freshness` | `string` | No | `""` | Discovery recency: `"day"`, `"week"`, `"month"`, or `"year"`. |
| `include_domains` | `[]string` | No | `[]` | Only return results matching these hostnames/subdomains. |
| `exclude_domains` | `[]string` | No | `[]` | Exclude any results matching these hostnames/subdomains. |
| `country` | `string` | No | `""` | 2-letter ISO 3166-1 alpha-2 country code (e.g. `"us"`, `"gb"`). |
| `language` | `string` | No | `""` | 2-letter ISO 639-1 language code (e.g. `"en"`, `"de"`). |
| `extract` | `object` | No | `null` | Optional webpage content extraction configuration. |

#### `extract` Options

| Field | Type | Default | Description |
|---|---|---|---|
| `enabled` | `boolean` | `false` | When `true`, fetches and cleans content for top results. |
| `mode` | `string` | `"highlights"` | Format: `"markdown"` (fills `content.markdown`), `"highlights"` (fills `content.highlights`, falling back to `content.text`), `"text"` (fills `content.text`), or `"metadata"` (title/description/author/date). |
| `max_pages` | `integer` | `3` | Maximum number of top results to extract (max `10`). |
| `max_chars_per_page` | `integer` | `4000` | Maximum character length per extracted page (max `20000`). |

**Result depth.** Each engine returns roughly 7–10 results per page. For windows (`offset + limit`) deeper than 10, every engine fetches further pages, up to three. That is slower (pages are fetched one after another) and best-effort: three engines typically yield 30–45 unique results, so deep pages can come back short. `metadata.stats.deduped_count` is the number of unique results available for the query.

**Caching.** Responses are cached for `PLEWTUS_CACHE_TTL_SECONDS`. A response in which any engine failed is cached for at most one minute.

Request bodies are limited to 1 MiB and must contain exactly one JSON value. Search-engine HTML responses are capped at 4 MiB. Usage counters and the frontend proxy limiter are process-local in-memory guards; use a shared gateway/store for limits that must span replicas or survive restarts.

---

#### Response Body (`SearchResponse`)

```json
{
  "query": "golang concurrency patterns",
  "results": [
    {
      "id": "res_1",
      "title": "Go Concurrency Patterns: Pipelines and cancellation",
      "url": "https://go.dev/blog/pipelines",
      "snippet": "Go Concurrency Patterns (video) presents the basics of Go's concurrency primitives...",
      "rank": 1,
      "score": 0.049180,
      "relevance_score": 1.0,
      "ranking_score": 1.0,
      "engines": [
        { "name": "duckduckgo", "position": 1 },
        { "name": "bing", "position": 1 },
        { "name": "yahoo", "position": 1 }
      ],
      "published_date": "Mar 13, 2014",
      "content": {
        "markdown": "# Go Concurrency Patterns: Pipelines and cancellation\n\n...",
        "extracted_at": "2026-11-03T10:00:00Z"
      }
    }
  ],
  "metadata": {
    "total_results": 1,
    "took_ms": 1150,
    "providers": {
      "queried": ["duckduckgo", "bing", "yahoo"],
      "successful": ["duckduckgo", "bing", "yahoo"],
      "failed": [],
      "timing_ms": {"duckduckgo": 420, "bing": 510, "yahoo": 630}
    },
    "stats": {
      "raw_count": 6,
      "deduped_count": 4,
      "relevance_rejected_count": 0,
      "domain_rejected_count": 0,
      "invalid_url_count": 0,
      "duplicate_count": 2,
      "provider_discarded_count": 0
    },
    "timestamp": "2026-11-03T10:00:00Z"
  }
}
```

Notes on optional fields:

- `metadata.filters` appears only when a filter or an offset was applied, and echoes it.
- `metadata.cached` appears (as `true`) only when the response came from the cache.
- `metadata.total_results` is the number of results in this response; `stats.deduped_count` is the number available. `raw_count` equals `deduped_count` plus the five rejection and merge counters.
- `content_error` appears on a result when extraction was attempted for it but produced no content, with the reason (blocked address, non-HTML page, timeout).

---

### 2. `GET /health` & `GET /healthz`

Health check probes for monitoring systems, load balancers, and orchestrators.

#### Response

```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 345.12,
  "timestamp": "2026-11-03T10:00:00Z"
}
```

---

## Error Handling

Standardized JSON error responses are returned on failures:

```json
{
  "error": "rate_limit_exceeded",
  "message": "rate limit exceeded: limit 10 req/min for tier 'free' (retry in 12s)",
  "code": 429
}
```

| HTTP Status | Error Token | Cause |
|---|---|---|
| `400 Bad Request` | `invalid_json` | Malformed JSON in request payload |
| `400 Bad Request` | `validation_error` | Missing or overlong query, invalid ISO codes, unknown provider, or invalid extract mode |
| `401 Unauthorized` | `unauthorized` | Missing API key when `PLEWTUS_AUTH_REQUIRED=true`, or invalid key |
| `404 Not Found` | `not_found` | Unknown endpoint |
| `405 Method Not Allowed` | `method_not_allowed` | Wrong HTTP method for the endpoint |
| `408 Request Timeout` | `request_cancelled` | The client cancelled the request |
| `429 Too Many Requests` | `rate_limit_exceeded` | Token bucket exhausted; check `Retry-After` header |
| `429 Too Many Requests` | `quota_exceeded` | Daily request quota reached; resets at 00:00:00 UTC |
| `500 Internal Error` | `search_error` | All providers failed, or an unexpected downstream failure |
| `504 Gateway Timeout` | `request_timeout` | The search exceeded its time budget |

Malformed or invalid requests are rejected before rate limits and quotas are charged; authentication is checked first.

---

## Example Requests

### 1. Simple Search
```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{"query": "kubernetes best practices", "limit": 5}'
```

### 2. Domain Whitelisting with Recency
```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "transformer architectures",
    "freshness": "month",
    "include_domains": ["arxiv.org", "paperswithcode.com"],
    "limit": 5
  }'
```

### 3. Search with AI Markdown Extraction
```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "go standard library net/http",
    "limit": 3,
    "extract": {
      "enabled": true,
      "mode": "markdown",
      "max_pages": 3,
      "max_chars_per_page": 5000
    }
  }'
```

### 4. Authenticated Request with Pro API Key
```bash
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pro-api-key" \
  -d '{"query": "distributed consensus algorithms"}'
```
