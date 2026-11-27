import { SearchX } from "lucide-react";

interface EmptyStateProps {
  query: string;
}

export function EmptyState({ query }: EmptyStateProps) {
  return (
    <div className="my-auto flex flex-col items-center justify-center gap-3 rounded-corner-lg border border-dashed border-outline-variant px-4 py-12 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-surface-container-highest text-on-surface-variant">
        <SearchX className="size-6" aria-hidden="true" />
      </div>
      <div className="flex max-w-sm flex-col gap-1">
        <h3 className="text-title-md text-on-surface">No matching results</h3>
        <p className="text-body-md text-on-surface-variant">
          No on-topic results were found for &ldquo;{query}&rdquo;. Try broader terms, another freshness window, or fewer domain filters.
        </p>
      </div>
    </div>
  );
}
