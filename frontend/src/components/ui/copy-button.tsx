"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";

interface CopyButtonProps {
  /** Returns the text to copy; read at click time so it is always current. */
  getText: () => string;
  ariaLabel: string;
  size?: "sm" | "md";
}

export function CopyButton({ getText, ariaLabel, size = "md" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(getText());
      setCopied(true);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard access can be blocked (e.g. insecure context); nothing to do.
    }
  }

  const sizing = size === "sm" ? "gap-1 px-2 py-0.5 text-[11px]" : "gap-1.5 px-2.5 py-1 text-xs";
  const icon = size === "sm" ? "w-3 h-3" : "w-3.5 h-3.5";

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={ariaLabel}
      className={`inline-flex items-center ${sizing} rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 text-zinc-700 dark:text-zinc-200 hover:border-zinc-400 dark:hover:border-zinc-500 font-mono transition-colors cursor-pointer shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/60`}
    >
      {copied ? (
        <>
          <Check className={`${icon} text-emerald-500`} />
          <span className="text-emerald-600 dark:text-emerald-400 font-semibold" aria-live="polite">
            Copied
          </span>
        </>
      ) : (
        <>
          <Copy className={icon} />
          <span>Copy</span>
        </>
      )}
    </button>
  );
}
