import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/cn";

/**
 * M3 common buttons. "filled" carries the brand purple (primary-container,
 * #8700ff in both themes); use it once per area for the main action.
 */
export type ButtonVariant = "filled" | "tonal" | "outlined" | "text" | "elevated";

/** M3 Expressive sizes: xs = 32px, sm = 40px tall. */
export type ButtonSize = "xs" | "sm";

const VARIANTS: Record<ButtonVariant, string> = {
  filled: "bg-primary-container text-on-primary-container disabled:bg-on-surface/10",
  tonal: "bg-secondary-container text-on-secondary-container disabled:bg-on-surface/10",
  outlined: "border border-outline-variant text-on-surface-variant disabled:border-on-surface/12",
  text: "text-primary",
  elevated:
    "bg-surface-container-low text-primary shadow-elevation-1 hover:shadow-elevation-2 disabled:bg-on-surface/10 disabled:shadow-none",
};

// Corners are half the height at rest and morph to a squarer shape while
// pressed (M3 Expressive). An explicit radius, not rounded-full, so the
// change animates.
const SIZES: Record<ButtonSize, string> = {
  xs: "h-8 gap-1.5 pr-3 rounded-[16px] active:rounded-corner-sm [&_svg]:size-[18px]",
  sm: "h-10 gap-2 pr-4 rounded-[20px] active:rounded-corner-md [&_svg]:size-5",
};

/** Left padding shrinks when a leading icon is shown. */
const LEFT_PADDING: Record<ButtonSize, { text: string; icon: string }> = {
  xs: { text: "pl-3", icon: "pl-2.5" },
  sm: { text: "pl-4", icon: "pl-3.5" },
};

export function buttonClasses({
  variant = "filled",
  size = "sm",
  hasIcon = false,
  className,
}: {
  variant?: ButtonVariant;
  size?: ButtonSize;
  hasIcon?: boolean;
  className?: string;
} = {}): string {
  return cn(
    "state-layer focus-ring inline-flex shrink-0 items-center justify-center whitespace-nowrap select-none cursor-pointer",
    "text-label-lg transition-[border-radius,background-color,color,box-shadow] motion-spatial-fast",
    "disabled:cursor-not-allowed disabled:text-on-surface/38 [&_svg]:shrink-0",
    VARIANTS[variant],
    SIZES[size],
    hasIcon ? LEFT_PADDING[size].icon : LEFT_PADDING[size].text,
    className
  );
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Leading icon (a lucide icon); sized by the button. */
  icon?: ReactNode;
  trailingIcon?: ReactNode;
}

export function Button({
  variant,
  size,
  icon,
  trailingIcon,
  className,
  children,
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button type={type} className={buttonClasses({ variant, size, hasIcon: Boolean(icon), className })} {...props}>
      {icon}
      {children}
      {trailingIcon}
    </button>
  );
}
