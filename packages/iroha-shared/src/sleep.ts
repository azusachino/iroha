export interface SleepSession {
  id: string;
  wake_date: string;
  started_at: string;
  ended_at: string;
  time_in_bed_s: number;
  asleep_s: number;
  efficiency: number;
  is_main_sleep: boolean;
  core_s: number;
  deep_s: number;
  rem_s: number;
  awake_s: number;
  unspecified_s: number;
  source: string;
  first_raw_file_id: string;
  created_at: string;
  updated_at: string;
}

export interface SleepSegment {
  id: string;
  stage: string;
  started_at: string;
  ended_at: string;
  seq: number;
}

export interface SleepAggregateBucket {
  period: string;
  session_count: number;
  main_sleep_count: number;
  nap_count: number;
  observed_wake_dates: number;
  average_asleep_s: number;
  average_time_in_bed_s: number;
  average_efficiency: number;
  core_s: number;
  deep_s: number;
  rem_s: number;
  awake_s: number;
  unspecified_s: number;
}

export interface SleepAggregates {
  granularity: "month" | "year" | "lifetime";
  buckets: SleepAggregateBucket[];
}

export interface SleepOverview {
  session_count: number;
  main_sleep_count: number;
  average_asleep_s: number;
  average_efficiency: number;
}

export interface ListSleepParams {
  // Canonical calendar range: from inclusive, to exclusive.
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
}
