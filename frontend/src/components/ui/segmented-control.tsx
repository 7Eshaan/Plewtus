"use client";

import type { ReactNode } from "react";
import { handleArrowNav } from "@/lib/arrow-nav";
import { cn } from "@/lib/cn";

export interface SegmentOption<T extends string> {
  value: T;
  label: ReactNode;
}

interface SegmentedControlProps<T extends string> {
  options: readonly SegmentOption<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  size?: "sm" | "md";
  /** Stretch the group to its container, sharing the width equally. */
  fullWidth?: boolean;
}

const SIZES = {
  sm: { button: "h-7 px-3 text-label-md", outer: 14 },
  md: { button: "h-8 px-3.5 text-label-lg", outer: 16 },
} as const;

/** Inner corners of the connected buttons (M3 corner extra-small). */
const INNER = 4;

/**
 * A single-choice M3 Expressive connected button group (radio group with
 * arrow-key navigation). The group's outer corners are fully round and the
 * corners between buttons small; the selected button morphs fully round.
 */
export function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  size = "md",
  fullWidth = false,
}: SegmentedControlProps<T>) {
  const { button, outer } = SIZES[size];
  const last = options.length - 1;

  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      onKeyDown={handleArrowNav}
      className={cn("items-center gap-0.5", fullWidth ? "flex w-full" : "inline-flex shrink-0")}
    >
      {options.map((option, i) => {
        const selected = option.value === value;
        // Per-corner radii (top-left, top-right, bottom-right, bottom-left) as
        // an inline style, so the morph between them animates.
        const left = selected || i === 0 ? outer : INNER;
        const right = selected || i === last ? outer : INNER;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={selected}
            tabIndex={selected ? 0 : -1}
            onClick={() => onChange(option.value)}
            style={{ borderRadius: `${left}px ${right}px ${right}px ${left}px` }}
            className={cn(
              "state-layer focus-ring inline-flex items-center justify-center gap-1.5 whitespace-nowrap select-none cursor-pointer",
              "transition-[border-radius,background-color,color] motion-spatial-fast",
              button,
              fullWidth && "flex-1",
              selected
                ? "bg-secondary-container text-on-secondary-container"
                : "bg-surface-container-highest text-on-surface-variant"
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
