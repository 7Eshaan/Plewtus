import { Header } from "@/components/header";
import { Playground } from "@/components/playground";
import { HowItWorks } from "@/components/how-it-works";
import { McpSection } from "@/components/mcp-section";
import { ApiQuickstart } from "@/components/api-quickstart";
import { AsciiHeroField } from "@/components/ascii-hero-field";

const HERO_ACCENT_COLOR: [number, number, number] = [135 / 255, 0, 1];

const SECTION_LINKS = [
  { href: "#demo", label: "Search Demo" },
  { href: "#how-it-works", label: "How It Works" },
  { href: "#mcp", label: "MCP Setup" },
  { href: "#quickstart", label: "API Quickstart" },
];

export default function HomePage() {
  return (
    <div className="min-h-screen flex flex-col text-zinc-900 dark:text-zinc-100">
      <Header />

      <main className="flex-1 w-full flex flex-col">
        {/* Full Viewport Hero with Interactive Skewed Background Boxes */}
        <section
          id="demo"
          className="relative w-full min-h-[calc(100vh-3.5rem)] min-h-[calc(100dvh-3.5rem)] flex flex-col items-center justify-center overflow-hidden scroll-mt-14"
        >
          {/* Interactive ASCII fluid background in the brand accent */}
          <AsciiHeroField word="PLEWTUS METASEARCH " accentColor={HERO_ACCENT_COLOR} />

          {/* Centered Hero Content Container */}
          <div className="relative z-10 w-full max-w-4xl mx-auto px-4 py-8 sm:py-12 flex flex-col items-center text-center gap-5">
            <div className="flex flex-col items-center gap-2.5 max-w-2xl">
              <h1 className="text-3xl sm:text-5xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans text-balance">
                Federated Search for AI Agents
              </h1>

              <p className="text-sm sm:text-base text-zinc-500 dark:text-zinc-400 max-w-xl text-balance leading-relaxed">
                Query DuckDuckGo, Bing, and Yahoo at once. Deduplicated canonical URLs,
                off-topic results removed, and relevance-aware consensus ranking.
              </p>
            </div>

            {/* Playground card with a subtle ambient brand glow */}
            <div className="relative w-full max-w-4xl mx-auto pt-2">
              <div
                className="absolute -inset-3 bg-gradient-to-r from-brand/10 via-brand/5 to-brand/10 rounded-3xl blur-2xl opacity-70 dark:opacity-40 pointer-events-none -z-10"
                aria-hidden="true"
              />
              <Playground />
            </div>
          </div>
        </section>

        {/* 2nd Page Container: Documentation, MCP & Rate Limits */}
        <div className="w-full max-w-4xl mx-auto px-4 flex flex-col gap-10 pb-16">
          {/* Section Divider - Appears on 2nd page scroll */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80" />

          {/* How It Works Section */}
          <HowItWorks />

          {/* Section Divider */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

          {/* Model Context Protocol (MCP) Section */}
          <McpSection />

          {/* Section Divider */}
          <div className="w-full border-t border-zinc-200/80 dark:border-zinc-800/80 my-2" />

          {/* API Quickstart & Code Integration Section */}
          <ApiQuickstart />
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full border-t border-zinc-200 dark:border-zinc-800/80 py-8 text-xs font-mono text-zinc-500 dark:text-zinc-500 bg-zinc-50/50 dark:bg-zinc-950/40">
        <div className="max-w-4xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-zinc-900 dark:text-zinc-100">Plewtus</span>
            <span aria-hidden="true">•</span>
            <span>Federated search for AI agents</span>
          </div>

          <nav className="flex flex-wrap items-center justify-center gap-4 text-xs" aria-label="Footer">
            {SECTION_LINKS.map((link) => (
              <a key={link.href} href={link.href} className="hover:text-brand dark:hover:text-brand-light transition-colors">
                {link.label}
              </a>
            ))}
          </nav>

          <div className="text-[11px] text-zinc-400 dark:text-zinc-600">
            DuckDuckGo • Bing • Yahoo
          </div>
        </div>
      </footer>
    </div>
  );
}
