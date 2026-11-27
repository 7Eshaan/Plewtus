import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/cn";

/** "code" is a standard icon button for the always-dark code surfaces. */
export type IconButtonVariant = "standard" | "filled" | "tonal" | "outlined" | "code";
export type IconButtonSize = "xs" | "sm";

/**
 * Colors per variant: `plain` for ordinary buttons; `off` / `on` for toggle
 * buttons (those given a `selected` state).
 */
const VARIANTS: Record<IconButtonVariant, { plain: string; off: string; on: string }> = {
  standard: { plain: "text-on-surface-variant", off: "text-on-surface-variant", on: "text-primary" },
  filled: {
    plain: "bg-primary-container text-on-primary-container",
    off: "bg-surface-container-highest text-primary",
    on: "bg-primary-container text-on-primary-container",
  },
  tonal: {
    plain: "bg-secondary-container text-on-secondary-container",
    off: "bg-surface-container-highest text-on-surface-variant",
    on: "bg-secondary-container text-on-secondary-container",
  },
  outlined: {
    plain: "border border-outline-variant text-on-surface-variant",
    off: "border border-outline-variant text-on-surface-variant",
    on: "bg-inverse-surface text-inverse-on-surface",
  },
  code: { plain: "text-code-on-surface-variant", off: "text-code-on-surface-variant", on: "text-code-on-surface" },
};

const SIZES: Record<IconButtonSize, string> = {
  xs: "size-8 rounded-[16px] active:rounded-corner-sm [&_svg]:size-[18px]",
  sm: "size-10 rounded-[20px] active:rounded-corner-md [&_svg]:size-5",
};

export function iconButtonClasses({
  variant = "standard",
  size = "sm",
  selected,
  className,
}: {
  variant?: IconButtonVariant;
  size?: IconButtonSize;
  /** Toggle buttons pass their state; leave undefined for ordinary buttons. */
  selected?: boolean;
  className?: string;
} = {}): string {
  const colors = VARIANTS[variant];
  return cn(
    "state-layer focus-ring inline-flex shrink-0 items-center justify-center select-none cursor-pointer",
    "transition-[border-radius,background-color,color] motion-spatial-fast",
    "disabled:cursor-not-allowed disabled:text-on-surface/38 [&_svg]:shrink-0",
    selected === undefined ? colors.plain : selected ? colors.on : colors.off,
    (variant === "filled" || variant === "tonal") && "disabled:bg-on-surface/10",
    SIZES[size],
    className
  );
}

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "aria-label"> {
  /** Accessible name; also shown as the tooltip unless `title` is given. */
  label: string;
  variant?: IconButtonVariant;
  size?: IconButtonSize;
  /** Set for toggle buttons; adds aria-pressed. */
  selected?: boolean;
  children: ReactNode;
}

export function IconButton({
  label,
  variant,
  size,
  selected,
  title,
  className,
  children,
  type = "button",
  ...props
}: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      aria-pressed={selected}
      title={title ?? label}
      className={iconButtonClasses({ variant, size, selected, className })}
      {...props}
    >
      {children}
    </button>
  );
}
