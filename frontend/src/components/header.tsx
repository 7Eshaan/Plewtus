import { ThemeToggle } from "./theme-toggle";

const NAV_LINKS = [
  { href: "#demo", label: "Search Demo" },
  { href: "#how-it-works", label: "How It Works" },
  { href: "#mcp", label: "MCP Setup" },
  { href: "#quickstart", label: "API Quickstart" },
];

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
          {NAV_LINKS.map((link) => (
            <a key={link.href} href={link.href} className="hover:text-brand dark:hover:text-brand-light transition-colors">
              {link.label}
            </a>
          ))}
        </nav>

        {/* Right side controls */}
        <div className="flex items-center gap-3">
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
