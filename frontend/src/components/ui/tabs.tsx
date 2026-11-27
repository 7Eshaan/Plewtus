"use client";

import type { ReactNode } from "react";
import { handleArrowNav } from "@/lib/arrow-nav";
import { cn } from "@/lib/cn";

export interface TabItem<T extends string> {
  value: T;
  label: ReactNode;
  icon?: ReactNode;
  /** Small count shown after the label. */
  badge?: ReactNode;
}

interface TabsProps<T extends string> {
  tabs: readonly TabItem<T>[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
  /** Tab ids are `${idBase}-tab-${value}`, for the panel's aria-labelledby. */
  idBase: string;
  /** id of the tabpanel the tabs control. */
  panelId: string;
  className?: string;
}

/**
 * M3 primary tabs: 48px tall, the active tab in the primary color with a
 * 3px indicator under it. Arrow keys move between tabs.
 */
export function Tabs<T extends string>({ tabs, value, onChange, ariaLabel, idBase, panelId, className }: TabsProps<T>) {
  return (
    <div role="tablist" aria-label={ariaLabel} onKeyDown={handleArrowNav} className={cn("flex items-center overflow-x-auto [scrollbar-width:none]", className)}>
      {tabs.map((tab) => {
        const selected = tab.value === value;
        return (
          <button
            key={tab.value}
            type="button"
            role="tab"
            id={`${idBase}-tab-${tab.value}`}
            aria-selected={selected}
            aria-controls={panelId}
            tabIndex={selected ? 0 : -1}
            onClick={() => onChange(tab.value)}
            className={cn(
              "state-layer focus-ring relative inline-flex h-12 shrink-0 items-center justify-center gap-2 px-3 sm:px-4 whitespace-nowrap select-none cursor-pointer",
              "text-title-sm transition-colors motion-effects [&_svg]:size-[18px] [&_svg]:shrink-0",
              selected ? "text-primary" : "text-on-surface-variant"
            )}
          >
            {tab.icon}
            <span>{tab.label}</span>
            {tab.badge}
            <span
              aria-hidden="true"
              className={cn(
                "absolute inset-x-3 bottom-0 h-[3px] rounded-t-full bg-primary transition-[opacity,transform] motion-spatial-fast",
                selected ? "opacity-100 scale-x-100" : "opacity-0 scale-x-50"
              )}
            />
          </button>
        );
      })}
    </div>
  );
}
