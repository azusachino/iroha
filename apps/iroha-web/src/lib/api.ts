import { API_BASE } from "./config";

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
  episode_count?: number;
  chapter_count?: number;
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

export interface Task {
  id: string;
  title: string;
  notes?: string;
  status: "open" | "completed" | "canceled";
  due_date?: string;
  priority: number;
  source: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface MediaResolutionTask {
  id: string;
  task_type: "dedupe_candidate" | "progress_conflict";
  status: "open" | "resolved" | "dismissed";
  candidates: Record<string, unknown>;
  resolution: Record<string, unknown>;
  created_at: string;
  resolved_at?: string;
}

export interface Job {
  id: string;
  kind: string;
  status: "queued" | "running" | "completed" | "failed" | "canceled";
  attempts: number;
  max_attempts: number;
  run_after: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export type ExpenseCurrency = "JPY" | "USD" | "EUR" | "GBP";
export type ExpenseCategory =
  | "food"
  | "groceries"
  | "transport"
  | "shopping"
  | "housing"
  | "utilities"
  | "health"
  | "entertainment"
  | "subscriptions"
  | "work"
  | "other";

export interface ExpenseItem {
  name: string;
  amount_minor?: number;
}

export interface ExpenseSource {
  kind: string;
  ref: string;
}

export interface Expense {
  id: string;
  occurred_on: string;
  currency: ExpenseCurrency;
  currency_exponent: number;
  amount_minor: number;
  category: ExpenseCategory;
  merchant: string;
  note: string;
  items: ExpenseItem[];
  source: ExpenseSource;
  created_at: string;
  updated_at: string;
}

export interface ExpenseInput {
  occurred_on: string;
  currency: ExpenseCurrency;
  amount_minor: number;
  category: ExpenseCategory;
  merchant?: string;
  note?: string;
  items?: ExpenseItem[];
}

export interface CreateExpenseInput extends ExpenseInput {
  source: ExpenseSource;
}

export interface ListExpensesParams {
  from?: string;
  to?: string;
  currency?: ExpenseCurrency;
  category?: ExpenseCategory;
  limit?: number;
  cursor?: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string;

  constructor(
    status: number,
    code: string,
    message: string,
    requestId: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
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
  native_title?: string;
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

export interface ReportSection<T> {
  schema: string;
  state: "available" | "empty" | "unavailable";
  data: T | null;
}

export interface MonthlyReport {
  schema: string;
  period: {
    kind: "month";
    month: string;
    from: string;
    to: string;
    timezone: string;
  };
  generated_at: string;
  sections: {
    movement: ReportSection<MovementReportData>;
    sleep: ReportSection<SleepReportData>;
    daily_health: ReportSection<DailyHealthReportData>;
    media: ReportSection<MediaReportData>;
    expenses: ReportSection<ExpensesReportData>;
  };
}

export interface MovementReportData {
  activity_count: number;
  distance_m: number;
  distance_activity_count: number;
  duration_s: number;
  by_sport: {
    sport: string;
    activity_count: number;
    distance_m: number;
    distance_activity_count: number;
    duration_s: number;
  }[];
}

export interface SleepReportData {
  session_count: number;
  main_sleep_count: number;
  nap_count: number;
  average_asleep_s: number;
  average_time_in_bed_s: number;
  average_efficiency: number;
  stage_seconds: {
    core: number;
    deep: number;
    rem: number;
    awake: number;
    unspecified: number;
  };
}

export interface DailyHealthReportData {
  observed_days: number;
  metric_averages: {
    metric: string;
    value: number;
    unit: string;
    observed_days: number;
  }[];
}

export interface MediaReportData {
  event_count: number;
  completed_count: number;
  rated_count: number;
  average_rating: number | null;
  by_kind: {
    kind: string;
    event_count: number;
    completed_count: number;
  }[];
  completed_items: {
    id: string;
    title: string;
    media_type: string;
    completed_at: string;
  }[];
}

export interface ExpensesReportData {
  expense_count: number;
  totals_by_currency: {
    currency: ExpenseCurrency;
    currency_exponent: number;
    amount_minor: number;
    expense_count: number;
  }[];
  by_category: {
    category: ExpenseCategory;
    currency: ExpenseCurrency;
    currency_exponent: number;
    amount_minor: number;
    expense_count: number;
  }[];
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

export function getMonthlyReport(
  month: string,
  timezone: string,
  fetchFn: typeof fetch = fetch,
): Promise<MonthlyReport> {
  const query = new URLSearchParams({ month, timezone });
  return getJSON<MonthlyReport>(
    `/api/v1/reports/monthly?${query.toString()}`,
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

async function requestJSON<T>(
  path: string,
  init: RequestInit = {},
  fetchFn: typeof fetch = fetch,
): Promise<T> {
  const res = await fetchFn(`${API_BASE}${path}`, {
    ...init,
    headers: { accept: "application/json", ...init.headers },
  });
  if (!res.ok) {
    let body: unknown;
    try {
      body = await res.json();
    } catch {
      body = undefined;
    }
    if (
      body &&
      typeof body === "object" &&
      "code" in body &&
      "message" in body &&
      typeof body.code === "string" &&
      typeof body.message === "string"
    ) {
      throw new ApiError(
        res.status,
        body.code,
        body.message,
        "request_id" in body && typeof body.request_id === "string"
          ? body.request_id
          : "",
      );
    }
    throw new Error(
      `request failed: ${res.status} ${res.statusText} (${path})`,
    );
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

async function getJSON<T>(path: string, fetchFn: typeof fetch = fetch) {
  return requestJSON<T>(path, {}, fetchFn);
}

async function mutateJSON<T>(
  path: string,
  method: "POST" | "PATCH" | "PUT" | "DELETE",
  body: unknown | undefined,
  fetchFn: typeof fetch = fetch,
): Promise<T> {
  const init: RequestInit = { method };
  if (body !== undefined) {
    init.headers = { "content-type": "application/json" };
    init.body = JSON.stringify(body);
  }
  return requestJSON<T>(path, init, fetchFn);
}

export function putJSON<T>(
  path: string,
  body: unknown,
  fetchFn: typeof fetch = fetch,
): Promise<T> {
  return mutateJSON<T>(path, "PUT", body, fetchFn);
}

export function deleteJSON(
  path: string,
  fetchFn: typeof fetch = fetch,
): Promise<void> {
  return mutateJSON<void>(path, "DELETE", undefined, fetchFn);
}

export function listTasks(
  params: { status?: Task["status"]; due?: string; limit?: number } = {},
  fetchFn: typeof fetch = fetch,
): Promise<Task[]> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.due) query.set("due", params.due);
  if (params.limit != null) query.set("limit", String(params.limit));
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Task[]>(`/api/v1/tasks${suffix}`, fetchFn);
}

export function createTask(
  input: {
    title: string;
    notes?: string;
    due_date?: string;
    priority?: number;
  },
  fetchFn: typeof fetch = fetch,
): Promise<Task> {
  return mutateJSON<Task>("/api/v1/tasks", "POST", input, fetchFn);
}

export function updateTask(
  id: string,
  status: "completed" | "canceled",
  fetchFn: typeof fetch = fetch,
): Promise<Task> {
  return mutateJSON<Task>(
    `/api/v1/tasks/${encodeURIComponent(id)}`,
    "PATCH",
    { status },
    fetchFn,
  );
}

export function listExpenses(
  params: ListExpensesParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<Expense>> {
  const query = new URLSearchParams();
  if (params.from) query.set("from", params.from);
  if (params.to) query.set("to", params.to);
  if (params.currency) query.set("currency", params.currency);
  if (params.category) query.set("category", params.category);
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<Expense>>(`/api/v1/expenses${suffix}`, fetchFn);
}

export function getExpense(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<Expense> {
  return getJSON<Expense>(
    `/api/v1/expenses/${encodeURIComponent(id)}`,
    fetchFn,
  );
}

export function createExpense(
  input: CreateExpenseInput,
  fetchFn: typeof fetch = fetch,
): Promise<Expense> {
  return mutateJSON<Expense>("/api/v1/expenses", "POST", input, fetchFn);
}

export function updateExpense(
  id: string,
  input: ExpenseInput,
  fetchFn: typeof fetch = fetch,
): Promise<Expense> {
  return mutateJSON<Expense>(
    `/api/v1/expenses/${encodeURIComponent(id)}`,
    "PUT",
    input,
    fetchFn,
  );
}

export function deleteExpense(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<void> {
  return deleteJSON(`/api/v1/expenses/${encodeURIComponent(id)}`, fetchFn);
}

export function listMediaResolutionTasks(
  params: { status?: MediaResolutionTask["status"] } = {},
  fetchFn: typeof fetch = fetch,
): Promise<MediaResolutionTask[]> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<MediaResolutionTask[]>(
    `/api/v1/media/resolution-tasks${suffix}`,
    fetchFn,
  );
}

export function updateMediaResolutionTask(
  id: string,
  status: "resolved" | "dismissed",
  resolution?: Record<string, unknown>,
  fetchFn: typeof fetch = fetch,
): Promise<MediaResolutionTask> {
  return mutateJSON<MediaResolutionTask>(
    `/api/v1/media/resolution-tasks/${encodeURIComponent(id)}`,
    "PATCH",
    { status, resolution },
    fetchFn,
  );
}

export function listJobs(
  params: { status?: Job["status"]; kind?: string; limit?: number } = {},
  fetchFn: typeof fetch = fetch,
): Promise<Job[]> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.kind) query.set("kind", params.kind);
  if (params.limit != null) query.set("limit", String(params.limit));
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Job[]>(`/api/v1/jobs${suffix}`, fetchFn);
}

export function getJob(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<Job> {
  return getJSON<Job>(`/api/v1/jobs/${encodeURIComponent(id)}`, fetchFn);
}

export function triggerAction(
  action: "media-sync-anilist" | "media-sync-bangumi",
  fetchFn: typeof fetch = fetch,
): Promise<Job> {
  return mutateJSON<Job>(`/api/v1/actions/${action}`, "POST", {}, fetchFn);
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

// Walk the keyset pages for views that need a complete activity sweep (the
// server intentionally caps any single page at 100 rows).
export async function listAllActivities(
  params: ListActivitiesParams = {},
  maxItems = 500,
  fetchFn: typeof fetch = fetch,
): Promise<Activity[]> {
  if (maxItems <= 0) return [];

  const pageLimit = Math.min(Math.max(params.limit ?? 100, 1), 100);
  const activities: Activity[] = [];
  let cursor = params.cursor;

  while (activities.length < maxItems) {
    const page = await listActivities(
      { ...params, limit: pageLimit, cursor },
      fetchFn,
    );
    activities.push(...page.items);
    if (!page.has_more || !page.next_cursor) break;
    cursor = page.next_cursor;
  }

  return activities.slice(0, maxItems);
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

export function getSleep(
  id: string,
  fetchFn: typeof fetch = fetch,
): Promise<SleepSession> {
  return getJSON<SleepSession>(
    `/api/v1/sleep/${encodeURIComponent(id)}`,
    fetchFn,
  );
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

// Walk the keyset pages for history views. The daily endpoint caps any single
// request at 100 rows, so a larger historical window must follow cursors.
export async function listAllDaily(
  params: ListDailyParams = {},
  maxItems = 2000,
  fetchFn: typeof fetch = fetch,
): Promise<DailyRow[]> {
  if (maxItems <= 0) return [];

  const pageLimit = Math.min(Math.max(params.limit ?? 100, 1), 100);
  const rows: DailyRow[] = [];
  let cursor = params.cursor;

  while (rows.length < maxItems) {
    const page = await listDaily(
      { ...params, limit: pageLimit, cursor },
      fetchFn,
    );
    rows.push(...page.items);
    if (!page.has_more || !page.next_cursor) break;
    cursor = page.next_cursor;
  }

  return rows.slice(0, maxItems);
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

// --- Activity aggregates ---

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

export interface ActivitySummaryParams {
  // Scope every breakdown to one calendar year and/or one sport_type. Omit for
  // all-time / all-sport totals.
  year?: string | null;
  sport?: string | null;
}

export function getActivitySummary(
  params: ActivitySummaryParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Summary> {
  const query = new URLSearchParams();
  if (params.year) query.set("year", params.year);
  if (params.sport) query.set("sport", params.sport);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Summary>(`/api/v1/activities/summary${suffix}`, fetchFn);
}

// A single route line, rendered as a GeoJSON LineString. Coordinates are
// [lon, lat] pairs (GeoJSON order), already privacy-trimmed and decimated by
// the server.
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

export function getActivityRoutes(
  fetchFn: typeof fetch = fetch,
): Promise<RouteFeatureCollection> {
  return getJSON<RouteFeatureCollection>("/api/v1/activities/routes", fetchFn);
}
