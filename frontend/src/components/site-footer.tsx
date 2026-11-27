import { REPO_URL, SECTION_LINKS } from "@/lib/site";
import { GitHubIcon } from "./ui/github-icon";

const LINK =
  "state-layer focus-ring inline-flex h-8 items-center gap-2 rounded-corner-full px-3 text-label-lg text-on-surface-variant transition-colors motion-effects hover:text-on-surface [&_svg]:size-4";

export function SiteFooter() {
  return (
    <footer className="w-full bg-surface-container-low py-8 text-body-md text-on-surface-variant">
      <div className="mx-auto flex max-w-5xl flex-col items-center justify-between gap-4 px-4 sm:flex-row">
        <div className="flex items-center gap-2">
          <span className="text-title-sm text-on-surface">Plewtus</span>
          <span aria-hidden="true">·</span>
          <span>Federated search for AI agents</span>
        </div>

        <nav className="flex flex-wrap items-center justify-center gap-1" aria-label="Footer">
          {SECTION_LINKS.map((link) => (
            <a key={link.href} href={link.href} className={LINK}>
              {link.label}
            </a>
          ))}
        </nav>

        <a href={REPO_URL} target="_blank" rel="noopener noreferrer" className={LINK}>
          <GitHubIcon />
          MIT · Source on GitHub
        </a>
      </div>
    </footer>
  );
}
