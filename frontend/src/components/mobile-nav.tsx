"use client";

import { useRef } from "react";
import { Menu, X } from "lucide-react";
import { SECTION_LINKS } from "@/lib/site";
import { IconButton } from "./ui/icon-button";
import { cn } from "@/lib/cn";

/**
 * M3 modal navigation drawer for screens too narrow for the header links.
 * A native <dialog> opened with showModal(): it traps focus, closes on
 * Escape, makes the page behind it inert, and returns focus to the button.
 */
export function MobileNav({ active }: { active: string }) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const close = () => dialogRef.current?.close();

  return (
    <>
      <IconButton label="Open menu" aria-haspopup="dialog" className="md:hidden" onClick={() => dialogRef.current?.showModal()}>
        <Menu />
      </IconButton>
      <dialog
        ref={dialogRef}
        aria-label="Menu"
        // A click on the dialog element itself (not its content) is on the scrim.
        onClick={(e) => {
          if (e.target === dialogRef.current) close();
        }}
        className="m3-drawer fixed inset-y-0 left-0 m-0 h-dvh max-h-none w-[min(360px,85vw)] max-w-none rounded-r-corner-lg bg-surface-container-low p-0 text-on-surface shadow-elevation-1 backdrop:bg-scrim/32"
      >
        <nav className="flex h-full flex-col gap-1 p-3" aria-label="Page sections">
          <div className="mb-2 flex h-14 items-center justify-between pl-4">
            <span className="text-title-sm text-on-surface-variant">Plewtus</span>
            <IconButton label="Close menu" onClick={close}>
              <X />
            </IconButton>
          </div>
          {SECTION_LINKS.map((link) => {
            const current = active === link.href;
            return (
              <a
                key={link.href}
                href={link.href}
                onClick={close}
                aria-current={current ? "location" : undefined}
                className={cn(
                  "state-layer focus-ring flex h-14 items-center rounded-corner-full px-4 text-label-lg",
                  current ? "bg-secondary-container text-on-secondary-container" : "text-on-surface-variant"
                )}
              >
                {link.label}
              </a>
            );
          })}
        </nav>
      </dialog>
    </>
  );
}
