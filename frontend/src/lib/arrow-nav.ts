import type { KeyboardEvent } from "react";

/**
 * Keyboard handler for radio groups and tab lists: arrow keys (and Home/End)
 * move focus between the group's options and select the one focused, which is
 * what assistive technology expects from role="radiogroup" / role="tablist".
 */
export function handleArrowNav(event: KeyboardEvent<HTMLElement>): void {
  const keys = ["ArrowRight", "ArrowDown", "ArrowLeft", "ArrowUp", "Home", "End"];
  if (!keys.includes(event.key)) return;

  const options = Array.from(
    event.currentTarget.querySelectorAll<HTMLElement>('[role="radio"], [role="tab"]')
  ).filter((el) => !el.hasAttribute("disabled"));
  const current = options.indexOf(document.activeElement as HTMLElement);
  if (current === -1 || options.length < 2) return;

  let next = current;
  if (event.key === "Home") next = 0;
  else if (event.key === "End") next = options.length - 1;
  else if (event.key === "ArrowRight" || event.key === "ArrowDown") next = (current + 1) % options.length;
  else next = (current - 1 + options.length) % options.length;

  event.preventDefault();
  options[next].focus();
  options[next].click();
}
