import { SearchX } from "lucide-react";

interface EmptyStateProps {
  query: string;
}

export function EmptyState({ query }: EmptyStateProps) {
  return (
    <div className="py-12 px-4 rounded-xl border border-dashed border-zinc-300 dark:border-zinc-800 flex flex-col items-center justify-center text-center gap-3 my-auto">
      <div className="w-10 h-10 rounded-full bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex items-center justify-center text-zinc-500">
        <SearchX className="w-5 h-5" />
      </div>
      <div className="flex flex-col gap-1 max-w-sm">
        <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">No matching results</h3>
        <p className="text-xs text-zinc-500 dark:text-zinc-400">
          No on-topic results were found for &ldquo;{query}&rdquo;. Try broader terms, another freshness window, or fewer domain filters.
        </p>
      </div>
    </div>
  );
}
