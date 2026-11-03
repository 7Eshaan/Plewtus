"use client";

import { Gauge, KeyRound, Shield, Clock } from "lucide-react";

export function UsageLimits() {
  return (
    <section id="limits" className="w-full pt-8 pb-4 flex flex-col gap-6">
      <div className="flex flex-col gap-1.5 text-center sm:text-left">
        <div className="inline-flex items-center gap-1.5 text-xs font-mono text-zinc-500 uppercase tracking-wider">
          <span>Usage Quotas &amp; Rate Limits</span>
        </div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-zinc-950 dark:text-zinc-50 font-sans">
          Honest, Predictable API Limits
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400 max-w-2xl leading-relaxed">
          Search metasearch aggregation requires computational and networking resources. Plewtus enforces deterministic token-bucket rate limits and daily request quotas rather than opaque throttles.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Tier 1: Anonymous Free Tier */}
        <div className="p-5 sm:p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-4 shadow-2xs">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/60 flex items-center justify-center text-zinc-700 dark:text-zinc-300">
                <Gauge className="w-4 h-4" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-zinc-950 dark:text-zinc-100">
                  Free Tier (Default)
                </h3>
                <span className="text-xs font-mono text-zinc-500">
                  Anonymous access by client IP
                </span>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 font-medium">
              Active on Demo
            </span>
          </div>

          <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
            Every client IP receives free metasearch access without requiring an API key, credit card, or account registration. Ideal for experimentation, personal tooling, and local AI agent testing.
          </p>

          <div className="flex flex-col gap-2 pt-2 border-t border-zinc-100 dark:border-zinc-800/80 font-mono text-xs">
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Rate Limit:</span>
              <span className="font-semibold text-zinc-950 dark:text-zinc-100">10 requests / min</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Daily Quota:</span>
              <span className="font-semibold text-zinc-950 dark:text-zinc-100">100 requests / day</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Quota Reset:</span>
              <span>00:00:00 UTC daily</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Authentication:</span>
              <span>Not required</span>
            </div>
          </div>
        </div>

        {/* Tier 2: Developer Pro Tier */}
        <div className="p-5 sm:p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-[#121215] flex flex-col gap-4 shadow-2xs">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-zinc-900 dark:bg-white text-white dark:text-zinc-950 flex items-center justify-center">
                <KeyRound className="w-4 h-4" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-zinc-950 dark:text-zinc-100">
                  Developer Pro Tier
                </h3>
                <span className="text-xs font-mono text-zinc-500">
                  Key-authenticated access
                </span>
              </div>
            </div>
            <span className="text-[11px] font-mono px-2 py-0.5 rounded-full bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700/60 font-medium">
              API Key
            </span>
          </div>

          <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
            Supplying a valid API key via <code className="text-[11px] font-mono">Authorization: Bearer &lt;key&gt;</code> or <code className="text-[11px] font-mono">X-API-Key</code> unlocks higher capacity suited for production agent harnesses and pipeline processing.
          </p>

          <div className="flex flex-col gap-2 pt-2 border-t border-zinc-100 dark:border-zinc-800/80 font-mono text-xs">
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Rate Limit:</span>
              <span className="font-semibold text-zinc-950 dark:text-zinc-100">60 requests / min</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Daily Quota:</span>
              <span className="font-semibold text-zinc-950 dark:text-zinc-100">2,000 requests / day</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Quota Reset:</span>
              <span>00:00:00 UTC daily</span>
            </div>
            <div className="flex items-center justify-between text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">Concurrency:</span>
              <span>Multi-goroutine dispatch</span>
            </div>
          </div>
        </div>
      </div>

      {/* Response Header Transparency Banner */}
      <div className="p-4 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/60 dark:bg-zinc-900/30 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs font-mono text-zinc-600 dark:text-zinc-400">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-zinc-400 flex-shrink-0" />
          <span>
            Every response includes standard usage headers: <code className="text-zinc-900 dark:text-zinc-200 font-semibold">X-RateLimit-*</code>, <code className="text-zinc-900 dark:text-zinc-200 font-semibold">X-Quota-*</code>, and <code className="text-zinc-900 dark:text-zinc-200 font-semibold">Retry-After</code> on 429.
          </span>
        </div>
        <div className="flex items-center gap-1.5 text-zinc-400 flex-shrink-0">
          <Shield className="w-3.5 h-3.5" />
          <span>Token Bucket Algorithm</span>
        </div>
      </div>
    </section>
  );
}
