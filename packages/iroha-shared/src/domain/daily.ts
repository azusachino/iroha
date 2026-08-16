export interface DailyRing {
  move_kcal: number;
  move_goal_kcal: number;
  exercise_min: number;
  exercise_goal_min: number;
  stand_hours: number;
  stand_goal_hours: number;
}

export interface DailyRow {
  id: string;
  day: string;
  ring: DailyRing | null;
  steps?: number;
  distance_km?: number;
  flights?: number;
  resting_hr?: number;
  walking_hr_avg?: number;
  hrv_sdnn?: number;
  spo2_avg?: number;
  spo2_min?: number;
  respiratory_rate?: number;
  vo2max?: number;
  body_mass_kg?: number;
  source: string;
  first_raw_file_id: string;
  created_at: string;
  updated_at: string;
}

export interface ListDailyParams {
  // Canonical scalar calendar scope; the server resolves it into from/to.
  date?: string;
  // Canonical calendar range: from inclusive, to exclusive.
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
}

export type DailyDates = string[];

export interface DailyMetricAggregate {
  metric: string;
  value: number;
  unit: string;
  observed_days: number;
}

export interface DailyAggregateBucket {
  period: string;
  days: number;
  move_kcal_avg: number;
  exercise_min_avg: number;
  stand_hours_avg: number;
  move_closed_pct: number;
  metrics: DailyMetricAggregate[];
}

export interface DailyAggregates {
  granularity: "month" | "year";
  buckets: DailyAggregateBucket[];
}
