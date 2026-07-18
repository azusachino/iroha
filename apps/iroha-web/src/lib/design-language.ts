export const DESIGN_LANGUAGES = [
  { id: "atlas", label: "Iroha Atlas", hint: "places and routes" },
  { id: "grapher", label: "Iroha Grapher", hint: "trends and comparisons" },
  {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
  },
  { id: "phenology", label: "Iroha Phenology", hint: "sleep and seasons" },
  { id: "sound-map", label: "Iroha Sound Map", hint: "rhythm and intensity" },
  { id: "archive", label: "Iroha Archive", hint: "media and history" },
] as const;

export type DesignLanguage = (typeof DESIGN_LANGUAGES)[number]["id"];

const STORAGE_KEY = "iroha-design-language";
const DEFAULT_LANGUAGE: DesignLanguage = "field-journal";

function isDesignLanguage(
  value: string | null | undefined,
): value is DesignLanguage {
  return DESIGN_LANGUAGES.some((language) => language.id === value);
}

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
