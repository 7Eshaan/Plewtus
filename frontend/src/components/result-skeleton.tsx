"use client";

export function ResultSkeleton() {
  return (
    <div className="flex flex-col gap-1.5 animate-pulse" aria-busy="true" aria-label="Loading search results">
      {[1, 2, 3, 4].map((index) => (
        <div
          key={index}
          className="p-2 sm:p-2.5 rounded-lg border border-zinc-200/80 dark:border-zinc-800/80 bg-white/50 dark:bg-[#121215]/50 flex flex-col gap-1"
        >
          {/* Top row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <div className="w-4 h-4 rounded bg-zinc-200 dark:bg-zinc-800 flex-shrink-0" />
              <div className="w-40 sm:w-56 h-3 rounded bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-16 h-2.5 rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>

          {/* Snippet line */}
          <div className="flex flex-col gap-0.5 pl-5.5">
            <div className="w-full h-2.5 rounded bg-zinc-100 dark:bg-zinc-800/60" />
            <div className="w-3/4 h-2.5 rounded bg-zinc-100 dark:bg-zinc-800/60" />
          </div>

          {/* Bottom row */}
          <div className="flex items-center justify-between pt-1 border-t border-zinc-100 dark:border-zinc-800/40 pl-5.5">
            <div className="flex items-center gap-1">
              <div className="w-12 h-3 rounded bg-zinc-200 dark:bg-zinc-800" />
              <div className="w-10 h-3 rounded bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-12 h-3 rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>
        </div>
      ))}
    </div>
  );
}
