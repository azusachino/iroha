// Light/dark/auto theme mode, persisted in localStorage and reflected on
// <meta name="color-scheme">, <meta name="theme-color">, and <html data-theme>.
// The initial value is applied by an inline script in app.html (before paint)
// to avoid a flash of the wrong theme; this module provides the reactive state
// and runtime toggle used after the app mounts.

export type ThemePreference = "auto" | "light" | "dark";
export type EffectiveTheme = "light" | "dark";
export type Theme = EffectiveTheme;

const STORAGE_KEY = "iroha-theme";
const THEME_COLOR_DARK = "#0b0f19";
const THEME_COLOR_LIGHT = "#f7f6f2";

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

export const themeState = $state<{
  preference: ThemePreference;
  effective: EffectiveTheme;
}>({
  preference: getThemePreference(),
  effective: getEffectiveTheme(),
});

function syncHeadMeta(pref: ThemePreference, effective: EffectiveTheme): void {
  if (typeof document === "undefined") return;
  const colorSchemeMeta = document.querySelector<HTMLMetaElement>(
    'meta[name="color-scheme"]',
  );
  if (colorSchemeMeta) {
    colorSchemeMeta.content = pref === "auto" ? "light dark" : pref;
  }
  const themeColorMeta = document.querySelector<HTMLMetaElement>(
    'meta[name="theme-color"]',
  );
  if (themeColorMeta) {
    themeColorMeta.content =
      effective === "light" ? THEME_COLOR_LIGHT : THEME_COLOR_DARK;
  }
  const favicon = document.querySelector<HTMLLinkElement>("#theme-favicon");
  if (favicon) {
    favicon.href =
      effective === "light" ? "/favicon-light.svg?v=3" : "/favicon.svg?v=3";
  }
}

function applyThemeInternal(
  pref: ThemePreference,
  persist: boolean,
): EffectiveTheme {
  const effective: EffectiveTheme =
    pref === "auto" ? (prefersLight() ? "light" : "dark") : pref;

  if (typeof document !== "undefined") {
    document.documentElement.dataset.theme = effective;
    document.documentElement.dataset.themePreference = pref;
    syncHeadMeta(pref, effective);
  }

  themeState.preference = pref;
  themeState.effective = effective;

  if (persist && typeof localStorage !== "undefined") {
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

export function applyTheme(pref: ThemePreference): EffectiveTheme {
  return applyThemeInternal(pref, true);
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
  const initialPref = getThemePreference();
  applyThemeInternal(initialPref, false);

  if (typeof window === "undefined") {
    return () => {};
  }

  const cleanups: (() => void)[] = [];

  if (typeof window.matchMedia === "function") {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: light)");
    const onChange = () => {
      // If the preference is auto, follow system changes; never clear an explicit override.
      if (getThemePreference() === "auto") {
        applyThemeInternal("auto", false);
      }
    };
    mediaQuery.addEventListener("change", onChange);
    cleanups.push(() => mediaQuery.removeEventListener("change", onChange));
  }

  const onStorage = (event: StorageEvent) => {
    if (event.key !== null && event.key !== STORAGE_KEY) return;
    applyThemeInternal(getThemePreference(), false);
  };
  window.addEventListener("storage", onStorage);
  cleanups.push(() => window.removeEventListener("storage", onStorage));

  return () => {
    for (const cleanup of cleanups) cleanup();
  };
}
