// Light/dark/auto theme mode, persisted in localStorage and reflected on
// <html data-theme>. The initial value is applied by an inline script in
// app.html (before paint) to avoid a flash of the wrong theme; this module is
// the runtime toggle used after the app mounts.

export type ThemePreference = "auto" | "light" | "dark";
export type EffectiveTheme = "light" | "dark";

const STORAGE_KEY = "iroha-theme";

export function prefersLight(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: light)").matches
  );
}

export function getThemePreference(): ThemePreference {
  if (typeof localStorage === "undefined") return "auto";
  try {
    const val = localStorage.getItem(STORAGE_KEY);
    if (val === "light" || val === "dark") return val;
  } catch (err) {
    void err;
  }
  return "auto";
}

export function getEffectiveTheme(): EffectiveTheme {
  if (typeof document === "undefined") return "dark";
  const attr = document.documentElement.dataset.theme;
  if (attr === "light" || attr === "dark") return attr;
  return prefersLight() ? "light" : "dark";
}

// Backwards compatibility alias
export const getTheme = getEffectiveTheme;
export type Theme = EffectiveTheme;

export function applyTheme(pref: ThemePreference): EffectiveTheme {
  const effective: EffectiveTheme =
    pref === "auto" ? (prefersLight() ? "light" : "dark") : pref;

  if (typeof document !== "undefined") {
    document.documentElement.dataset.theme = effective;
    document.documentElement.dataset.themePreference = pref;
    const favicon = document.querySelector<HTMLLinkElement>("#theme-favicon");
    if (favicon) {
      favicon.href =
        effective === "light" ? "/favicon-light.svg?v=3" : "/favicon.svg?v=3";
    }
  }

  if (typeof localStorage !== "undefined") {
    try {
      if (pref === "auto") {
        localStorage.removeItem(STORAGE_KEY);
      } else {
        localStorage.setItem(STORAGE_KEY, pref);
      }
    } catch (err) {
      void err;
    }
  }

  return effective;
}

export function setTheme(theme: EffectiveTheme): void {
  applyTheme(theme);
}

// Cycles: auto -> light -> dark -> auto
export function cycleTheme(): ThemePreference {
  const current = getThemePreference();
  const next: ThemePreference =
    current === "auto" ? "light" : current === "light" ? "dark" : "auto";
  applyTheme(next);
  return next;
}

export function toggleTheme(): EffectiveTheme {
  cycleTheme();
  return getEffectiveTheme();
}

export function initThemeWatcher(): () => void {
  if (
    typeof window === "undefined" ||
    typeof window.matchMedia !== "function"
  ) {
    return () => {};
  }
  const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
  const onChange = (e: MediaQueryListEvent) => {
    // If the preference is auto, immediately apply system changes
    if (getThemePreference() === "auto") {
      applyTheme("auto");
    }
  };
  mediaQuery.addEventListener("change", onChange);
  return () => mediaQuery.removeEventListener("change", onChange);
}
