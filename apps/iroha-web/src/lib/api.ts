import { API_BASE, API_TOKEN } from "./config";

// Types mirror the iroha-server read API JSON contract (snake_case).
// Optional fields use `?` because the server omits them when absent.

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
  granularity: "month" | "year";
  buckets: SleepAggregateBucket[];
}

export interface ListSleepParams {
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
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
  // RFC3339 timestamps; inclusive bounds on started_at.
  started_from?: string;
  started_to?: string;
  // Distance bounds in meters; rows with no distance are excluded.
  min_distance_m?: number;
  max_distance_m?: number;
  limit?: number;
  cursor?: string;
}

export interface MediaRow {
  id: string;
  title: string;
  media_type: string;
  item_role: string;
  cover_image_url?: string;
  status?: string;
  position?: number;
  total?: number;
  unit?: string;
  progress_percent?: number;
  last_update_at: string;
  rating?: number;
  hidden_from_continue?: boolean;
  native_title?: string;
}

export interface ListMediaParams {
  status?: string;
  media_type?: string;
  family?: string;
  completed_year?: number;
  limit?: number;
  cursor?: string;
}

export interface MediaCompletionBucket {
  year: number;
  count: number;
}

export interface MediaScoreBucket {
  score: number;
  count: number;
}

export interface MediaTypeBucket {
  type: string;
  count: number;
}

export interface MediaAggregates {
  totals: {
    item_count: number;
    completed_count: number;
    this_year_completed: number;
    average_rating: number;
  };
  completions_by_year: MediaCompletionBucket[];
  score_distribution: MediaScoreBucket[];
  type_split: MediaTypeBucket[];
}

export interface MediaDetail {
  item: MediaRow;
  work: {
    id: string;
    work_kind: string;
    primary_title: string;
    original_title: string;
    original_language: string;
    first_release_date?: string;
    description: string;
  };
  progress?: {
    status: string;
    unit: string;
    position?: number;
    total?: number;
    progress_percent?: number;
    started_at?: string;
    last_update_at?: string;
    finished_at?: string;
    play_count: number;
  };
  creators: { id: string; name: string; role: string }[];
  relations: {
    id: string;
    relation_type: string;
    direction: string;
    related_item_id: string;
    related_title: string;
    related_type: string;
    cover_image_url?: string;
  }[];
  events: {
    id: string;
    event_type: string;
    event_at?: string;
    unit?: string;
    position?: number;
    total?: number;
    progress_percent?: number;
    rating?: number;
    note?: string;
  }[];
}

export interface MediaHomeEvent {
  id: string;
  media_id: string;
  title: string;
  cover_image_url?: string;
  event_type: string;
  occurred_at: string;
  unit?: string;
  position?: number;
  total?: number;
  progress_percent?: number;
  rating?: number;
}

export interface ListMediaEventsParams {
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
}

// One keyset page. `next_cursor` is null when no further rows exist.
export interface Page<T> {
  items: T[];
  next_cursor: string | null;
  has_more: boolean;
  status_counts?: Record<string, number>;
  active_count?: number;
}

export interface BriefingSection<T = unknown> {
  key: string;
  schema: string;
  state: "ready" | "empty" | "unavailable";
  data: T;
}

export interface BriefingResponse {
  date: string;
  previous_date: string;
  next_date: string;
  sections: BriefingSection[];
}

export function getBriefing(
  date: string,
  fetchFn: typeof fetch = fetch,
): Promise<BriefingResponse> {
  return getJSON<BriefingResponse>(
    `/api/v1/briefing?date=${encodeURIComponent(date)}`,
    fetchFn,
  );
}

// One day of the daily-activity + body-vitals module. Rings are always present
// (zeroed on non-ring days); every scalar metric is optional because a day may
// have some vitals but no ring, or vice versa.
export interface DailyRow {
  id: string;
  day: string;
  move_kcal: number;
  move_goal_kcal: number;
  exercise_min: number;
  exercise_goal_min: number;
  stand_hours: number;
  stand_goal_hours: number;
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
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
}

// One month/year rollup. Ring fields are per-day averages over ring days;
// `metrics` is a per-day average keyed by metric slug (steps, resting_hr, …),
// open-ended to match tb_daily_metrics.
export interface DailyAggregateBucket {
  period: string;
  days: number;
  move_kcal_avg: number;
  exercise_min_avg: number;
  stand_hours_avg: number;
  move_closed_pct: number;
  metrics: Record<string, number>;
}

export interface DailyAggregates {
  granularity: "month" | "year";
  buckets: DailyAggregateBucket[];
}

export function listDailyAggregates(
  granularity: "month" | "year",
  params: { from?: string; to?: string } = {},
  fetchFn: typeof fetch = fetch,
): Promise<DailyAggregates> {
  const query = new URLSearchParams({ granularity });
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  return getJSON<DailyAggregates>(
    `/api/v1/daily/aggregates?${query.toString()}`,
    fetchFn,
  );
}

async function getJSON<T>(
  path: string,
  fetchFn: typeof fetch = fetch,
): Promise<T> {
  const headers: HeadersInit = { accept: "application/json" };
  if (API_TOKEN) {
    headers.authorization = `Bearer ${API_TOKEN}`;
  }
  const res = await fetchFn(`${API_BASE}${path}`, {
    headers,
  });
  if (!res.ok) {
    throw new Error(
      `request failed: ${res.status} ${res.statusText} (${path})`,
    );
  }
  return (await res.json()) as T;
}

export function listActivities(
  params: ListActivitiesParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<Activity>> {
  const query = new URLSearchParams();
  if (params.sport_type) query.set("sport_type", params.sport_type);
  if (params.started_from) query.set("started_from", params.started_from);
  if (params.started_to) query.set("started_to", params.started_to);
  if (params.min_distance_m != null)
    query.set("min_distance_m", String(params.min_distance_m));
  if (params.max_distance_m != null)
    query.set("max_distance_m", String(params.max_distance_m));
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<Activity>>(`/api/v1/activities${suffix}`, fetchFn);
}

export function listMedia(
  params: ListMediaParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<MediaRow>> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.media_type) query.set("media_type", params.media_type);
  if (params.family) query.set("family", params.family);
  if (params.completed_year != null)
    query.set("completed_year", String(params.completed_year));
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<MediaRow>>(`/api/v1/media${suffix}`, fetchFn);
}

export function getMediaAggregates(
  fetchFn: typeof fetch = fetch,
): Promise<MediaAggregates> {
  return getJSON<MediaAggregates>("/api/v1/media/aggregates", fetchFn);
}

export function getMedia(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<MediaDetail> {
  return getJSON<MediaDetail>(
    `/api/v1/media/${encodeURIComponent(id)}`,
    fetchFn,
  );
}

export function listMediaEvents(
  params: ListMediaEventsParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<MediaHomeEvent>> {
  const query = new URLSearchParams();
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<MediaHomeEvent>>(
    `/api/v1/media/events${suffix}`,
    fetchFn,
  );
}

export function listSleep(
  params: ListSleepParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<SleepSession>> {
  const query = new URLSearchParams();
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<SleepSession>>(`/api/v1/sleep${suffix}`, fetchFn);
}

export function getSleepSegments(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<SleepSegment[]> {
  return getJSON<SleepSegment[]>(
    `/api/v1/sleep/${encodeURIComponent(id)}/segments`,
    fetchFn,
  );
}

export function listSleepAggregates(
  granularity: "month" | "year",
  params: { from?: string; to?: string } = {},
  fetchFn: typeof fetch = fetch,
): Promise<SleepAggregates> {
  const query = new URLSearchParams({ granularity });
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  return getJSON<SleepAggregates>(
    `/api/v1/sleep/aggregates?${query.toString()}`,
    fetchFn,
  );
}

export function listDaily(
  params: ListDailyParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<DailyRow>> {
  const query = new URLSearchParams();
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<DailyRow>>(`/api/v1/daily${suffix}`, fetchFn);
}

export function getActivity(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<Activity> {
  return getJSON<Activity>(
    `/api/v1/activities/${encodeURIComponent(id)}`,
    fetchFn,
  );
}

export function getActivityRoute(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<RoutePoint[]> {
  return getJSON<RoutePoint[]>(
    `/api/v1/activities/${encodeURIComponent(id)}/route`,
    fetchFn,
  );
}

export function getActivitySamplings(
  id: string,
  types?: string[],
  fetchFn: typeof fetch = fetch,
): Promise<SamplingPoint[]> {
  const suffix =
    types && types.length
      ? `?type=${types.map(encodeURIComponent).join(",")}`
      : "";
  return getJSON<SamplingPoint[]>(
    `/api/v1/activities/${encodeURIComponent(id)}/samplings${suffix}`,
    fetchFn,
  );
}

export function getActivityLaps(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<Lap[]> {
  return getJSON<Lap[]>(
    `/api/v1/activities/${encodeURIComponent(id)}/laps`,
    fetchFn,
  );
}

// --- Public API (sanitized, no auth) ---
// Mirrors iroha-server's /public/v1 routes. Activities carry a strict subset
// of fields — no source/raw-file identifiers or other private metadata.

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

export interface ListPublicActivitiesParams {
  sport_type?: string;
  started_from?: string;
  started_to?: string;
  min_distance_m?: number;
  max_distance_m?: number;
  limit?: number;
  cursor?: string;
}

export interface SummaryTotals {
  activity_count: number;
  distance_m: number;
  duration_s: number;
  moving_time_s: number;
}

// A single row in one of the summary's grouped breakdowns (by_year / by_month
// / by_sport). `key` is a year ("2026"), a "YYYY-MM" month, or a sport_type.
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

export interface PublicSummaryParams {
  // Scope every breakdown to one calendar year and/or one sport_type. Omit for
  // all-time / all-sport totals.
  year?: string | null;
  sport?: string | null;
}

export function getPublicSummary(
  params: PublicSummaryParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Summary> {
  const query = new URLSearchParams();
  if (params.year) query.set("year", params.year);
  if (params.sport) query.set("sport", params.sport);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Summary>(`/public/v1/summary${suffix}`, fetchFn);
}

// A single public route line, rendered as a GeoJSON LineString. Coordinates
// are [lon, lat] pairs (GeoJSON order), already privacy-trimmed and
// decimated by the server.
export interface RouteFeatureProperties {
  sport_type: string;
  year: string;
  city?: string;
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

export function getPublicRoutes(
  fetchFn: typeof fetch = fetch,
): Promise<RouteFeatureCollection> {
  return getJSON<RouteFeatureCollection>("/public/v1/routes", fetchFn);
}

export function listPublicActivities(
  params: ListPublicActivitiesParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<PublicActivity>> {
  const query = new URLSearchParams();
  if (params.sport_type) query.set("sport_type", params.sport_type);
  if (params.started_from) query.set("started_from", params.started_from);
  if (params.started_to) query.set("started_to", params.started_to);
  if (params.min_distance_m != null)
    query.set("min_distance_m", String(params.min_distance_m));
  if (params.max_distance_m != null)
    query.set("max_distance_m", String(params.max_distance_m));
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<PublicActivity>>(
    `/public/v1/activities${suffix}`,
    fetchFn,
  );
}
