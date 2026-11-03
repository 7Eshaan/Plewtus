/**
 * Phase 3 Verification Script: Options & Features
 * Validates:
 * 1. AI Content extraction in markdown and highlights mode.
 * 2. Provider subset filtering (querying selected engines only).
 * 3. Domain inclusion / exclusion filtering.
 * 4. Freshness and metadata filter reflection.
 * 5. Strict prohibition of "unlimited".
 */

import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function runPhase3Verification() {
  console.log("=== Plewtus Frontend Phase 3 Verification ===\n");

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

  // TEST 1: Markdown Extraction
  console.log("1. Live Markdown Content Extraction Test:");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "go standard library net/http",
        limit: 2,
        offset: 0,
        extract: {
          enabled: true,
          mode: "markdown",
          max_pages: 1,
          max_chars_per_page: 2000,
        },
      }),
    });

    assert("Markdown extraction request returned 200 OK", res.status === 200);
    const data = await res.json();
    const firstResult = data.results?.[0];
    assert("First result has content object", Boolean(firstResult?.content));
    assert(
      "Content has non-empty markdown string",
      typeof firstResult?.content?.markdown === "string" && firstResult.content.markdown.length > 50
    );
    assert(
      "Content has extracted_at ISO timestamp",
      typeof firstResult?.content?.extracted_at === "string" && firstResult.content.extracted_at.length > 0
    );
  } catch (err) {
    assert("Markdown extraction succeeded", false, String(err));
  }

  // TEST 2: Highlights Extraction
  console.log("\n2. Live Highlights Extraction Test:");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "go standard library net/http",
        limit: 2,
        offset: 0,
        extract: {
          enabled: true,
          mode: "highlights",
          max_pages: 1,
          max_chars_per_page: 2000,
        },
      }),
    });

    assert("Highlights extraction request returned 200 OK", res.status === 200);
    const data = await res.json();
    const firstResult = data.results?.[0];
    assert(
      "Content has highlights array with items",
      Array.isArray(firstResult?.content?.highlights) && firstResult.content.highlights.length > 0
    );
  } catch (err) {
    assert("Highlights extraction succeeded", false, String(err));
  }

  // TEST 3: Provider Filtering (subset)
  console.log("\n3. Provider Chips Filtering Test:");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "golang concurrency patterns",
        limit: 2,
        offset: 0,
        providers: ["bing"],
      }),
    });

    assert("Provider filter request returned 200 OK", res.status === 200);
    const data = await res.json();
    const queried = data.metadata?.providers?.queried;
    assert("Queried providers strictly equals ['bing']", Array.isArray(queried) && queried.length === 1 && queried[0] === "bing");
    assert(
      "DuckDuckGo and Yahoo were NOT queried",
      !queried?.includes("duckduckgo") && !queried?.includes("yahoo")
    );
  } catch (err) {
    assert("Provider filter succeeded", false, String(err));
  }

  // TEST 4: Domain Inclusion and Recency
  console.log("\n4. Domain Whitelisting & Recency Filters Test:");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "concurrency patterns",
        limit: 2,
        offset: 0,
        freshness: "month",
        include_domains: ["go.dev"],
      }),
    });

    assert("Domain filter request returned 200 OK", res.status === 200);
    const data = await res.json();
    assert(
      "All results match included domain 'go.dev'",
      data.results?.every((r: { url: string }) => r.url.includes("go.dev"))
    );
    assert(
      "Metadata filters reflect freshness: 'month'",
      data.metadata?.filters?.freshness === "month"
    );
    assert(
      "Metadata filters reflect include_domains: ['go.dev']",
      data.metadata?.filters?.include_domains?.includes("go.dev")
    );
  } catch (err) {
    assert("Domain filter succeeded", false, String(err));
  }

  // TEST 5: Prohibition of "unlimited"
  console.log("\n5. Rule Verification: Zero 'unlimited' strings in codebase:");
  const srcDir = path.resolve(__dirname, "../src");
  let foundUnlimited = false;

  function scanForUnlimited(dir: string) {
    const files = fs.readdirSync(dir);
    for (const file of files) {
      const fullPath = path.join(dir, file);
      const stat = fs.statSync(fullPath);
      if (stat.isDirectory()) {
        scanForUnlimited(fullPath);
      } else if (file.endsWith(".tsx") || file.endsWith(".ts") || file.endsWith(".css")) {
        const content = fs.readFileSync(fullPath, "utf-8").toLowerCase();
        if (content.includes("unlimited")) {
          foundUnlimited = true;
          console.error(`Found 'unlimited' in ${fullPath}`);
        }
      }
    }
  }
  scanForUnlimited(srcDir);
  assert("No 'unlimited' word in frontend/src", !foundUnlimited);

  console.log(`\nPhase 3 Verification Summary: ${passed} passed, ${failed} failed.`);
  if (failed > 0) {
    process.exit(1);
  }
}

runPhase3Verification();
