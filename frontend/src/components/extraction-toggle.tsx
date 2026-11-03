"use client";

import { FileText, Sparkles, Ban } from "lucide-react";

export type ExtractionMode = "none" | "highlights" | "markdown";

interface ExtractionToggleProps {
  value: ExtractionMode;
  onChange: (mode: ExtractionMode) => void;
  disabled?: boolean;
}

const MODES: { id: ExtractionMode; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: "none", label: "None", icon: Ban },
  { id: "highlights", label: "Highlights", icon: Sparkles },
  { id: "markdown", label: "Markdown", icon: FileText },
];

export function ExtractionToggle({ value, onChange, disabled }: ExtractionToggleProps) {
  return (
    <div
      className="inline-flex items-center p-0.5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-100/70 dark:bg-zinc-900/70"
      role="group"
      aria-label="Content extraction mode"
    >
      <span className="text-[11px] font-mono text-zinc-400 dark:text-zinc-500 px-2 hidden md:inline">
        Extract:
      </span>
      {MODES.map((mode) => {
        const Icon = mode.icon;
        const isActive = value === mode.id;

        return (
          <button
            key={mode.id}
            type="button"
            onClick={() => onChange(mode.id)}
            disabled={disabled}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono transition-all select-none cursor-pointer ${
              isActive
                ? "bg-white text-zinc-950 dark:bg-zinc-800 dark:text-zinc-50 shadow-xs font-medium border border-zinc-200/80 dark:border-zinc-700/80"
                : "text-zinc-500 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200"
            } ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
            title={`Set extraction mode to ${mode.label}`}
            aria-pressed={isActive}
          >
            <Icon className="w-3 h-3" />
            <span>{mode.label}</span>
          </button>
        );
      })}
    </div>
  );
}
