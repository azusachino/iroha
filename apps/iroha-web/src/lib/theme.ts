// Light/dark theme, persisted in localStorage and reflected on
// <html data-theme>. The initial value is applied by an inline script in
// app.html (before paint) to avoid a flash of the wrong theme; this module is
// the runtime toggle used after the app mounts.

export type Theme = "light" | "dark";

const STORAGE_KEY = "iroha-theme";

function prefersLight(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: light)").matches
  );
}

export function getTheme(): Theme {
  if (typeof document === "undefined") return "dark";
  const attr = document.documentElement.dataset.theme;
  if (attr === "light" || attr === "dark") return attr;
  return prefersLight() ? "light" : "dark";
}

export function setTheme(theme: Theme): void {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    // Ignore storage failures (private mode etc.); the in-DOM attribute still applies.
  }
}

export function toggleTheme(): Theme {
  const next: Theme = getTheme() === "dark" ? "light" : "dark";
  setTheme(next);
  return next;
}
