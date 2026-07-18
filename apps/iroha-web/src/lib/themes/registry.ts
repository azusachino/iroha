import type { DesignLanguage, ThemeDefinition } from "$lib/themes/types";

export const THEME_DEFINITIONS = [
  {
    id: "atlas",
    label: "Iroha Atlas",
    hint: "places and routes",
    description: "A cartographic language for movement, places, and distance.",
    implementation: "palette-only",
  },
  {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
    implementation: "palette-only",
  },
  {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
    description: "A dated, narrative language for the shape of a day.",
    implementation: "palette-only",
  },
  {
    id: "phenology",
    label: "Iroha Phenology",
    hint: "sleep and seasons",
    description:
      "A cyclical language for recovery, rest, and unfolding patterns.",
    implementation: "palette-only",
  },
  {
    id: "sound-map",
    label: "Iroha Sound Map",
    hint: "rhythm and intensity",
    description: "A rhythmic language for cadence, intensity, and flow.",
    implementation: "palette-only",
  },
  {
    id: "archive",
    label: "Iroha Archive",
    hint: "media and history",
    description: "A chronological language for collections and memory.",
    implementation: "palette-only",
  },
] as const satisfies readonly ThemeDefinition[];

export function isDesignLanguage(
  value: string | null | undefined,
): value is DesignLanguage {
  return THEME_DEFINITIONS.some((theme) => theme.id === value);
}

export function getThemeDefinition(language: DesignLanguage): ThemeDefinition {
  const theme = THEME_DEFINITIONS.find((item) => item.id === language);
  if (!theme) throw new Error(`Unknown Iroha design language: ${language}`);
  return theme;
}
