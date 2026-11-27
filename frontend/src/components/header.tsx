"use client";

import { useEffect, useState } from "react";
import { ThemeToggle } from "./theme-toggle";
import { MobileNav } from "./mobile-nav";
import { REPO_URL, SECTION_LINKS } from "@/lib/site";
import { GitHubIcon } from "./ui/github-icon";
import { buttonClasses } from "./ui/button";
import { cn } from "@/lib/cn";

/**
 * Which page section is in view: the one crossing a band just above the
 * middle of the screen. Drives the active link in the header and the drawer.
 */
function useActiveSection(): string {
  const [active, setActive] = useState<string>(SECTION_LINKS[0].href);
  useEffect(() => {
    const sections = SECTION_LINKS.map((link) => document.querySelector<HTMLElement>(link.href)).filter(
      (el): el is HTMLElement => el !== null
    );
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) setActive(`#${entry.target.id}`);
        }
      },
      { rootMargin: "-40% 0px -55% 0px" }
    );
    sections.forEach((section) => observer.observe(section));
    return () => observer.disconnect();
  }, []);
  return active;
}

/** Whether the page has scrolled, so the app bar can take on its scrolled color. */
function useScrolled(): boolean {
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => {
    const update = () => setScrolled(window.scrollY > 4);
    const frame = requestAnimationFrame(update);
    window.addEventListener("scroll", update, { passive: true });
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("scroll", update);
    };
  }, []);
  return scrolled;
}

/** M3 small top app bar: surface at rest, surface-container once the page scrolls. */
export function Header() {
  const active = useActiveSection();
  const scrolled = useScrolled();

  return (
    <header
      className={cn(
        "sticky top-0 z-40 w-full backdrop-blur-md transition-colors motion-effects",
        scrolled ? "bg-surface-container/90" : "bg-surface/70"
      )}
    >
      <div className="mx-auto flex h-16 max-w-5xl items-center justify-between gap-2 px-2 sm:px-4">
        <div className="flex items-center gap-1">
          <MobileNav active={active} />
          {/* Brand */}
          <a
            href="#demo"
            className="focus-ring flex h-10 select-none items-center rounded-corner-sm px-3 text-title-lg font-semibold text-on-surface"
            aria-label="Plewtus home"
          >
            Plewtus
          </a>
        </div>

        {/* Section links */}
        <nav className="hidden items-center gap-1 md:flex" aria-label="Page sections">
          {SECTION_LINKS.map((link) => {
            const current = active === link.href;
            return (
              <a
                key={link.href}
                href={link.href}
                aria-current={current ? "location" : undefined}
                className={cn(
                  "state-layer focus-ring inline-flex h-10 items-center rounded-corner-full px-4 text-label-lg transition-colors motion-effects",
                  current ? "bg-secondary-container text-on-secondary-container" : "text-on-surface-variant"
                )}
              >
                {link.label}
              </a>
            );
          })}
        </nav>

        <div className="flex items-center gap-1">
          <span className="hidden sm:block">
            <a
              href={REPO_URL}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonClasses({ variant: "outlined", size: "sm", hasIcon: true })}
            >
              <GitHubIcon />
              GitHub
            </a>
          </span>
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
