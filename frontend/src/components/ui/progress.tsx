import { cn } from "@/lib/cn";

interface LinearProgressProps {
  /** Progress from 0 to 1; leave undefined for an indeterminate bar. */
  value?: number;
  label?: string;
  className?: string;
}

/**
 * M3 linear progress indicator (4px). Determinate bars follow M3 Expressive:
 * a gap between the indicator and the track, and a stop dot at the end.
 */
export function LinearProgress({ value, label = "Loading", className }: LinearProgressProps) {
  const determinate = typeof value === "number";
  const percent = determinate ? Math.round(Math.min(1, Math.max(0, value)) * 100) : undefined;

  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={determinate ? 0 : undefined}
      aria-valuemax={determinate ? 100 : undefined}
      aria-valuenow={percent}
      className={cn("relative h-1 w-full overflow-hidden rounded-full", className)}
    >
      {determinate ? (
        <>
          <div
            className="absolute inset-y-0 left-0 rounded-full bg-primary transition-[width] motion-spatial"
            style={{ width: `${percent}%` }}
          />
          {percent! < 100 && (
            <>
              <div
                className="absolute inset-y-0 right-0 rounded-full bg-primary-container/25 transition-[left] motion-spatial"
                style={{ left: `calc(${percent}% + 4px)` }}
              />
              <div className="absolute inset-y-0 right-0 aspect-square rounded-full bg-primary" />
            </>
          )}
        </>
      ) : (
        <>
          {/* A tint of the brand purple, so the track stays distinct from the indicator. */}
          <div className="absolute inset-0 rounded-full bg-primary-container/25" />
          <div className="m3-linear-indeterminate absolute inset-y-0 left-0 w-2/5 rounded-full bg-primary" />
        </>
      )}
    </div>
  );
}

interface CircularProgressProps {
  /** Diameter in px (default 24). */
  size?: number;
  label?: string;
  /** "primary" (default), or "current" to inherit the text color (inside buttons). */
  color?: "primary" | "current";
  /** Hidden from assistive technology when its container already says what is loading. */
  decorative?: boolean;
  className?: string;
}

/** M3 indeterminate circular progress indicator. */
export function CircularProgress({
  size = 24,
  label = "Loading",
  color = "primary",
  decorative = false,
  className,
}: CircularProgressProps) {
  return (
    <svg
      role={decorative ? undefined : "progressbar"}
      aria-label={decorative ? undefined : label}
      aria-hidden={decorative || undefined}
      viewBox="0 0 48 48"
      width={size}
      height={size}
      className={cn("m3-circular shrink-0", color === "primary" && "text-primary", className)}
    >
      <circle cx="24" cy="24" r="20" fill="none" stroke="currentColor" strokeWidth="5" strokeLinecap="round" />
    </svg>
  );
}
