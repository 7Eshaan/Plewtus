import { ThemeToggle } from "./theme-toggle";
import { MobileNav } from "./mobile-nav";
import { REPO_URL, SECTION_LINKS } from "@/lib/site";

export function Header() {
  return (
    <header className="w-full border-b border-zinc-200 dark:border-zinc-800/80 bg-white/80 dark:bg-surface-dark/80 backdrop-blur-md sticky top-0 z-40 transition-colors">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        {/* Brand */}
        <a
          href="#demo"
          className="flex items-center gap-2.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 rounded-md group"
          aria-label="Plewtus Metasearch Home"
        >
          <div className="w-7 h-7 rounded-md bg-brand text-white flex items-center justify-center font-mono font-bold text-sm select-none shadow-xs shadow-brand/25 group-hover:scale-105 transition-transform">
            P
          </div>
          <span className="font-semibold text-sm tracking-tight text-zinc-950 dark:text-zinc-50 group-hover:text-brand dark:group-hover:text-brand-light transition-colors">
            Plewtus
          </span>
        </a>

        {/* Center navigation links */}
        <nav className="hidden md:flex items-center gap-6 text-xs font-mono text-zinc-600 dark:text-zinc-400" aria-label="Page sections">
          {SECTION_LINKS.map((link) => (
            <a key={link.href} href={link.href} className="hover:text-brand dark:hover:text-brand-light transition-colors">
              {link.label}
            </a>
          ))}
        </nav>

        {/* Right side controls */}
        <div className="flex items-center gap-2">
          <a
            href={REPO_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="hidden sm:inline-flex items-center h-8 px-2.5 rounded-md border border-zinc-200 dark:border-zinc-800 text-xs font-mono text-zinc-600 dark:text-zinc-400 hover:text-zinc-950 dark:hover:text-zinc-100 hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60"
          >
            GitHub
          </a>
          <ThemeToggle />
          <MobileNav />
        </div>
      </div>
    </header>
  );
}
