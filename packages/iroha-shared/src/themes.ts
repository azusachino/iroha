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
  atlas: {
    id: "atlas",
    label: "Iroha Atlas",
    hint: "places and routes",
    description: "A cartographic language for movement, places, and distance.",
    mark: "N",
    swatch: "#58c7c0",
    lenses: {
      expenses: {
        question: "Where did the month travel?",
        lead: "A chronological spending survey with location only when it exists.",
        time: "Selected month, read from first day to last.",
        interaction: "Scan the survey, then drill into a canonical record.",
        detail: "Records and provenance follow the survey.",
        avoid: "Do not invent geography for expenses without a place.",
      },
      reports: {
        question: "What territory did the month cover?",
        lead: "A cross-domain monthly survey with drill-through to evidence.",
        time: "Selected month with a twelve-month horizon.",
        interaction: "Compare domain plates, then open their source detail.",
        detail: "Exact domain facts follow the overview.",
        avoid: "Do not turn an aggregate into a map without spatial data.",
      },
    },
  },
  grapher: {
    id: "grapher",
    label: "Iroha Grapher",
    hint: "trends and comparisons",
    description: "An evidence-first language for comparison and change.",
    mark: "↗",
    swatch: "#6da9ff",
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
        avoid: "Do not draw a trend for an empty month or annualize a partial one.",
      },
    },
  },
  "field-journal": {
    id: "field-journal",
    label: "Iroha Field Journal",
    hint: "days and evidence",
    description:
      "A dated, observational language for entries, continuity, and the shape of a day.",
    mark: "✎",
    swatch: "#39c5bb",
    lenses: {
      expenses: {
        question: "What did the days contain?",
        lead: "Dated entries that preserve notes, continuity, and gaps.",
        time: "A month as a sequence of observed days.",
        interaction: "Read the entry sequence, then open one record.",
        detail: "Canonical notes and source fields close the entry.",
        avoid: "Do not infer a cause or story that the record does not contain.",
      },
      reports: {
        question: "What was observed this month?",
        lead: "A dated monthly record that keeps observations and absences visible.",
        time: "Selected month with a twelve-month continuity strip.",
        interaction: "Follow the dated record, then inspect evidence.",
        detail: "Coverage and exact values remain at the bottom.",
        avoid: "Do not hide missing observations behind a polished score.",
      },
    },
  },
  phenology: {
    id: "phenology",
    label: "Iroha Phenology",
    hint: "sleep and seasons",
    description:
      "A cyclical language for recovery, rest, and unfolding patterns.",
    mark: "◒",
    swatch: "#d68bba",
    lenses: {
      expenses: {
        question: "What recurred through the month?",
        lead: "Category rhythm and repeated needs across calendar days.",
        time: "A month as a cycle rather than a ranked list.",
        interaction: "Trace intensity and recurrence, then inspect records.",
        detail: "The ledger grounds the cyclical reading.",
        avoid: "Do not call repetition a trend without a comparable period.",
      },
      reports: {
        question: "What phase is the year in?",
        lead: "Monthly cycles and recurring phases across the available domains.",
        time: "Twelve months as a seasonal loop ending at the selected month.",
        interaction: "Follow the cycle, then inspect the selected month’s facts.",
        detail: "Exact values and coverage remain explicit.",
        avoid: "Do not imply biological seasonality from a sparse observation.",
      },
    },
  },
  "sound-map": {
    id: "sound-map",
    label: "Iroha Sound Map",
    hint: "rhythm and intensity",
    description: "A rhythmic language for cadence, intensity, and flow.",
    mark: "≈",
    swatch: "#63b7ff",
    lenses: {
      expenses: {
        question: "Where did intensity arrive?",
        lead: "Bursts and quiet intervals rendered as spending intensity.",
        time: "A selected month on a cadence-oriented day axis.",
        interaction: "Follow bursts, then inspect the exact ledger.",
        detail: "Rows and provenance keep the signal honest.",
        avoid: "Do not pretend expense data is audio or fill quiet days as zero.",
      },
      reports: {
        question: "What cadence carried the month?",
        lead: "Monthly intensity and cadence bands across the cockpit.",
        time: "A twelve-month signal ending at the selected month.",
        interaction: "Compare bands, then open domain evidence.",
        detail: "The report preserves units, coverage, and method.",
        avoid: "Do not use decoration to suggest a relationship the data lacks.",
      },
    },
  },
  archive: {
    id: "archive",
    label: "Iroha Archive",
    hint: "media and history",
    description: "A chronological language for collections and memory.",
    mark: "№",
    swatch: "#b8a0ff",
    lenses: {
      expenses: {
        question: "What exactly was recorded?",
        lead: "The canonical ledger, source identity, and immutable record detail.",
        time: "Selected month in record order, with derived views secondary.",
        interaction: "Open the row first; use charts as an index to the archive.",
        detail: "Canonical record detail and source trail lead.",
        avoid: "Do not let a decorative aggregate replace the source record.",
      },
      reports: {
        question: "What does the record preserve?",
        lead: "A monthly folio of exact totals, provenance, and source coverage.",
        time: "Selected month with a twelve-month archival index.",
        interaction: "Turn through the index, then inspect the folio.",
        detail: "Exact facts and generation metadata close the report.",
        avoid: "Do not collapse provenance into an unexplained headline.",
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
