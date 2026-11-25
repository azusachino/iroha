import {
  THEME_IDENTITIES,
  THEME_ROUTES,
  isDesignLanguage,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeRoute,
} from "../theme/themes";
import GrapherActivities from "./grapher/Activities.svelte";
import GrapherActivityDetail from "./grapher/ActivityDetail.svelte";
import GrapherDaily from "./grapher/Daily.svelte";
import GrapherDashboard from "./grapher/Dashboard.svelte";
import GrapherExpenses from "./grapher/Expenses.svelte";
import GrapherMedia from "./grapher/Media.svelte";
import GrapherMediaDetail from "./grapher/MediaDetail.svelte";
import GrapherReports from "./grapher/Reports.svelte";
import GrapherShell from "./grapher/Shell.svelte";
import GrapherSleep from "./grapher/Sleep.svelte";
import GrapherToday from "./grapher/Today.svelte";

export const GRAPHER_COMPONENTS = {
  shell: GrapherShell,
  today: GrapherToday,
  daily: GrapherDaily,
  activities: GrapherActivities,
  "activity-detail": GrapherActivityDetail,
  sleep: GrapherSleep,
  media: GrapherMedia,
  "media-detail": GrapherMediaDetail,
  dashboard: GrapherDashboard,
  expenses: GrapherExpenses,
  reports: GrapherReports,
} as const;

export type ThemeComponents = typeof GRAPHER_COMPONENTS;

export const GRAPHER_THEME: ThemeDefinition<ThemeComponents> = {
  identity: THEME_IDENTITIES.grapher,
  implementation: "curated",
  primitives: { periodControl: { appearance: "grapher" } },
  routes: THEME_ROUTES,
  components: GRAPHER_COMPONENTS,
};

export const THEME_DEFINITIONS: readonly ThemeDefinition<ThemeComponents>[] = [
  GRAPHER_THEME,
];

export { isDesignLanguage };

export function getThemeDefinition(
  language: DesignLanguage,
): ThemeDefinition<ThemeComponents> {
  if (language !== "grapher") {
    throw new Error(`Unknown Iroha design language: ${language}`);
  }
  return GRAPHER_THEME;
}

export function hasThemeRoute(
  theme: ThemeDefinition<ThemeComponents>,
  route: ThemeRoute,
): boolean {
  return theme.routes.includes(route);
}
