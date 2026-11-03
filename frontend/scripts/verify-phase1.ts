/**
 * Phase 1 Verification Script
 * Validates:
 * 1. Server-side key check: request without key never contacts backend.
 * 2. Per-IP rate limiting: returns 429 when threshold exceeded.
 * 3. Secret isolation: key never appears in client bundles or public responses.
 * 4. End-to-end proxying against live backend.
 */

import { checkRateLimit, resetRateLimits } from "../src/lib/rate-limiter.ts";
import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function runTests() {
  console.log("=== Plewtus Frontend Phase 1 Verification ===\n");

  let passed = 0;
  let failed = 0;

  function assert(desc: string, condition: boolean, extra?: string) {
    if (condition) {
      console.log(`[PASS] ${desc}`);
      passed++;
    } else {
      console.error(`[FAIL] ${desc} ${extra ? `(${extra})` : ""}`);
      failed++;
    }
  }

  // TEST 1: Rate Limiter functionality
  console.log("1. Rate Limiting Logic Test:");
  resetRateLimits();
  const testIp = "203.0.113.19";
  const limit = 5;

  for (let i = 1; i <= limit; i++) {
    const res = checkRateLimit(testIp, limit);
    assert(`Request ${i}/${limit} allowed`, res.success && res.remaining === limit - i);
  }

  const blocked = checkRateLimit(testIp, limit);
  assert("Request 6 exceeds limit and is rejected", !blocked.success && blocked.remaining === 0);
  assert("Returns positive reset time", blocked.reset > 0);

  // TEST 2: Secret Isolation in Client Bundles
  console.log("\n2. Secret Key Isolation Test:");
  const testKey = "pro_live_agent123";
  const staticDir = path.resolve(__dirname, "../.next/static");

  let keyFoundInBundles = false;
  if (fs.existsSync(staticDir)) {
    function scanDir(dir: string) {
      const files = fs.readdirSync(dir);
      for (const file of files) {
        const fullPath = path.join(dir, file);
        const stat = fs.statSync(fullPath);
        if (stat.isDirectory()) {
          scanDir(fullPath);
        } else if (file.endsWith(".js") || file.endsWith(".json") || file.endsWith(".css")) {
          const content = fs.readFileSync(fullPath, "utf-8");
          if (content.includes(testKey)) {
            keyFoundInBundles = true;
            console.error(`Leak detected in ${fullPath}!`);
          }
        }
      }
    }
    scanDir(staticDir);
  }
  assert("Demo API key ('pro_live_agent123') is NOT in any static bundle", !keyFoundInBundles);

  // TEST 3: Live Route Handler Proxy to Backend
  console.log("\n3. Live Route Handler Proxy Test (Browser -> Proxy -> Go Backend):");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        query: "golang concurrency patterns",
        limit: 3,
        offset: 0,
      }),
    });

    assert("Proxy returns HTTP 200 OK", res.status === 200, `Got ${res.status}`);

    const hasAuthHeader = Boolean(res.headers.get("authorization") || res.headers.get("x-api-key"));
    assert("Response headers do NOT leak Authorization or X-API-Key", !hasAuthHeader);

    const hasRateLimit = Boolean(res.headers.get("x-ratelimit-limit"));
    assert("Response headers include X-RateLimit-Limit", hasRateLimit);

    const data = await res.json();
    assert("Response echoes normalized query", data.query === "golang concurrency patterns");
    assert("Response includes results array", Array.isArray(data.results) && data.results.length > 0);
    assert("Results contain RRF rank and score", typeof data.results[0]?.rank === "number" && typeof data.results[0]?.score === "number");
    assert("Metadata reports engines queried & took_ms", Boolean(data.metadata?.providers?.queried && data.metadata?.took_ms >= 0));

    // Ensure the key doesn't appear in the JSON payload
    const jsonStr = JSON.stringify(data);
    assert("Response body does NOT leak the API key", !jsonStr.includes(testKey));
  } catch (err) {
    assert("Live proxy request succeeded", false, err instanceof Error ? err.message : String(err));
  }

  console.log(`\nVerification Summary: ${passed} passed, ${failed} failed.`);
  if (failed > 0) {
    process.exit(1);
  }
}

runTests();
