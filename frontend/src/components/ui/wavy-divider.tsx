import { cn } from "@/lib/cn";

interface WavyDividerProps {
  /** Number of full waves (default 7). */
  waves?: number;
  /** Length of one wave in px (default 48). */
  wavelength?: number;
  /** Height of a crest above the center line in px (default 5). */
  amplitude?: number;
  /** Line thickness in px (default 4). */
  strokeWidth?: number;
  className?: string;
}

/**
 * The M3 Expressive wavy line, used to divide page sections: a centered sine
 * wave with round ends, drawn in the primary color.
 */
export function WavyDivider({ waves = 7, wavelength = 48, amplitude = 5, strokeWidth = 4, className }: WavyDividerProps) {
  const pad = strokeWidth / 2;
  const width = waves * wavelength + strokeWidth;
  const height = amplitude * 2 + strokeWidth;
  const half = wavelength / 2;

  // A quadratic curve whose control point sits at twice the amplitude peaks at
  // the amplitude; each smooth "t" continuation mirrors the previous control
  // point, alternating crests and troughs.
  let d = `M ${pad} ${height / 2} q ${half / 2} ${-amplitude * 2} ${half} 0`;
  for (let i = 1; i < waves * 2; i++) d += ` t ${half} 0`;

  return (
    <div role="separator" className={cn("flex w-full justify-center text-primary", className)}>
      <svg
        aria-hidden="true"
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        className="max-w-full overflow-visible"
      >
        <path d={d} fill="none" stroke="currentColor" strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  );
}
