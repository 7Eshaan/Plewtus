"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "./button";
import { IconButton } from "./icon-button";
import { useSnackbar } from "./snackbar";

interface CopyButtonProps {
  /** Returns the text to copy; read at click time so it is always current. */
  getText: () => string;
  ariaLabel: string;
  /** "sm" is an icon button; "md" an outlined button labelled "Copy". */
  size?: "sm" | "md";
  /** "code" styles the small icon button for the always-dark code surfaces. */
  tone?: "default" | "code";
}

/** Copies text, confirming with a check mark and a snackbar. */
export function CopyButton({ getText, ariaLabel, size = "md", tone = "default" }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const showSnackbar = useSnackbar();

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(getText());
      setCopied(true);
      showSnackbar("Copied to clipboard");
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard access can be blocked (e.g. an insecure context).
      showSnackbar("Couldn't copy: clipboard access is blocked");
    }
  }

  const icon = copied ? <Check className={tone === "code" ? "text-code-primary" : "text-primary"} /> : <Copy />;

  if (size === "sm") {
    return (
      <IconButton label={ariaLabel} size="xs" variant={tone === "code" ? "code" : "standard"} onClick={handleCopy}>
        {icon}
      </IconButton>
    );
  }
  return (
    <Button variant="outlined" size="xs" icon={icon} onClick={handleCopy} aria-label={ariaLabel}>
      {copied ? "Copied" : "Copy"}
    </Button>
  );
}
