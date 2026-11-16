import type { ReactNode } from "react";

interface SectionHeaderProps {
  eyebrow: string;
  title: string;
  children?: ReactNode;
}

/** The heading block shared by the documentation sections below the hero. */
export function SectionHeader({ eyebrow, title, children }: SectionHeaderProps) {
  return (
    <header className="flex flex-col gap-4 max-w-2xl">
      <p className="text-[15px] font-medium text-brand dark:text-brand-light">{eyebrow}</p>
      <h2 className="text-4xl sm:text-5xl font-semibold tracking-[-0.03em] leading-[1.1] text-zinc-950 dark:text-zinc-50 text-balance">
        {title}
      </h2>
      {children && <p className="text-lg text-zinc-600 dark:text-zinc-400 leading-relaxed text-pretty">{children}</p>}
    </header>
  );
}
