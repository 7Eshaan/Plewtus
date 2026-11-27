/** Placeholder result cards while a search runs (shaped like result-card.tsx). */
export function ResultSkeleton() {
  return (
    <div className="flex animate-pulse flex-col gap-2" aria-hidden="true">
      {[1, 2, 3, 4].map((index) => (
        <div key={index} className="flex flex-col gap-2 rounded-corner-md bg-surface-container p-3 sm:p-4">
          <div className="flex items-start gap-3">
            <div className="size-8 shrink-0 rounded-full bg-surface-container-highest" />
            <div className="flex flex-1 flex-col gap-1.5 pt-0.5">
              <div className="h-3 w-28 rounded-corner-xs bg-surface-container-highest" />
              <div className="h-4 w-3/5 rounded-corner-xs bg-surface-container-highest" />
            </div>
          </div>
          <div className="flex flex-col gap-1.5 sm:pl-11">
            <div className="h-3 w-full rounded-corner-xs bg-surface-container-high" />
            <div className="h-3 w-4/5 rounded-corner-xs bg-surface-container-high" />
          </div>
          <div className="flex items-center justify-between pt-1 sm:pl-11">
            <div className="flex gap-1">
              <div className="h-6 w-20 rounded-corner-xs bg-surface-container-highest" />
              <div className="h-6 w-16 rounded-corner-xs bg-surface-container-highest" />
            </div>
            <div className="h-5 w-16 rounded-corner-full bg-surface-container-highest" />
          </div>
        </div>
      ))}
    </div>
  );
}
