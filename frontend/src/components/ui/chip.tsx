import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/cn";

/**
 * M3 chips. Interactive chips are assist chips (an action) or, when given a
 * `selected` state, filter chips (a toggle, with a check once selected).
 * `chipClasses({ interactive: false, dense: true })` styles static tags.
 */
export function chipClasses({
  selected = false,
  tone = "default",
  hasIcon = false,
  hasTrailing = false,
  interactive = true,
  dense = false,
  className,
}: {
  selected?: boolean;
  /** "error" uses the error-container colors (the JSON tab's failed-status tag). */
  tone?: "default" | "error";
  hasIcon?: boolean;
  hasTrailing?: boolean;
  interactive?: boolean;
  /** 24px tall with smaller text, for tags inside cards. */
  dense?: boolean;
  className?: string;
} = {}): string {
  return cn(
    "inline-flex shrink-0 items-center whitespace-nowrap select-none border [&_svg]:shrink-0",
    "transition-[background-color,border-color,color] motion-effects",
    dense
      ? cn("h-6 gap-1 rounded-corner-xs text-label-md [&_svg]:size-3.5", hasIcon ? "pl-1.5" : "pl-2", hasTrailing ? "pr-1.5" : "pr-2")
      : cn("h-8 gap-2 rounded-corner-sm text-label-lg [&_svg]:size-[18px]", hasIcon ? "pl-2" : "pl-4", hasTrailing ? "pr-2" : "pr-4"),
    interactive && "state-layer focus-ring cursor-pointer disabled:cursor-not-allowed disabled:text-on-surface/38",
    selected
      ? "border-transparent bg-secondary-container text-on-secondary-container"
      : tone === "error"
        ? "border-transparent bg-error-container text-on-error-container"
        : "border-outline-variant text-on-surface-variant",
    className
  );
}

interface ChipProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** Makes this a filter chip (aria-pressed) showing a check when selected. */
  selected?: boolean;
  icon?: ReactNode;
  trailingIcon?: ReactNode;
}

export function Chip({ selected, icon, trailingIcon, className, children, type = "button", ...props }: ChipProps) {
  const leading = icon ?? (selected ? <Check aria-hidden="true" /> : null);
  return (
    <button
      type={type}
      aria-pressed={selected}
      className={chipClasses({ selected, hasIcon: Boolean(leading), hasTrailing: Boolean(trailingIcon), className })}
      {...props}
    >
      {leading}
      {children}
      {trailingIcon}
    </button>
  );
}
