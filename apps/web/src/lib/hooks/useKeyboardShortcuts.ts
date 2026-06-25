import { useEffect, useRef } from "react";

type ShortcutHandler = (e: KeyboardEvent) => void;

/** Blocks shortcuts when the user is typing in a form element. */
function isTyping(e: KeyboardEvent): boolean {
  const tag = (e.target as HTMLElement)?.tagName?.toUpperCase();
  const editable = (e.target as HTMLElement)?.isContentEditable;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || editable;
}

/**
 * Register single-key shortcuts.
 *
 * @param map   Record of key → handler.  Keys are matched against `e.key`.
 * @param active Set to false to temporarily disable all shortcuts.
 */
export function useKeyboardShortcuts(
  map: Record<string, ShortcutHandler>,
  active = true,
): void {
  // Use a ref so callers don't need to memoize `map`
  const mapRef = useRef(map);
  useEffect(() => { mapRef.current = map; });

  useEffect(() => {
    if (!active) return;
    function handle(e: KeyboardEvent) {
      if (isTyping(e)) return;
      const fn = mapRef.current[e.key];
      if (fn) { e.preventDefault(); fn(e); }
    }
    window.addEventListener("keydown", handle);
    return () => window.removeEventListener("keydown", handle);
  }, [active]);
}

/**
 * Register two-key "go" sequences: press `g` then a letter within 1 second.
 *
 * @param destinations Record of second-key → callback (e.g. `{ d: goToDashboard }`)
 * @param active       Set to false to disable.
 */
export function useGoShortcuts(
  destinations: Record<string, () => void>,
  active = true,
): void {
  const destRef = useRef(destinations);
  useEffect(() => { destRef.current = destinations; });

  useEffect(() => {
    if (!active) return;
    let awaitingSecond = false;
    let timer: ReturnType<typeof setTimeout>;

    function handle(e: KeyboardEvent) {
      if (isTyping(e)) { awaitingSecond = false; return; }

      if (awaitingSecond) {
        clearTimeout(timer);
        awaitingSecond = false;
        const fn = destRef.current[e.key];
        if (fn) { e.preventDefault(); fn(); }
        return;
      }

      if (e.key === "g" && !e.ctrlKey && !e.metaKey && !e.altKey) {
        awaitingSecond = true;
        timer = setTimeout(() => { awaitingSecond = false; }, 1000);
      }
    }

    window.addEventListener("keydown", handle);
    return () => { window.removeEventListener("keydown", handle); clearTimeout(timer); };
  }, [active]);
}
