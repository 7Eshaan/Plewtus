import type { HTMLAttributes } from "react";
import { cn } from "@/lib/cn";

/** M3 cards: filled (default), elevated, or outlined. */
export type CardVariant = "filled" | "elevated" | "outlined";

const VARIANTS: Record<CardVariant, string> = {
  filled: "bg-surface-container-highest",
  elevated: "bg-surface-container-low shadow-elevation-1",
  outlined: "bg-surface border border-outline-variant",
};

export function cardClasses({
  variant = "filled",
  interactive = false,
  className,
}: {
  variant?: CardVariant;
  /** Adds the hover/press state layer for clickable cards. */
  interactive?: boolean;
  className?: string;
} = {}): string {
  return cn(
    "rounded-corner-md text-on-surface",
    VARIANTS[variant],
    interactive && "state-layer focus-ring cursor-pointer",
    interactive && variant === "elevated" && "transition-shadow motion-effects hover:shadow-elevation-2",
    className
  );
}

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant;
}

export function Card({ variant, className, ...props }: CardProps) {
  return <div className={cardClasses({ variant, className })} {...props} />;
}
