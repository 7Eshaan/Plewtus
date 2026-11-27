"use client";

import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";

/** How long a snackbar stays up (M3 suggests 4-10 s). */
const DURATION_MS = 4000;

type ShowSnackbar = (message: string) => void;

const SnackbarContext = createContext<ShowSnackbar>(() => {});

/**
 * M3 snackbar host: one short message at a time at the bottom of the screen,
 * announced politely to screen readers. A new message replaces the current one.
 */
export function SnackbarProvider({ children }: { children: ReactNode }) {
  const [snackbar, setSnackbar] = useState<{ id: number; message: string } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const show = useCallback<ShowSnackbar>((message) => {
    if (timer.current) clearTimeout(timer.current);
    setSnackbar((prev) => ({ id: (prev?.id ?? 0) + 1, message }));
    timer.current = setTimeout(() => setSnackbar(null), DURATION_MS);
  }, []);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    []
  );

  return (
    <SnackbarContext.Provider value={show}>
      {children}
      <div
        role="status"
        aria-live="polite"
        className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex justify-center px-4"
      >
        {snackbar && (
          <div
            key={snackbar.id}
            className="m3-snackbar pointer-events-auto flex min-h-12 max-w-xl items-center rounded-corner-xs bg-inverse-surface px-4 py-3.5 text-body-md text-inverse-on-surface shadow-elevation-3"
          >
            {snackbar.message}
          </div>
        )}
      </div>
    </SnackbarContext.Provider>
  );
}

/** Returns a function that shows a snackbar message. */
export function useSnackbar(): ShowSnackbar {
  return useContext(SnackbarContext);
}
