import type { ReactNode } from "react";

interface SectionHeaderProps {
  eyebrow: string;
  title: string;
  children?: ReactNode;
}

/** The heading block shared by the documentation sections below the hero. */
export function SectionHeader({ eyebrow, title, children }: SectionHeaderProps) {
  return (
    <header className="flex max-w-2xl flex-col gap-4">
      <p className="text-title-md text-primary">{eyebrow}</p>
      {/* Display type in its M3 Expressive emphasized weight. */}
      <h2 className="text-balance text-display-sm font-medium text-on-surface sm:text-display-md">{title}</h2>
      {children && <p className="text-pretty text-body-lg text-on-surface-variant sm:text-title-lg">{children}</p>}
    </header>
  );
}
