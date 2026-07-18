import { THEME_DEFINITIONS, isDesignLanguage } from "$lib/themes/registry";
import type { DesignLanguage } from "$lib/themes/types";

export type { DesignLanguage } from "$lib/themes/types";

export const DESIGN_LANGUAGES = THEME_DEFINITIONS;

const STORAGE_KEY = "iroha-design-language";
const DEFAULT_LANGUAGE: DesignLanguage = "field-journal";

export function getDesignLanguage(): DesignLanguage {
  if (typeof document === "undefined") return DEFAULT_LANGUAGE;
  const attr = document.documentElement.dataset.language;
  return isDesignLanguage(attr) ? attr : DEFAULT_LANGUAGE;
}

export function setDesignLanguage(language: DesignLanguage): void {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.language = language;
  try {
    localStorage.setItem(STORAGE_KEY, language);
  } catch {
    // Keep the in-DOM selection when storage is unavailable.
  }
}
