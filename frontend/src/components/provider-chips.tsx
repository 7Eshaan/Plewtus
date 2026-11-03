"use client";

export type ProviderName = "duckduckgo" | "bing" | "yahoo";

interface ProviderChipsProps {
  selected: ProviderName[];
  onChange: (providers: ProviderName[]) => void;
  disabled?: boolean;
}

const ALL_PROVIDERS: { id: ProviderName; label: string; short: string }[] = [
  { id: "duckduckgo", label: "DuckDuckGo", short: "DDG" },
  { id: "bing", label: "Bing", short: "Bing" },
  { id: "yahoo", label: "Yahoo", short: "Yahoo" },
];

export function ProviderChips({ selected, onChange, disabled }: ProviderChipsProps) {
  function toggleProvider(p: ProviderName) {
    if (disabled) return;
    if (selected.includes(p)) {
      // Prevent deselecting all providers (at least 1 must remain active)
      if (selected.length <= 1) return;
      onChange(selected.filter((x) => x !== p));
    } else {
      onChange([...selected, p]);
    }
  }

  return (
    <div className="flex items-center gap-1.5" role="group" aria-label="Search provider engines">
      <span className="text-[11px] font-mono text-zinc-400 dark:text-zinc-500 mr-0.5 hidden sm:inline">
        Engines:
      </span>
      {ALL_PROVIDERS.map((prov) => {
        const isSelected = selected.includes(prov.id);
        const isLastOne = isSelected && selected.length === 1;

        return (
          <button
            key={prov.id}
            type="button"
            onClick={() => toggleProvider(prov.id)}
            disabled={disabled || (isSelected && isLastOne)}
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono transition-all select-none cursor-pointer ${
              isSelected
                ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-950 border border-zinc-900 dark:border-white shadow-2xs font-medium"
                : "bg-zinc-100 dark:bg-zinc-900 text-zinc-500 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 hover:text-zinc-800 dark:hover:text-zinc-200"
            } ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
            title={
              isLastOne
                ? `${prov.label} (at least one engine required)`
                : `Toggle ${prov.label}`
            }
            aria-pressed={isSelected}
          >
            <span
              className={`w-1.5 h-1.5 rounded-full ${
                isSelected
                  ? "bg-emerald-400 dark:bg-emerald-600"
                  : "bg-zinc-300 dark:bg-zinc-600"
              }`}
            />
            <span>{prov.label}</span>
          </button>
        );
      })}
    </div>
  );
}
