import type {
  ActivitySummary,
  ActivitySummaryBucket,
  ActivitySummaryTotals,
  Lap,
  RouteFeature,
  RouteFeatureCollection,
  RouteFeatureProperties,
  RoutePoint,
  SamplingPoint,
} from "./activity";

// The public export is a privacy-trimmed projection of the canonical activity
// record. Keeping its wire shape here prevents the static site from inventing
// a second frontend-only contract.
export interface PublicActivity {
  id: string;
  sport_type: string;
  title: string;
  started_at: string;
  ended_at?: string;
  timezone: string;
  distance_m?: number;
  duration_s?: number;
  moving_time_s?: number;
  elevation_gain_m?: number;
  avg_hr?: number;
  max_hr?: number;
  avg_pace_s_per_km?: number;
}

export interface PublicActivityDetail {
  activity: PublicActivity & { source_kind: string };
  route: RoutePoint[];
  samplings: SamplingPoint[];
  laps: Lap[];
}

export type PublicActivityDetailRoutePoint = RoutePoint;
export type PublicActivityDetailSampling = SamplingPoint;
export type PublicActivityDetailLap = Lap;

export type PublicSummary = ActivitySummary;
export type PublicSummaryTotals = ActivitySummaryTotals;
export type PublicSummaryBucket = ActivitySummaryBucket;

export type PublicRouteFeatureProperties = RouteFeatureProperties;
export type PublicRouteFeature = RouteFeature;
export type PublicRouteFeatureCollection = RouteFeatureCollection;

export interface PublicMeta {
  generated_at: string;
  routes_included?: boolean;
  activity_count?: number;
}
