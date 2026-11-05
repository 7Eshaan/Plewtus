/** Static smoke checks for the current frontend sections and API example. */

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = resolve(fileURLToPath(import.meta.url), "..");
const frontendDir = resolve(scriptDir, "..");
const quickstart = readFileSync(resolve(frontendDir, "src/components/api-quickstart.tsx"), "utf8");
const sampleResponse = quickstart.split("const SAMPLE_RESPONSE = `")[1]?.split("`;\n\nexport function")[0] ?? "";
const mcp = readFileSync(resolve(frontendDir, "src/components/mcp-section.tsx"), "utf8");
const page = readFileSync(resolve(frontendDir, "src/app/page.tsx"), "utf8");

function assert(condition: boolean, message: string): void {
  if (!condition) throw new Error(`FAIL: ${message}`);
  console.log(`PASS: ${message}`);
}

assert(page.includes("<ApiQuickstart />"), "the page renders the API quickstart");
assert(page.includes("<McpSection />"), "the page renders the MCP section");
assert(quickstart.includes('"total_results": 1'), "the sample response includes total_results");
assert(quickstart.includes('"stats": {'), "the sample response nests pipeline counts under stats");
assert(quickstart.includes('"timestamp":'), "the sample response includes metadata timestamp");
assert(!sampleResponse.includes('"mode": "markdown"'), "content example follows ExtractedContent schema");
assert(mcp.includes("plewtus-mcp"), "MCP quickstart references the dedicated executable");

console.log("Frontend static checks passed.");
