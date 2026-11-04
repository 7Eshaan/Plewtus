"use client";

export function ResultSkeleton() {
  return (
    <div className="flex flex-col gap-2 animate-pulse" aria-busy="true" aria-label="Loading search results">
      {[1, 2, 3, 4].map((index) => (
        <div
          key={index}
          className="p-3 sm:p-3.5 rounded-xl border border-zinc-200/80 dark:border-zinc-800/80 bg-white/50 dark:bg-[#121215]/50 flex flex-col gap-1.5"
        >
          {/* Top row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-5 h-5 rounded-md bg-zinc-200 dark:bg-zinc-800 flex-shrink-0" />
              <div className="w-48 sm:w-64 h-3.5 rounded bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-20 h-3 rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>

          {/* Snippet line */}
          <div className="flex flex-col gap-1 pl-7">
            <div className="w-full h-3 rounded bg-zinc-100 dark:bg-zinc-800/60" />
            <div className="w-3/4 h-3 rounded bg-zinc-100 dark:bg-zinc-800/60" />
          </div>

          {/* Bottom row */}
          <div className="flex items-center justify-between pt-1.5 border-t border-zinc-100 dark:border-zinc-800/40 pl-7">
            <div className="flex items-center gap-1.5">
              <div className="w-14 h-3.5 rounded bg-zinc-200 dark:bg-zinc-800" />
              <div className="w-12 h-3.5 rounded bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-16 h-3.5 rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>
        </div>
      ))}
    </div>
  );
}
