export const THEME_IDS = [
  "atlas",
  "grapher",
  "field-journal",
  "phenology",
  "sound-map",
  "archive",
] as const;

export type DesignLanguage = (typeof THEME_IDS)[number];

export const THEME_ROUTES = [
  "today",
  "dashboard",
  "daily",
  "activities",
  "activity-detail",
  "sleep",
  "media",
  "media-detail",
  "expenses",
  "reports",
] as const;

export type ThemeRoute = (typeof THEME_ROUTES)[number];
export type ThemeImplementationStatus = "palette-only" | "preview" | "curated";

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

export type ThemeImplementation<Component> = {
  implementation: ThemeImplementationStatus;
  components: Partial<Record<ThemeRoute, Component>> & { shell: Component };
};

export type ThemeDefinition<Component> = {
  identity: ThemeIdentity;
  implementation: ThemeImplementationStatus;
  routes: readonly ThemeRoute[];
  components: Partial<Record<ThemeRoute, Component>> & { shell: Component };
};

export type ThemeRegistry<Component> = {
  definitions: readonly ThemeDefinition<Component>[];
  get(language: DesignLanguage): ThemeDefinition<Component>;
  hasRoute(theme: ThemeDefinition<Component>, route: ThemeRoute): boolean;
};

export function defineThemeRegistry<Component>(
  implementations: Record<DesignLanguage, ThemeImplementation<Component>>,
): ThemeRegistry<Component> {
  const definitions = THEME_IDS.map((id) => {
    const entry = implementations[id];
    const routes = Object.keys(entry.components).filter(
      (route): route is ThemeRoute => route !== "shell",
    );
    for (const route of routes) {
      if (!THEME_ROUTES.some((known) => known === route)) {
        throw new Error(`Unknown theme route: ${route}`);
      }
    }
    return {
      identity: THEME_IDENTITIES[id],
      implementation: entry.implementation,
      routes,
      components: entry.components,
    };
  });

  return {
    definitions,
    get(language) {
      const definition = definitions.find(
        (item) => item.identity.id === language,
      );
      if (!definition) {
        throw new Error(`Unknown Iroha design language: ${language}`);
      }
      return definition;
    },
    hasRoute(theme, route) {
      return theme.routes.includes(route);
    },
  };
}

export function isDesignLanguage(
  value: string | null | undefined,
): value is DesignLanguage {
  return THEME_IDS.some((id) => id === value);
}
