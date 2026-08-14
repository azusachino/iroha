import type { Snippet } from "svelte";
import type {
  Activity,
  ActivitySummary,
  RouteFeatureCollection,
} from "./activity";
import type { MediaAggregates } from "./media";
import type { DesignLanguage } from "./themes";

export interface DashboardSleepSummary {
  averageAsleepS: number;
  averageEfficiency: number;
  nightCount: number;
}

export type DashboardThemeProps = {
  summary: ActivitySummary | null;
  activities: Activity[];
  routes: RouteFeatureCollection | null;
  streak: string;
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  routesLoading: boolean;
  routesError: string | null;
  onLoadRoutes: () => void;
  onOpenActivity: (id: string) => void;
  onOpenSport: (sport: string) => void;
  sleepSummary: DashboardSleepSummary;
  mediaAggregates: MediaAggregates | null;
  theme: DesignLanguage;
  // The host supplies this because route maps depend on the host's map
  // runtime and fetch lifecycle, not on the shared theme package.
  children?: Snippet;
};
