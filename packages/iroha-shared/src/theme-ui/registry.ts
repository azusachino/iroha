import type { Component } from "svelte";
import {
  defineThemeRegistry,
  isDesignLanguage,
  type DesignLanguage,
  type ThemeDefinition,
  type ThemeRoute,
} from "../theme/themes";
import GrapherActivities from "./grapher/Activities.svelte";
import GrapherDaily from "./grapher/Daily.svelte";
import GrapherShell from "./grapher/Shell.svelte";
import GrapherSleep from "./grapher/Sleep.svelte";
import GrapherToday from "./grapher/Today.svelte";
import GrapherDashboard from "./grapher/Dashboard.svelte";
import GrapherActivityDetail from "./grapher/ActivityDetail.svelte";
import GrapherMedia from "./grapher/Media.svelte";
import GrapherMediaDetail from "./grapher/MediaDetail.svelte";
import GrapherExpenses from "./grapher/Expenses.svelte";
import GrapherReports from "./grapher/Reports.svelte";

// Registry entries are intentionally heterogeneous. `never` erases their
// props at the shared boundary; each host supplies the route-specific props.
export type ThemeComponent = Component<never>;

const registry = defineThemeRegistry<ThemeComponent>({
  grapher: {
    implementation: "curated",
    primitives: { periodControl: { appearance: "grapher" } },
    components: {
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
    },
  },
});

export const THEME_DEFINITIONS = registry.definitions;

export { isDesignLanguage };

export function getThemeDefinition(
  language: DesignLanguage,
): ThemeDefinition<ThemeComponent> {
  return registry.get(language);
}

export function hasThemeRoute(
  theme: ThemeDefinition<ThemeComponent>,
  route: ThemeRoute,
): boolean {
  return registry.hasRoute(theme, route);
}
