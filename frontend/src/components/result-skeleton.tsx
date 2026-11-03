"use client";

export function ResultSkeleton() {
  return (
    <div className="flex flex-col gap-3 animate-pulse" aria-busy="true" aria-label="Loading search results">
      {[1, 2, 3].map((index) => (
        <div
          key={index}
          className="p-4 sm:p-5 rounded-xl border border-zinc-200 dark:border-zinc-800/80 bg-white/50 dark:bg-[#121215]/50 flex flex-col gap-3"
        >
          {/* Top row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-6 h-5 rounded bg-zinc-200 dark:bg-zinc-800" />
              <div className="w-48 h-3.5 rounded bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-4 h-4 rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>

          {/* Title */}
          <div className="w-3/4 h-5 rounded bg-zinc-200 dark:bg-zinc-800" />

          {/* Snippet lines */}
          <div className="flex flex-col gap-1.5">
            <div className="w-full h-3.5 rounded bg-zinc-100 dark:bg-zinc-800/60" />
            <div className="w-5/6 h-3.5 rounded bg-zinc-100 dark:bg-zinc-800/60" />
          </div>

          {/* Bottom row */}
          <div className="flex items-center justify-between pt-2 border-t border-zinc-100 dark:border-zinc-800/40">
            <div className="flex items-center gap-1.5">
              <div className="w-16 h-5 rounded-full bg-zinc-200 dark:bg-zinc-800" />
              <div className="w-14 h-5 rounded-full bg-zinc-200 dark:bg-zinc-800" />
            </div>
            <div className="w-20 h-5 rounded-full bg-zinc-200 dark:bg-zinc-800" />
          </div>
        </div>
      ))}
    </div>
  );
}
