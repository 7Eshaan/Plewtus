import { Header } from "@/components/header";
import { Playground } from "@/components/playground";
import { HowItWorks } from "@/components/how-it-works";
import { McpSection } from "@/components/mcp-section";
import { ApiQuickstart } from "@/components/api-quickstart";
import { SiteFooter } from "@/components/site-footer";
import { HorizonBackground } from "@/components/horizon-background";
import { DeepSpaceBackground } from "@/components/deep-space-background";
import { WavyDivider } from "@/components/ui/wavy-divider";

export default function HomePage() {
  return (
    <div className="flex min-h-screen flex-col text-on-surface">
      <Header />

      <main className="flex w-full flex-1 flex-col">
        {/* Full-viewport hero over the planet-horizon background */}
        <section
          id="demo"
          className="relative flex min-h-[calc(100vh-4rem)] min-h-[calc(100dvh-4rem)] w-full scroll-mt-16 flex-col items-center justify-center overflow-hidden"
        >
          {/* Starfield sky with a glowing planet rim that follows the pointer */}
          <HorizonBackground />

          <div className="relative z-10 mx-auto flex w-full max-w-4xl flex-col items-center gap-6 px-4 py-8 text-center sm:py-12">
            <div className="flex max-w-2xl flex-col items-center gap-3">
              {/* Display type in its M3 Expressive emphasized weight. */}
              <h1 className="text-balance text-display-sm font-medium text-on-surface sm:text-display-md">
                Federated Search for AI Agents
              </h1>

              <p className="max-w-xl text-balance text-body-lg text-on-surface-variant">
                Connect your agents to the web with one search API. Clean, deduplicated results ranked by consensus, with full page
                content when you need it.
              </p>
            </div>

            {/* Playground card with a soft brand glow behind it */}
            <div className="relative mx-auto w-full max-w-4xl pt-2">
              <div
                className="pointer-events-none absolute -inset-4 -z-10 rounded-corner-2xl bg-primary-container/25 opacity-70 blur-3xl dark:opacity-40"
                aria-hidden="true"
              />
              <Playground />
            </div>
          </div>
        </section>

        {/* Documentation, MCP and API sections, over the space beyond the planet */}
        <div className="relative w-full">
          <DeepSpaceBackground />

          <div className="relative z-10 mx-auto flex w-full max-w-5xl flex-col px-4 sm:px-6">
            <HowItWorks />

            <WavyDivider />

            <McpSection />

            <WavyDivider />

            <ApiQuickstart />

            <WavyDivider className="pb-8" />
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
