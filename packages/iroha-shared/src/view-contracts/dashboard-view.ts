import type { Snippet } from "svelte";
import type {
  Activity,
  ActivityActiveDay,
  ActivitySummary,
  RouteFeatureCollection,
} from "../domain/activity";
import type { MediaAggregates } from "../domain/media";
import type { DesignLanguage } from "../theme/themes";

export interface DashboardSleepSummary {
  averageAsleepS: number;
  averageEfficiency: number;
  nightCount: number;
}

export type DashboardThemeProps = {
  summary: ActivitySummary | null;
  activeDays: ActivityActiveDay[];
  heatmapEndDay: string;
  activities: Activity[];
  routes: RouteFeatureCollection | null;
  streak: string;
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  onOpenActivity: (id: string) => void;
  onOpenSport: (sport: string) => void;
  sleepSummary: DashboardSleepSummary;
  sleepLoading: boolean;
  sleepError: string | null;
  mediaAggregates: MediaAggregates | null;
  mediaLoading: boolean;
  mediaError: string | null;
  theme: DesignLanguage;
  // The host supplies this because route maps depend on the host's map
  // runtime and fetch lifecycle, not on the shared theme package.
  children?: Snippet;
};
