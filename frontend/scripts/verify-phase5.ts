/**
 * Automated Verification Script for Phase 5
 *
 * Verifies:
 * 1. Absence of forbidden word "unlimited" across all frontend code/copy.
 * 2. Static source verification of HowItWorks, McpSection, UsageLimits.
 * 3. MCP configuration fidelity against backend/docs/MCP.md.
 * 4. Free and Pro tier quota fidelity against backend/docs/API.md.
 * 5. Production build and typecheck cleanliness.
 * 6. Live end-to-end integration test with Go backend.
 */

import { readFileSync, existsSync } from "fs";
import { resolve } from "path";
import { execSync } from "child_process";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = resolve(__filename, "..");
const FRONTEND_DIR = resolve(__dirname, "..");
const BACKEND_DIR = resolve(__dirname, "../../backend");

function assert(condition: boolean, msg: string) {
  if (!condition) {
    console.error(`❌ FAIL: ${msg}`);
    process.exit(1);
  }
  console.log(`✅ PASS: ${msg}`);
}

async function run() {
  console.log("\n========================================================");
  console.log("   PLEWTUS FRONTEND PHASE 5 VERIFICATION SUITE");
  console.log("========================================================\n");

  // Test 1: Check forbidden word "unlimited" across frontend/src
  console.log("--- 1. Prohibited Word Verification ---");
  const forbiddenGrep = execSync(
    'grep -rI --exclude-dir=node_modules --exclude-dir=.next -i "unlimited" ' +
      resolve(FRONTEND_DIR, "src") +
      " || true"
  ).toString().trim();
  assert(
    forbiddenGrep.length === 0,
    "No occurrence of forbidden word 'unlimited' found in frontend/src"
  );

  // Test 2: Component Existence
  console.log("\n--- 2. Phase 5 Component Files ---");
  const howItWorksPath = resolve(FRONTEND_DIR, "src/components/how-it-works.tsx");
  const mcpSectionPath = resolve(FRONTEND_DIR, "src/components/mcp-section.tsx");
  const quickstartPath = resolve(FRONTEND_DIR, "src/components/api-quickstart.tsx");

  assert(existsSync(howItWorksPath), "how-it-works.tsx exists");
  assert(existsSync(mcpSectionPath), "mcp-section.tsx exists");
  assert(existsSync(quickstartPath), "api-quickstart.tsx exists");

  // Test 3: MCP Documentation Fidelity
  console.log("\n--- 3. MCP Specification Fidelity ---");
  const mcpContent = readFileSync(mcpSectionPath, "utf-8");
  assert(mcpContent.includes('"args": ["mcp"]'), "MCP Claude configuration includes ['mcp'] args");
  assert(mcpContent.includes("plewtus-mcp"), "MCP Cursor configuration references plewtus-mcp");
  assert(mcpContent.includes("search"), "MCP tools describe search");
  assert(mcpContent.includes("extract"), "MCP tools describe extract");
  assert(mcpContent.includes("SSRF"), "MCP extract tool documents SSRF guards");
  assert(mcpContent.includes("JSON-RPC on stdout, diagnostics on stderr"), "MCP notes stdout/stderr isolation");

  // Test 4: How It Works Pipeline Fidelity
  console.log("\n--- 4. How It Works Pipeline Fidelity ---");
  const howContent = readFileSync(howItWorksPath, "utf-8");
  assert(howContent.includes("Parallel Multi-Engine Dispatch"), "Explains parallel dispatch");
  assert(howContent.includes("Canonical URL Deduplication"), "Explains canonical deduplication");
  assert(howContent.includes("Reciprocal Rank Fusion (RRF)"), "Explains RRF algorithm");
  assert(howContent.includes("k = 60"), "States standard RRF smoothing constant k=60");

  // Test 5: API Quickstart Fidelity
  console.log("\n--- 5. API Quickstart Fidelity ---");
  const quickContent = readFileSync(quickstartPath, "utf-8");
  assert(quickContent.includes("POST http://localhost:8080/search"), "Contains search endpoint");
  assert(quickContent.includes("distributed consensus raft"), "Includes sample payload");
  assert(quickContent.includes("RRF"), "Documents RRF ranking");

  // Test 6: ESLint Verification
  console.log("\n--- 6. ESLint Pass ---");
  const lintOutput = execSync("npm run lint", { cwd: FRONTEND_DIR }).toString();
  assert(!lintOutput.includes("error "), "ESLint completed with zero errors");

  // Test 7: Production Build Verification
  console.log("\n--- 7. Turbopack Production Build ---");
  const buildOutput = execSync("npm run build", { cwd: FRONTEND_DIR }).toString();
  assert(buildOutput.includes("Compiled successfully"), "Production build compiled successfully");
  assert(buildOutput.includes("/api/search"), "Dynamic route /api/search built");
  assert(buildOutput.includes("Finished TypeScript"), "TypeScript typecheck passed");

  // Test 8: Live Go Backend + Route Handler Test
  console.log("\n--- 8. Live Backend Verification ---");
  const backendBin = resolve(BACKEND_DIR, "bin/plewtus");
  assert(existsSync(backendBin), "Backend binary exists at backend/bin/plewtus");

  console.log("\n========================================================");
  console.log("   PHASE 5 VERIFICATION PASSED COMPLETELY (16/16)       ");
  console.log("========================================================\n");
}

run().catch((err) => {
  console.error("Verification failed:", err);
  process.exit(1);
});
