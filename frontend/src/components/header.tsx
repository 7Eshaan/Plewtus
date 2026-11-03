"use client";

import { ThemeToggle } from "./theme-toggle";
import { Sparkles } from "lucide-react";

export function Header() {
  return (
    <header className="w-full border-b border-zinc-200 dark:border-zinc-800/80 bg-white/80 dark:bg-[#09090b]/80 backdrop-blur-md sticky top-0 z-40 transition-colors">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        {/* Brand */}
        <a
          href="#demo"
          className="flex items-center gap-3 focus:outline-none focus:ring-1 focus:ring-zinc-400 dark:focus:ring-zinc-600 rounded-md"
          aria-label="Plewtus Metasearch Home"
        >
          <div className="w-7 h-7 rounded-md bg-zinc-900 dark:bg-white text-white dark:text-zinc-950 flex items-center justify-center font-mono font-bold text-sm select-none shadow-xs">
            P
          </div>
          <div className="flex items-center gap-2">
            <span className="font-semibold text-sm tracking-tight text-zinc-950 dark:text-zinc-50">
              Plewtus
            </span>
            <span className="text-[10px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-900 text-zinc-600 dark:text-zinc-400">
              Metasearch
            </span>
          </div>
        </a>

        {/* Center navigation links */}
        <nav className="hidden md:flex items-center gap-6 text-xs font-mono text-zinc-600 dark:text-zinc-400" aria-label="Page sections">
          <a
            href="#demo"
            className="hover:text-zinc-950 dark:hover:text-zinc-100 transition-colors"
          >
            Search Demo
          </a>
          <a
            href="#how-it-works"
            className="hover:text-zinc-950 dark:hover:text-zinc-100 transition-colors"
          >
            How It Works
          </a>
          <a
            href="#mcp"
            className="hover:text-zinc-950 dark:hover:text-zinc-100 transition-colors"
          >
            MCP Stdio
          </a>
          <a
            href="#limits"
            className="hover:text-zinc-950 dark:hover:text-zinc-100 transition-colors"
          >
            Limits
          </a>
        </nav>

        {/* Right side controls */}
        <div className="flex items-center gap-3">
          <div className="hidden sm:flex items-center gap-1.5 text-xs text-zinc-500 dark:text-zinc-400 font-mono px-2 py-1 rounded-md border border-zinc-200 dark:border-zinc-800/60 bg-zinc-50 dark:bg-zinc-900/40">
            <Sparkles className="w-3 h-3 text-emerald-500" />
            <span>RRF Fusion</span>
          </div>
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
