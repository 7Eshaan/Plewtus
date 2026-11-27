"use client";

import { useId, type InputHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange" | "value" | "id"> {
  label: string;
  value: string;
  onChange: (value: string) => void;
}

/**
 * A compact M3 outlined text field for menus and forms: the label sits above
 * the field (rather than floating in its border) to save vertical space.
 */
export function TextField({ label, value, onChange, className, type = "text", ...props }: TextFieldProps) {
  const id = useId();
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <label htmlFor={id} className="text-body-sm text-on-surface-variant">
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          "h-10 w-full min-w-0 rounded-corner-xs border border-outline bg-transparent px-3",
          "text-body-lg text-on-surface placeholder:text-on-surface-variant/60",
          "outline-none transition-[border-color,box-shadow] motion-effects-fast hover:border-on-surface",
          "focus-visible:border-primary focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-primary"
        )}
        {...props}
      />
    </div>
  );
}
