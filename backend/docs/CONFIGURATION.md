# Plewtus Configuration Reference

Plewtus is configured via environment variables. Sane defaults are provided so the service operates out of the box with zero required configuration.

For local development, Plewtus also reads an optional `.env` file from the working directory (or the path in `PLEWTUS_ENV_FILE`). Values from the real environment always take precedence, and container images exclude `.env` files.

---

## Environment Variables

| Variable | Type | Default | Description |
|---|---|---|---|
| `PLEWTUS_HOST` | `string` | `0.0.0.0` | IP interface to bind HTTP server. |
| `PLEWTUS_PORT` | `integer` | `8080` | Port to listen on. |
| `PLEWTUS_ENV` | `string` | `development` | Environment mode (`development` or `production`). In production, logs format as JSON. |
| `PLEWTUS_LOG_LEVEL` | `string` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`). |
| `PLEWTUS_DEFAULT_PROVIDERS` | `string` | `duckduckgo,bing,yahoo` | Comma-separated list of active providers queried by default. Available: `duckduckgo`, `bing`, `yahoo`, `marginalia`. |
| `PLEWTUS_MARGINALIA_API_KEY` | `string` | `public` | Key for the Marginalia Search API. `public` is shared and heavily rate limited; free personal keys are available from Marginalia. Free-key results are CC-BY-NC-SA 4.0. |
| `PLEWTUS_DEFAULT_TIMEOUT_MS` | `integer` | `6000` | Per-provider scrape timeout in milliseconds (default 6s). |
| `PLEWTUS_EXTRACT_TIMEOUT_MS` | `integer` | `8000` | Time budget for the page-extraction stage of a search. A single page fetch is capped at the smaller of this value and 5 seconds, so one slow site cannot hold back the others. |
| `PLEWTUS_STRAGGLER_GRACE_MS` | `integer` | `1500` | When three or more engines are queried and all but one have returned results, how long to keep waiting for the last one before answering without it. It bounds tail latency when one engine hangs; `0` always waits for the full timeout. A cut-off engine is listed under `providers.failed` and does not count against its circuit breaker. |
| `PLEWTUS_TRUSTED_PROXIES` | `string` | `""` | Comma-separated IP addresses/CIDRs allowed to supply `X-Forwarded-For`. When empty, forwarded headers are ignored and the socket peer identifies anonymous callers. |
| `PLEWTUS_DEFAULT_LIMIT` | `integer` | `10` | Default results count when `limit` is omitted. |
| `PLEWTUS_MAX_LIMIT` | `integer` | `50` | Maximum allowed results limit in a single request (capped at the 50-result window). |
| `PLEWTUS_USER_AGENT` | `string` | Standard Chrome User-Agent | Polite desktop User-Agent string sent to search engines and extraction targets. |

---

## Caching Configuration

| Variable | Type | Default | Description |
|---|---|---|---|
| `PLEWTUS_CACHE_ENABLED` | `boolean` | `true` | Enables in-memory LRU search response cache. |
| `PLEWTUS_CACHE_TTL_SECONDS` | `integer` | `600` | Time-to-live for cached search results in seconds (default 10 minutes). |
| `PLEWTUS_CACHE_MAX_ENTRIES` | `integer` | `1000` | Maximum number of cached queries before LRU eviction. |

---

## Usage Management & Authentication

| Variable | Type | Default | Description |
|---|---|---|---|
| `PLEWTUS_AUTH_REQUIRED` | `boolean` | `false` | When `true`, requests without a registered API key are rejected (`401 Unauthorized`). |
| `PLEWTUS_API_KEYS` | `string` | `""` | Comma-separated list of valid API keys. Can specify tier mapping (e.g. `key1:pro,key2:free`). Untagged keys default to `pro`. |
| `PLEWTUS_FREE_TIER_RATE_LIMIT` | `integer` | `10` | Requests allowed per minute for anonymous clients and free tier keys. |
| `PLEWTUS_FREE_TIER_DAILY_QUOTA` | `integer` | `100` | Requests allowed per UTC calendar day for the free tier (0 = unlimited). |
| `PLEWTUS_PRO_TIER_RATE_LIMIT` | `integer` | `60` | Requests allowed per minute for pro tier API keys. |
| `PLEWTUS_PRO_TIER_DAILY_QUOTA` | `integer` | `2000` | Requests allowed per UTC calendar day for pro tier keys. |

---

## Reliability & Circuit Breaking

| Variable | Type | Default | Description |
|---|---|---|---|
| `PLEWTUS_CIRCUIT_BREAKER_THRESHOLD` | `integer` | `3` | Consecutive failures required to trip a search engine's circuit breaker to `Open`. Errors, captchas, and responses discarded as off-topic all count. |
| `PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS` | `integer` | `30000` | Cooldown in milliseconds (default 30s) before an open breaker allows a trial probe. |

---

## Example `.env` File

```bash
# Server settings
PLEWTUS_HOST=0.0.0.0
PLEWTUS_PORT=8080
PLEWTUS_ENV=production
PLEWTUS_LOG_LEVEL=info

# Providers & Timeouts
PLEWTUS_DEFAULT_PROVIDERS=duckduckgo,bing,yahoo
PLEWTUS_DEFAULT_TIMEOUT_MS=6000
PLEWTUS_EXTRACT_TIMEOUT_MS=8000
PLEWTUS_STRAGGLER_GRACE_MS=1500
PLEWTUS_TRUSTED_PROXIES=10.0.0.0/8
PLEWTUS_DEFAULT_LIMIT=10
PLEWTUS_MAX_LIMIT=50

# Caching
PLEWTUS_CACHE_ENABLED=true
PLEWTUS_CACHE_TTL_SECONDS=600
PLEWTUS_CACHE_MAX_ENTRIES=2000

# Authentication & Rate Limiting
PLEWTUS_AUTH_REQUIRED=false
PLEWTUS_API_KEYS=sk_live_agent123:pro,sk_test_free456:free
PLEWTUS_FREE_TIER_RATE_LIMIT=10
PLEWTUS_FREE_TIER_DAILY_QUOTA=100
PLEWTUS_PRO_TIER_RATE_LIMIT=60
PLEWTUS_PRO_TIER_DAILY_QUOTA=2000

# Circuit Breakers
PLEWTUS_CIRCUIT_BREAKER_THRESHOLD=3
PLEWTUS_CIRCUIT_BREAKER_COOLDOWN_MS=30000
```
