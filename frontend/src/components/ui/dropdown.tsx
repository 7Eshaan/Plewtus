"use client";

import { useEffect, useLayoutEffect, useRef, type ReactNode } from "react";
import { Check, ChevronDown } from "lucide-react";
import { chipClasses } from "./chip";
import { cn } from "@/lib/cn";

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
 * An M3 chip that opens a menu below the toolbar. The chip shows as selected
 * (filter-chip colors) when its setting differs from the default. Closes on
 * Escape (returning focus to the chip) and on any click outside.
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
        className={chipClasses({
          selected: modified,
          hasIcon: Boolean(icon),
          hasTrailing: true,
          className: cn(!modified && "aria-expanded:bg-surface-container-highest"),
        })}
      >
        {icon && (
          <span aria-hidden="true" className={cn("flex", !modified && "text-primary")}>
            {icon}
          </span>
        )}
        {/* The icon stands in for the label below desktop width; screen readers always get it. */}
        <span className="sr-only lg:not-sr-only opacity-80">{label}</span>
        <span className="font-semibold">{value}</span>
        <ChevronDown className={cn("transition-transform motion-spatial-fast", open && "rotate-180")} aria-hidden="true" />
      </button>

      {open && (
        <div
          ref={panelRef}
          id={`${id}-panel`}
          className={cn(
            "dropdown-panel absolute z-30 top-full mt-1 left-3 right-3 p-1.5 text-left",
            "rounded-corner-lg bg-surface-container text-on-surface shadow-elevation-2",
            panelClassName ?? "sm:w-64"
          )}
        >
          {children}
        </div>
      )}
    </div>
  );
}

/** A selectable M3 menu item with a radio button or a checkbox. */
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
      className={cn(
        "state-layer focus-ring w-full flex items-start gap-3 px-3 py-2 rounded-corner-sm text-left text-on-surface",
        disabled ? "cursor-not-allowed opacity-[0.38]" : "cursor-pointer"
      )}
    >
      {kind === "radio" ? (
        // M3 radio button: a 20px ring with a dot when selected.
        <span
          aria-hidden="true"
          className={cn(
            "mt-px size-5 shrink-0 rounded-full border-2 flex items-center justify-center transition-colors motion-effects-fast",
            checked ? "border-primary" : "border-on-surface-variant"
          )}
        >
          <span
            className={cn(
              "size-2.5 rounded-full bg-primary transition-transform motion-spatial-fast",
              checked ? "scale-100" : "scale-0"
            )}
          />
        </span>
      ) : (
        // M3 checkbox: an 18px box with 2px corners, filled when checked.
        <span
          aria-hidden="true"
          className={cn(
            "mt-0.5 size-[18px] shrink-0 rounded-[2px] border-2 flex items-center justify-center transition-colors motion-effects-fast",
            checked ? "bg-primary border-primary text-on-primary" : "border-on-surface-variant"
          )}
        >
          {checked && <Check className="size-3.5 stroke-[3]" />}
        </span>
      )}
      <span className="flex flex-col min-w-0">
        <span className={cn("text-label-lg", checked && "font-semibold")}>{label}</span>
        {hint && <span className="text-body-sm text-on-surface-variant">{hint}</span>}
      </span>
    </button>
  );
}
