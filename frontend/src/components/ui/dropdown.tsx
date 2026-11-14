"use client";

import { useEffect, useLayoutEffect, useRef, type ReactNode } from "react";
import { Check, ChevronDown } from "lucide-react";

interface DropdownProps {
  /** Stable id; the panel gets `${id}-panel`. */
  id: string;
  /** Muted label before the value, e.g. "Engines". */
  label: string;
  /** Current selection shown on the button. */
  value: ReactNode;
  icon?: ReactNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Highlights the button when the selection differs from the default. */
  modified?: boolean;
  /** Panel width class (defaults to a comfortable menu width). */
  panelClassName?: string;
  children: ReactNode;
}

/**
 * A toolbar button that opens a small panel below the toolbar. Closes on
 * Escape (returning focus to the button) and on any click outside.
 *
 * The panel is positioned against the toolbar (the nearest positioned
 * ancestor, which must be `relative`). On narrow screens it spans the toolbar;
 * on wider ones it lines up with its button but is shifted left as needed to
 * stay inside the toolbar, however the buttons have wrapped.
 */
export function Dropdown({
  id,
  label,
  value,
  icon,
  open,
  onOpenChange,
  modified,
  panelClassName,
  children,
}: DropdownProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  // Runs before paint, so the panel never flashes in the wrong place.
  useLayoutEffect(() => {
    const panel = panelRef.current;
    const root = rootRef.current;
    const toolbar = panel?.offsetParent as HTMLElement | null;
    if (!open || !panel || !root || !toolbar) return;
    if (window.matchMedia("(min-width: 640px)").matches) {
      const gutter = 12;
      const maxLeft = toolbar.clientWidth - panel.offsetWidth - gutter;
      panel.style.left = `${Math.max(gutter, Math.min(root.offsetLeft, maxLeft))}px`;
      panel.style.right = "auto";
    } else {
      panel.style.left = "";
      panel.style.right = "";
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: PointerEvent) {
      if (!rootRef.current?.contains(e.target as Node)) onOpenChange(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onOpenChange(false);
        buttonRef.current?.focus();
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, onOpenChange]);

  return (
    <div ref={rootRef}>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        aria-controls={`${id}-panel`}
        onClick={() => onOpenChange(!open)}
        className={`inline-flex items-center gap-1.5 h-8 px-2.5 rounded-lg border text-xs font-mono whitespace-nowrap transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 ${
          open
            ? "border-zinc-300 dark:border-zinc-700 bg-zinc-100 dark:bg-zinc-800/80 text-zinc-950 dark:text-zinc-50"
            : modified
              ? "border-brand/30 bg-brand/5 text-zinc-900 dark:text-zinc-100 hover:border-brand/50"
              : "border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 text-zinc-700 dark:text-zinc-300 hover:border-zinc-300 dark:hover:border-zinc-700 hover:text-zinc-950 dark:hover:text-zinc-100"
        }`}
      >
        {icon && <span className="text-zinc-400 dark:text-zinc-500 [&>svg]:w-3.5 [&>svg]:h-3.5">{icon}</span>}
        {/* The icon stands in for the label below desktop width; screen readers always get it. */}
        <span className="sr-only lg:not-sr-only text-zinc-500 dark:text-zinc-500">{label}</span>
        <span className="font-semibold text-zinc-900 dark:text-zinc-100">{value}</span>
        <ChevronDown className={`w-3.5 h-3.5 text-zinc-400 transition-transform ${open ? "rotate-180" : ""}`} aria-hidden="true" />
      </button>

      {open && (
        <div
          ref={panelRef}
          id={`${id}-panel`}
          className={`dropdown-panel absolute z-30 top-full mt-1 left-3 right-3 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-xl shadow-black/10 dark:shadow-black/40 p-1.5 text-left ${
            panelClassName ?? "sm:w-64"
          }`}
        >
          {children}
        </div>
      )}
    </div>
  );
}

/** A selectable row in a dropdown panel: a radio choice or a checkbox. */
export function DropdownOption({
  kind,
  checked,
  onSelect,
  label,
  hint,
  disabled,
  disabledReason,
}: {
  kind: "radio" | "checkbox";
  checked: boolean;
  onSelect: () => void;
  label: ReactNode;
  hint?: string;
  disabled?: boolean;
  disabledReason?: string;
}) {
  return (
    <button
      type="button"
      role={kind}
      aria-checked={checked}
      tabIndex={kind === "radio" && !checked ? -1 : undefined}
      onClick={onSelect}
      aria-disabled={disabled || undefined}
      title={disabled ? disabledReason : undefined}
      className={`w-full flex items-start gap-2.5 px-2 py-1.5 rounded-lg text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60 ${
        disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800/80"
      }`}
    >
      <span
        aria-hidden="true"
        className={`mt-0.5 w-4 h-4 shrink-0 flex items-center justify-center border transition-colors ${
          kind === "radio" ? "rounded-full" : "rounded"
        } ${checked ? "bg-brand border-brand text-white" : "border-zinc-300 dark:border-zinc-600"}`}
      >
        {checked && (kind === "radio" ? <span className="w-1.5 h-1.5 rounded-full bg-white" /> : <Check className="w-3 h-3 stroke-[3]" />)}
      </span>
      <span className="flex flex-col min-w-0">
        <span className={`text-xs font-mono ${checked ? "text-zinc-950 dark:text-zinc-50 font-semibold" : "text-zinc-700 dark:text-zinc-300"}`}>
          {label}
        </span>
        {hint && <span className="text-[11px] leading-snug text-zinc-500 dark:text-zinc-400">{hint}</span>}
      </span>
    </button>
  );
}
