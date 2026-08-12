// Shapes mirror apps/iroha-server/pkg/publicexport's JSON output exactly --
// this site has no live API, so these types describe the static
// summary.json/activities.json/routes.geojson snapshot instead of a fetch
// contract.

export interface Activity {
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

export interface ActivityDetailRoutePoint {
  seq: number;
  ts?: string;
  lat: number;
  lon: number;
  elevation_m?: number;
  distance_m?: number;
  speed_mps?: number;
  heart_rate?: number;
}

export interface ActivityDetailSampling {
  id: string;
  sampling_type: string;
  ts: string;
  value: number;
  unit: string;
}

export interface ActivityDetailLap {
  id: string;
  lap_no: number;
  start_ts?: string;
  end_ts?: string;
  distance_m?: number;
  duration_s?: number;
  avg_hr?: number;
  avg_pace_s_per_km?: number;
}

export interface ActivityDetail {
  activity: Activity & { source_kind: string };
  route: ActivityDetailRoutePoint[];
  samplings: ActivityDetailSampling[];
  laps: ActivityDetailLap[];
}

export interface SummaryTotals {
  activity_count: number;
  distance_m: number;
  duration_s: number;
  moving_time_s: number;
}

export interface SummaryBucket {
  key: string;
  activity_count: number;
  distance_m: number;
  duration_s: number;
  moving_time_s: number;
}

export interface Summary {
  totals: SummaryTotals;
  by_year: SummaryBucket[];
  by_month: SummaryBucket[];
  by_sport: SummaryBucket[];
}

export interface RouteFeatureProperties {
  activity_id?: string;
  sport_type: string;
  year: string;
  city?: string;
  city_status?: "pending" | "resolved" | "unknown";
}

export interface RouteFeature {
  type: "Feature";
  geometry: {
    type: "LineString";
    coordinates: [number, number][];
  };
  properties: RouteFeatureProperties;
}

export interface RouteFeatureCollection {
  type: "FeatureCollection";
  features: RouteFeature[];
}

export interface Meta {
  generated_at: string;
  routes_included?: boolean;
  activity_count?: number;
}
