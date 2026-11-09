/** Checks for the search proxy's request sanitizer and client identification. */

import { sanitizeSearchRequest, DEMO_LIMITS } from "../src/lib/sanitize-request.ts";
import { getClientIp, checkRateLimit, resetRateLimits } from "../src/lib/rate-limiter.ts";

function assert(condition: boolean, message: string): void {
  if (!condition) throw new Error(`FAIL: ${message}`);
  console.log(`PASS: ${message}`);
}

function request(headers: Record<string, string>): Request {
  return new Request("http://localhost/api/search", { method: "POST", headers });
}

// Sanitizer
const clean = sanitizeSearchRequest({
  query: "  golang  ",
  limit: 500,
  providers: ["bing", "evil", "yahoo"],
  freshness: "decade",
  country: "US",
  language: "english",
  include_domains: Array.from({ length: 20 }, (_, i) => `d${i}.com`),
  extract: { enabled: true, mode: "markdown", max_pages: 10, max_chars_per_page: 20000 },
  injected: "nope",
});
assert(clean.ok, "valid request is accepted");
if (clean.ok) {
  const r = clean.request as Record<string, unknown>;
  assert(r.query === "golang", "query is trimmed");
  assert(r.limit === DEMO_LIMITS.results, "limit is clamped");
  assert(JSON.stringify(r.providers) === JSON.stringify(["bing", "yahoo"]), "unknown providers are dropped");
  assert(r.freshness === undefined, "invalid freshness is dropped");
  assert(r.country === "us" && r.language === undefined, "ISO codes are validated");
  assert((r.include_domains as string[]).length === DEMO_LIMITS.domains, "domain lists are capped");
  const ex = r.extract as Record<string, unknown>;
  assert(ex.max_pages === DEMO_LIMITS.extractPages && ex.max_chars_per_page === DEMO_LIMITS.extractChars, "extraction is bounded");
  assert(!("injected" in r), "unknown fields are not forwarded");
}
assert(!sanitizeSearchRequest({ query: "  " }).ok, "blank query is rejected");
assert(!sanitizeSearchRequest([]).ok, "non-object body is rejected");
assert(!sanitizeSearchRequest({ query: "x".repeat(DEMO_LIMITS.queryChars + 1) }).ok, "overlong query is rejected");

// Client identification
delete process.env.VERCEL;
delete process.env.PLEWTUS_TRUST_PROXY_HEADERS;
assert(getClientIp(request({ "x-forwarded-for": "1.2.3.4" })) === "untrusted-shared", "spoofable headers are ignored by default");
resetRateLimits();
const codes = ["10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"].map(
  (ip) => checkRateLimit(getClientIp(request({ "x-forwarded-for": ip })), 3).success
);
assert(codes.join() === "true,true,true,false", "rotating X-Forwarded-For no longer bypasses the limit");

process.env.PLEWTUS_TRUST_PROXY_HEADERS = "true";
assert(getClientIp(request({ "x-forwarded-for": "6.6.6.6, 9.9.9.9" })) === "9.9.9.9", "trusted proxy: right-most hop is the client");
delete process.env.PLEWTUS_TRUST_PROXY_HEADERS;

process.env.VERCEL = "1";
assert(getClientIp(request({ "x-vercel-forwarded-for": "8.8.8.8" })) === "8.8.8.8", "Vercel platform header is trusted on Vercel");
delete process.env.VERCEL;

console.log("Proxy checks passed.");
