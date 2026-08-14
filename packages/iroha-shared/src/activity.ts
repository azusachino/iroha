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
  source_kind: string;
  source_activity_id?: string;
  first_raw_file_id: string;
  created_at: string;
  updated_at: string;
}

export interface ActivityDisplaySummary {
  activity_count: number;
  distance_m: number;
  duration_s: number;
}

export interface ActivitySummaryTotals {
  activity_count: number;
  distance_m: number;
  distance_known_count: number;
  distance_unknown_count: number;
  duration_s: number;
  elevation_gain_m: number;
  // Legacy presentation data is not emitted by the canonical summary API.
  moving_time_s?: number;
}

export interface ActivitySummaryBucket {
  key: string;
  activity_count: number;
  distance_m: number;
  distance_known_count: number;
  distance_unknown_count: number;
  duration_s: number;
  elevation_gain_m: number;
  moving_time_s?: number;
}

export interface ActivitySummary {
  totals: ActivitySummaryTotals;
  by_year: ActivitySummaryBucket[];
  by_month: ActivitySummaryBucket[];
  by_sport: ActivitySummaryBucket[];
}

export interface ActivityActiveDay {
  day: string;
  activity_count: number;
}

export interface ActivityOverview {
  summary: ActivitySummary;
  active_days: ActivityActiveDay[];
  recent: Activity[];
  current_streak: number;
}

// Privacy-trimmed route geometry returned by the activity overview API.
// Map rendering stays in the host application; the geometry contract is
// shared so visual compositions do not need to know about the map library.
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

export interface RoutePoint {
  seq: number;
  ts?: string;
  lat: number;
  lon: number;
  elevation_m?: number;
  distance_m?: number;
  speed_mps?: number;
  heart_rate?: number;
}

export interface SamplingPoint {
  id: string;
  sampling_type: string;
  ts: string;
  value: number;
  unit: string;
}

export interface Lap {
  id: string;
  lap_no: number;
  start_ts?: string;
  end_ts?: string;
  distance_m?: number;
  duration_s?: number;
  avg_hr?: number;
  avg_pace_s_per_km?: number;
}

export interface ListActivitiesParams {
  sport_type?: string;
  // RFC3339 timestamps; started_from is inclusive and started_to is exclusive.
  started_from?: string;
  started_to?: string;
  // Distance bounds in meters; rows with no distance are excluded.
  min_distance_m?: number;
  max_distance_m?: number;
  limit?: number;
  cursor?: string;
}
