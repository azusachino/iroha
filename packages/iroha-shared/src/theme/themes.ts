export const THEME_IDS = ["grapher"] as const;

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
  "metrics",
] as const;

export type ThemeRoute = (typeof THEME_ROUTES)[number];
export type ThemeImplementationStatus = "palette-only" | "preview" | "curated";

export type ThemePageLens = {
  question: string;
  lead: string;
  time: string;
  interaction: string;
  detail: string;
  avoid: string;
};

export type ThemeIdentity = {
  id: DesignLanguage;
  label: string;
  hint: string;
  description: string;
  mark: string;
  swatch: string;
  lenses: {
    expenses: ThemePageLens;
    reports: ThemePageLens;
  };
};

export const THEME_IDENTITIES = {
  grapher: {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
    mark: "↗",
    swatch: "#00d2b4",
    lenses: {
      expenses: {
        question: "How did spending move?",
        lead: "Daily and category comparisons with unit-safe money values.",
        time: "Selected month on a canonical day axis.",
        interaction: "Read the curve, change dimensions, inspect exact rows.",
        detail: "The ledger and table are the evidence layer.",
        avoid: "Do not compare currencies as if they were one value.",
      },
      reports: {
        question: "What changed across the year?",
        lead: "Aligned twelve-month comparisons across all available domains.",
        time: "Twelve monthly points ending at the selected month.",
        interaction: "Read slopes and deltas, then inspect the selected month.",
        detail: "Exact section facts and method metadata follow charts.",
        avoid:
          "Do not draw a trend for an empty month or annualize a partial one.",
      },
    },
  },
} as const satisfies Record<DesignLanguage, ThemeIdentity>;

export type ThemePrimitives<Language extends DesignLanguage = DesignLanguage> =
  {
    periodControl: { appearance: Language };
  };

export type ThemeImplementation<
  Component,
  Language extends DesignLanguage = DesignLanguage,
> = {
  implementation: ThemeImplementationStatus;
  primitives: ThemePrimitives<Language>;
  components: Record<ThemeRoute, Component> & { shell: Component };
};

export type ThemeImplementations<Component> = {
  [Language in DesignLanguage]: ThemeImplementation<Component, Language>;
};

export type ThemeDefinition<Component> = {
  identity: ThemeIdentity;
  implementation: ThemeImplementationStatus;
  primitives: ThemePrimitives;
  routes: readonly ThemeRoute[];
  components: Record<ThemeRoute, Component> & { shell: Component };
};

export type ThemeRegistry<Component> = {
  definitions: readonly ThemeDefinition<Component>[];
  get(language: DesignLanguage): ThemeDefinition<Component>;
  hasRoute(theme: ThemeDefinition<Component>, route: ThemeRoute): boolean;
};

export function defineThemeRegistry<Component>(
  implementations: ThemeImplementations<Component>,
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
      primitives: entry.primitives,
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
