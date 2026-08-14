import type { Snippet } from "svelte";
import type { Activity, ActivityDisplaySummary } from "./activity";
import type { MetricSeriesResponse } from "./metric-series";
import type { DesignLanguage } from "./themes";

export type ActivityThemeProps = {
  activities: Activity[];
  displaySummary: ActivityDisplaySummary;
  sportType: string;
  sportOptions: string[];
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  loadingMore: boolean;
  onSportType: (value: string) => void;
  onLoadMore: () => void;
  onOpenDetail: (id: string) => void;
  activitySeries?: MetricSeriesResponse | null;
  activityDurationSeries?: MetricSeriesResponse | null;
  activitySeriesLoading?: boolean;
  activitySeriesError?: string | null;
  activitySeriesScope?: string;
  theme: DesignLanguage;
  children?: Snippet;
};
