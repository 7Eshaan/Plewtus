"use client";

import { useTheme } from "next-themes";
import { useSyncExternalStore } from "react";
import { Sun, Moon } from "lucide-react";
import { IconButton } from "./ui/icon-button";

const emptySubscribe = () => () => {};

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const mounted = useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false
  );

  // The theme is only known on the client; keep the space until then.
  if (!mounted) {
    return <div className="size-10" aria-hidden="true" />;
  }

  const isDark = theme === "dark";
  return (
    <IconButton label={isDark ? "Switch to light theme" : "Switch to dark theme"} onClick={() => setTheme(isDark ? "light" : "dark")}>
      {isDark ? <Sun /> : <Moon />}
    </IconButton>
  );
}
