import { Cpu, GitMerge, Award, ArrowRight } from "lucide-react";

export function HowItWorks() {
  return (
    <section id="how-it-works" className="w-full pt-8 pb-4 flex flex-col gap-6 scroll-mt-16">
      <div className="flex flex-col gap-1.5 text-center sm:text-left">
        <div className="inline-flex items-center gap-1.5 text-xs font-mono text-brand dark:text-brand-light uppercase tracking-wider font-semibold">
          <span>Architecture &amp; Pipeline</span>
        </div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans">
          How Plewtus Aggregates the Web
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400 max-w-2xl leading-relaxed">
          Standard search APIs query a single engine and return raw, ad-heavy results. Plewtus operates as a metasearch aggregator designed specifically for autonomous AI agents.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Step 1: Parallel Dispatch */}
        <div className="p-5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-3 shadow-2xs hover:border-zinc-400 dark:hover:border-zinc-700 transition-colors">
          <div className="w-8 h-8 rounded-lg bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/60 flex items-center justify-center text-zinc-900 dark:text-zinc-100">
            <Cpu className="w-4 h-4" />
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500 font-semibold">01</span>
              <h3 className="text-sm font-semibold text-zinc-950 dark:text-zinc-100">
                Parallel Multi-Engine Dispatch
              </h3>
            </div>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Dispatches concurrent goroutines across DuckDuckGo, Bing, and Yahoo with bounded timeouts and independent circuit breakers. If one provider encounters a challenge or times out, healthy responses are preserved.
            </p>
          </div>
          <div className="mt-auto pt-3 border-t border-zinc-100 dark:border-zinc-800/80 flex flex-wrap gap-1.5 text-[10px] font-mono text-zinc-500">
            <span className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">DuckDuckGo</span>
            <span className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">Bing</span>
            <span className="px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800">Yahoo</span>
          </div>
        </div>

        {/* Step 2: Canonical URL Deduplication */}
        <div className="p-5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-3 shadow-2xs hover:border-zinc-400 dark:hover:border-zinc-700 transition-colors">
          <div className="w-8 h-8 rounded-lg bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/60 flex items-center justify-center text-zinc-900 dark:text-zinc-100">
            <GitMerge className="w-4 h-4" />
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs text-zinc-400 dark:text-zinc-500 font-semibold">02</span>
              <h3 className="text-sm font-semibold text-zinc-950 dark:text-zinc-100">
                Canonical URL Deduplication
              </h3>
            </div>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Unwraps search redirect wrappers (<code className="text-[11px] font-mono">/ck/a</code>, <code className="text-[11px] font-mono">/RU=</code>, <code className="text-[11px] font-mono">/l/?uddg</code>) and strips tracking tokens (<code className="text-[11px] font-mono">utm_*</code>, <code className="text-[11px] font-mono">gclid</code>, <code className="text-[11px] font-mono">fbclid</code>), merging identical pages into a single consensus entry.
            </p>
          </div>
          <div className="mt-auto pt-3 border-t border-zinc-100 dark:border-zinc-800/80 flex items-center gap-1.5 text-[10px] font-mono text-zinc-500">
            <span>Canonical hash normalization</span>
            <ArrowRight className="w-3 h-3 text-zinc-400" />
            <span className="text-emerald-500">One entry per page</span>
          </div>
        </div>

        {/* Step 3: Query-aware rank fusion */}
        <div className="p-5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-3 shadow-2xs hover:border-brand/40 dark:hover:border-brand/40 transition-colors">
          <div className="w-8 h-8 rounded-lg bg-brand/10 dark:bg-brand/20 border border-brand/30 flex items-center justify-center text-brand dark:text-brand-light">
            <Award className="w-4 h-4" />
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs text-brand dark:text-brand-light font-semibold">03</span>
              <h3 className="text-sm font-semibold text-zinc-950 dark:text-zinc-100">
                Query-aware rank fusion
              </h3>
            </div>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
              Removes off-topic results (and discards decoy result pages some engines serve to bots), then blends query relevance with multi-engine consensus. Relevance favors query coverage, title matches, and exact phrases; RRF combines provider positions with <code className="text-[11px] font-mono">k = 60</code>.
            </p>
          </div>
          <div className="mt-auto pt-3 border-t border-zinc-100 dark:border-zinc-800/80 font-mono text-[10px] text-brand dark:text-brand-light bg-brand/5 dark:bg-brand/10 border border-brand/15 p-2 rounded">
            rank = 0.8 &times; relevance + 0.2 &times; normalized RRF
          </div>
        </div>
      </div>
    </section>
  );
}
