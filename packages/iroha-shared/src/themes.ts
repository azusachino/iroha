export const THEME_IDS = [
  "atlas",
  "grapher",
  "field-journal",
  "phenology",
  "sound-map",
  "archive",
] as const;

export type DesignLanguage = (typeof THEME_IDS)[number];

export type ThemeIdentity = {
  id: DesignLanguage;
  label: string;
  hint: string;
  description: string;
};

export const THEME_IDENTITIES = {
  atlas: {
    id: "atlas",
    label: "Iroha Atlas",
    hint: "places and routes",
    description: "A cartographic language for movement, places, and distance.",
  },
  grapher: {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
  },
  "field-journal": {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
    description:
      "A dated, observational language for entries, continuity, and the shape of a day.",
  },
  phenology: {
    id: "phenology",
    label: "Iroha Phenology",
    hint: "sleep and seasons",
    description:
      "A cyclical language for recovery, rest, and unfolding patterns.",
  },
  "sound-map": {
    id: "sound-map",
    label: "Iroha Sound Map",
    hint: "rhythm and intensity",
    description: "A rhythmic language for cadence, intensity, and flow.",
  },
  archive: {
    id: "archive",
    label: "Iroha Archive",
    hint: "media and history",
    description: "A chronological language for collections and memory.",
  },
} as const satisfies Record<DesignLanguage, ThemeIdentity>;

export function isDesignLanguage(
  value: string | null | undefined,
): value is DesignLanguage {
  return THEME_IDS.some((id) => id === value);
}
