"use client";

import { ThemeToggle } from "./theme-toggle";

export function Header() {
  return (
    <header className="w-full border-b border-zinc-200 dark:border-zinc-800/80 bg-white/80 dark:bg-[#09090b]/80 backdrop-blur-md sticky top-0 z-40 transition-colors">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        {/* Brand */}
        <a
          href="#demo"
          className="flex items-center gap-2.5 focus:outline-none focus:ring-1 focus:ring-[#8700ff]/50 rounded-md group"
          aria-label="Plewtus Metasearch Home"
        >
          <div className="w-7 h-7 rounded-md bg-[#8700ff] text-white flex items-center justify-center font-mono font-bold text-sm select-none shadow-xs shadow-[#8700ff]/25 group-hover:scale-105 transition-transform">
            P
          </div>
          <span className="font-semibold text-sm tracking-tight text-zinc-950 dark:text-zinc-50 group-hover:text-[#8700ff] dark:group-hover:text-[#a84dff] transition-colors">
            Plewtus
          </span>
        </a>

        {/* Center navigation links */}
        <nav className="hidden md:flex items-center gap-6 text-xs font-mono text-zinc-600 dark:text-zinc-400" aria-label="Page sections">
          <a
            href="#demo"
            className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors"
          >
            Search Demo
          </a>
          <a
            href="#how-it-works"
            className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors"
          >
            How It Works
          </a>
          <a
            href="#mcp"
            className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors"
          >
            MCP Stdio
          </a>
          <a
            href="#quickstart"
            className="hover:text-[#8700ff] dark:hover:text-[#a84dff] transition-colors"
          >
            API Quickstart
          </a>
        </nav>

        {/* Right side controls */}
        <div className="flex items-center gap-3">
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
