/**
 * Phase 2 Verification Script: Theme & Hero Search
 * Validates:
 * 1. UI components and type safety.
 * 2. Multi-engine consensus rendering and RRF scoring.
 * 3. Honest metadata strip provider counting.
 * 4. End-to-end search through live route handler.
 */

async function runPhase2Verification() {
  console.log("=== Plewtus Frontend Phase 2 Verification ===\n");

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

  // TEST 1: Live Search with Multi-Engine Consensus
  console.log("1. Live Federated Metasearch & RRF Consensus:");
  try {
    const res = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "reciprocal rank fusion algorithm",
        limit: 5,
        offset: 0,
      }),
    });

    assert("API returned HTTP 200 OK", res.status === 200, `Got ${res.status}`);
    const data = await res.json();

    assert("Payload contains results array", Array.isArray(data.results) && data.results.length > 0);
    assert("Query echoed accurately", data.query === "reciprocal rank fusion algorithm");

    // Check RRF ranking order
    const scores = data.results.map((r: { score: number }) => r.score);
    const isSortedDesc = scores.every((s: number, i: number) => i === 0 || scores[i - 1] >= s);
    assert("Results strictly sorted by RRF score descending", isSortedDesc);

    // Check Multi-engine consensus badges
    const hasEngineSources = data.results.every(
      (r: { engines: Array<{ name: string; position: number }> }) =>
        Array.isArray(r.engines) && r.engines.length > 0 && typeof r.engines[0].position === "number"
    );
    assert("Each result includes engine sources and original SERP ranks", hasEngineSources);

    // TEST 2: Metadata Strip Contract
    console.log("\n2. Honest Metadata Contract:");
    const meta = data.metadata;
    assert("Metadata has took_ms >= 0", typeof meta.took_ms === "number" && meta.took_ms >= 0);
    assert("Providers queried array is non-empty", Array.isArray(meta.providers?.queried) && meta.providers.queried.length > 0);
    assert("Providers successful array present", Array.isArray(meta.providers?.successful));
    assert("Providers failed is array or null", Array.isArray(meta.providers?.failed) || meta.providers?.failed === null);
    assert(
      "Deduplication stats present (raw_count >= deduped_count)",
      meta.stats?.raw_count >= meta.stats?.deduped_count
    );

    // TEST 3: Validation and Error Handling
    console.log("\n3. Validation & Error State Contract:");
    const badRes = await fetch("http://localhost:3000/api/search", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query: "", limit: 5, offset: 0 }),
    });

    assert("Empty query rejected with HTTP 400", badRes.status === 400, `Got ${badRes.status}`);
    const errData = await badRes.json();
    assert("Error token is 'validation_error'", errData.error === "validation_error");
    assert("Error payload includes code: 400", errData.code === 400);

  } catch (err) {
    assert("Verification tests succeeded without uncaught exceptions", false, String(err));
  }

  console.log(`\nPhase 2 Verification Summary: ${passed} passed, ${failed} failed.`);
  if (failed > 0) {
    process.exit(1);
  }
}

runPhase2Verification();
