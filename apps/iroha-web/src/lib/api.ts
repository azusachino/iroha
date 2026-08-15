import { API_BASE, IROHA_TIMEZONE } from "./config";
import type { DateBounds } from "@iroha/shared/scope";
import type {
  Activity,
  ActivityOverview,
  ActivitySummary,
  Lap,
  ListActivitiesParams,
  RoutePoint,
  RouteFeatureCollection,
  SamplingPoint,
} from "@iroha/shared/activity";
import type { MetricSeriesResponse } from "@iroha/shared/metric-series";
import type {
  DailyAggregates,
  DailyDates,
  DailyRow,
  ListDailyParams,
} from "@iroha/shared/daily";
import type {
  MediaAggregates,
  MediaChange,
  MediaChangePage,
  MediaDaySection,
  MediaDetail,
  MediaEvent,
  MediaEventInput,
  MediaEventPage,
  MediaHomeEvent,
  MediaRow,
} from "@iroha/shared/media";
import type {
  CreateExpenseInput,
  Expense,
  ExpenseCategory,
  ExpenseCurrency,
  ExpenseInput,
  ExpenseItem,
  ExpenseSource,
  ListExpensesParams,
} from "@iroha/shared/expense";
import type {
  DailyHealthReportData,
  ExpensesReportData,
  MediaReportData,
  MovementReportData,
  MonthlyReport,
  MonthlyReportSeries,
  MonthlyReportSeriesPoint,
  ReportSection,
  SleepReportData,
} from "@iroha/shared/report";
import type {
  ListSleepParams,
  SleepAggregateBucket,
  SleepAggregates,
  SleepOverview,
  SleepSegment,
  SleepSession,
} from "@iroha/shared/sleep";

export type {
  Activity,
  ActivityActiveDay,
  ActivityDisplaySummary,
  ActivityOverview,
  ActivitySummary,
  ActivitySummaryBucket as SummaryBucket,
  ActivitySummaryTotals as SummaryTotals,
  ActivitySummary as Summary,
  Lap,
  ListActivitiesParams,
  RouteFeature,
  RouteFeatureCollection,
  RouteFeatureProperties,
  RoutePoint,
  SamplingPoint,
} from "@iroha/shared/activity";
export type { MetricSeriesResponse } from "@iroha/shared/metric-series";
export type {
  DailyAggregateBucket,
  DailyAggregates,
  DailyDates,
  DailyMetricAggregate,
  DailyRing,
  DailyRow,
  ListDailyParams,
} from "@iroha/shared/daily";
export type {
  MediaAggregates,
  MediaChange,
  MediaChangePage,
  MediaDaySection,
  MediaCompletionBucket,
  MediaDetail,
  MediaEvent,
  MediaEventInput,
  MediaEventPage,
  MediaHomeEvent,
  MediaPage,
  MediaRow,
  MediaScoreBucket,
  MediaTypeBucket,
} from "@iroha/shared/media";
export type {
  CreateExpenseInput,
  Expense,
  ExpenseCategory,
  ExpenseCurrency,
  ExpenseInput,
  ExpenseItem,
  ExpenseSource,
  ListExpensesParams,
} from "@iroha/shared/expense";
export type {
  DailyHealthReportData,
  ExpensesReportData,
  MediaReportData,
  MovementReportData,
  MonthlyReport,
  MonthlyReportSeries,
  MonthlyReportSeriesPoint,
  ReportSection,
  SleepReportData,
} from "@iroha/shared/report";
export type {
  ListSleepParams,
  SleepAggregateBucket,
  SleepAggregates,
  SleepOverview,
  SleepSegment,
  SleepSession,
} from "@iroha/shared/sleep";

// Types mirror the iroha-server read API JSON contract (snake_case).
// Optional fields use `?` because the server omits them when absent.

export interface ListMediaParams {
  status?: string;
  media_type?: string;
  family?: string;
  completed_year?: number;
  limit?: number;
  cursor?: string;
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

export interface ListMediaEventsParams {
  date?: string;
  from?: string;
  to?: string;
  limit?: number;
  cursor?: string;
}

export interface ListMediaChangesParams extends ListMediaEventsParams {}

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

export interface MetricDimension {
  id: string;
  label: string;
  values: string[];
  required: boolean;
  expand_by_default: boolean;
}

export interface MetricDefinition {
  id: string;
  domain: string;
  label: string;
  description: string;
  kind: "canonical" | "derived";
  value_type: string;
  unit: string;
  short_unit: string;
  supported_grains: ("day" | "month" | "year")[];
  dimensions: MetricDimension[];
  reducer: string;
  rollup: "sum" | "average" | "count";
  aggregation_version: string;
  coverage_kind: string;
  semantic_color_token: string;
  preferred_view: string;
}

export interface MetricCatalogResponse {
  schema: "metric-catalog.v1";
  metrics: MetricDefinition[];
}

export interface MetricDefinitionResponse {
  schema: "metric-catalog.v1";
  metric: MetricDefinition;
}

export function getMetricCatalog(
  fetchFn: typeof fetch = fetch,
): Promise<MetricCatalogResponse> {
  return getJSON<MetricCatalogResponse>("/api/v1/metrics", fetchFn);
}

export function getMetricDefinition(
  metricId: string,
  fetchFn: typeof fetch = fetch,
): Promise<MetricDefinitionResponse> {
  return getJSON<MetricDefinitionResponse>(
    `/api/v1/metrics/${encodeURIComponent(metricId)}`,
    fetchFn,
  );
}

export interface MetricSeriesParams {
  from: string;
  to: string;
  grain: "day" | "month" | "year";
  timezone?: string;
  dimensions?: string[];
}

function setTimezone(query: URLSearchParams, timezone = IROHA_TIMEZONE): void {
  query.set("timezone", timezone);
}

export function getMetricSeries(
  metricId: string,
  params: MetricSeriesParams,
  fetchFn: typeof fetch = fetch,
): Promise<MetricSeriesResponse> {
  const query = new URLSearchParams({
    from: params.from,
    to: params.to,
    grain: params.grain,
  });
  setTimezone(query, params.timezone ?? IROHA_TIMEZONE);
  for (const dimension of params.dimensions ?? []) {
    query.append("dimension", dimension);
  }
  return getJSON<MetricSeriesResponse>(
    `/api/v1/metrics/${encodeURIComponent(metricId)}/series?${query.toString()}`,
    fetchFn,
  );
}

export function getBriefing(
  date: string,
  fetchFn: typeof fetch = fetch,
): Promise<BriefingResponse> {
  const query = new URLSearchParams({ date });
  setTimezone(query);
  return getJSON<BriefingResponse>(
    `/api/v1/briefing?${query.toString()}`,
    fetchFn,
  );
}

export function getMonthlyReport(
  month: string,
  fetchFn: typeof fetch = fetch,
): Promise<MonthlyReport> {
  const query = new URLSearchParams({ date: month });
  setTimezone(query);
  return getJSON<MonthlyReport>(
    `/api/v1/reports/monthly?${query.toString()}`,
    fetchFn,
  );
}

export function getMonthlyReportSeries(
  endMonth: string,
  months = 12,
  fetchFn: typeof fetch = fetch,
): Promise<MonthlyReportSeries> {
  const query = new URLSearchParams({ date: endMonth, months: String(months) });
  setTimezone(query);
  return getJSON<MonthlyReportSeries>(
    `/api/v1/reports/monthly-series?${query.toString()}`,
    fetchFn,
  );
}

// One day of the daily-activity + body-vitals module. The API keeps activity
// rings as one nullable object so the canonical wire shape is explicit: a
// missing ring is different from a ring whose values happen to be zero.
export function listDailyAggregates(
  granularity: "month" | "year",
  params: { date?: string; from?: string; to?: string } = {},
  fetchFn: typeof fetch = fetch,
): Promise<DailyAggregates> {
  const query = new URLSearchParams({ granularity });
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  setTimezone(query);
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
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  if (params.date || params.from || params.to) setTimezone(query);
  if (params.currency) query.set("currency", params.currency);
  if (params.category) query.set("category", params.category);
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<Expense>>(`/api/v1/expenses${suffix}`, fetchFn);
}

export async function listAllExpenses(
  params: ListExpensesParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Expense[]> {
  const expenses: Expense[] = [];
  let cursor = params.cursor;
  do {
    const page = await listExpenses({ ...params, limit: 100, cursor }, fetchFn);
    expenses.push(...page.items);
    cursor = page.has_more ? (page.next_cursor ?? undefined) : undefined;
  } while (cursor);
  return expenses;
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

export function getExpenseBounds(
  fetchFn: typeof fetch = fetch,
): Promise<DateBounds> {
  const query = new URLSearchParams();
  setTimezone(query);
  return getJSON<DateBounds>(
    `/api/v1/expenses/bounds?${query.toString()}`,
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
  if (params.date) query.set("date", params.date);
  if (params.sport_type) query.set("sport_type", params.sport_type);
  if (!params.date && params.started_from)
    query.set("started_from", params.started_from);
  if (!params.date && params.started_to)
    query.set("started_to", params.started_to);
  if (params.min_distance_m != null)
    query.set("min_distance_m", String(params.min_distance_m));
  if (params.max_distance_m != null)
    query.set("max_distance_m", String(params.max_distance_m));
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.date) setTimezone(query);
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
  params: Pick<
    ListMediaParams,
    "status" | "media_type" | "family" | "completed_year"
  > = {},
  fetchFn: typeof fetch = fetch,
): Promise<MediaAggregates> {
  const query = new URLSearchParams();
  if (params.status) query.set("status", params.status);
  if (params.media_type) query.set("media_type", params.media_type);
  if (params.family) query.set("family", params.family);
  if (params.completed_year != null)
    query.set("completed_year", String(params.completed_year));
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<MediaAggregates>(`/api/v1/media/aggregates${suffix}`, fetchFn);
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
): Promise<MediaEventPage> {
  const query = new URLSearchParams();
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.date || params.from || params.to) setTimezone(query);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<MediaEventPage>(`/api/v1/media/events${suffix}`, fetchFn);
}

export function createMediaEvent(
  input: MediaEventInput,
  fetchFn: typeof fetch = fetch,
): Promise<MediaEvent> {
  return mutateJSON<MediaEvent>("/api/v1/media/events", "POST", input, fetchFn);
}

export function listMediaChanges(
  params: ListMediaChangesParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<MediaChangePage> {
  const query = new URLSearchParams();
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.date || params.from || params.to) setTimezone(query);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<MediaChangePage>(`/api/v1/media/changes${suffix}`, fetchFn);
}

export function listSleep(
  params: ListSleepParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<SleepSession>> {
  const query = new URLSearchParams();
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.date || params.from || params.to) setTimezone(query);
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
  granularity: "month" | "year" | "lifetime",
  params: { date?: string; from?: string; to?: string } = {},
  fetchFn: typeof fetch = fetch,
): Promise<SleepAggregates> {
  const query = new URLSearchParams({ granularity });
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  setTimezone(query);
  return getJSON<SleepAggregates>(
    `/api/v1/sleep/aggregates?${query.toString()}`,
    fetchFn,
  );
}

export function getSleepOverview(
  params: { recent?: number } = {},
  fetchFn: typeof fetch = fetch,
): Promise<SleepOverview> {
  const query = new URLSearchParams();
  if (params.recent != null) query.set("recent", String(params.recent));
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<SleepOverview>(`/api/v1/sleep/overview${suffix}`, fetchFn);
}

export function getSleepBounds(
  fetchFn: typeof fetch = fetch,
): Promise<DateBounds> {
  const query = new URLSearchParams();
  setTimezone(query);
  return getJSON<DateBounds>(
    `/api/v1/sleep/bounds?${query.toString()}`,
    fetchFn,
  );
}

export function listDaily(
  params: ListDailyParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<Page<DailyRow>> {
  const query = new URLSearchParams();
  if (params.date) query.set("date", params.date);
  else {
    if (params.from) query.set("from", params.from);
    if (params.to) query.set("to", params.to);
  }
  if (params.limit != null) query.set("limit", String(params.limit));
  if (params.cursor) query.set("cursor", params.cursor);
  if (params.date || params.from || params.to) setTimezone(query);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<Page<DailyRow>>(`/api/v1/daily${suffix}`, fetchFn);
}

export function getDailyDates(
  fetchFn: typeof fetch = fetch,
): Promise<DailyDates> {
  const query = new URLSearchParams();
  setTimezone(query);
  return getJSON<DailyDates>(
    `/api/v1/daily/dates?${query.toString()}`,
    fetchFn,
  );
}

export function getDailyBounds(
  fetchFn: typeof fetch = fetch,
): Promise<DateBounds> {
  const query = new URLSearchParams();
  setTimezone(query);
  return getJSON<DateBounds>(
    `/api/v1/daily/bounds?${query.toString()}`,
    fetchFn,
  );
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

export interface ActivitySummaryParams {
  // Scope every breakdown to one calendar year/month and/or one sport_type.
  // `date` is canonical; `year` remains a compatibility input for callers
  // that have not migrated yet. Omit both for all-time / all-sport totals.
  date?: string | null;
  year?: string | null;
  sport?: string | null;
  timezone?: string | null;
}

export function getActivitySummary(
  params: ActivitySummaryParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<ActivitySummary> {
  const query = new URLSearchParams();
  if (params.date) query.set("date", params.date);
  else if (params.year) query.set("year", params.year);
  if (params.sport) query.set("sport", params.sport);
  if (params.timezone) query.set("timezone", params.timezone);
  else if (params.date || params.year) setTimezone(query);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<ActivitySummary>(
    `/api/v1/activities/summary${suffix}`,
    fetchFn,
  );
}

export function getActivityBounds(
  fetchFn: typeof fetch = fetch,
): Promise<DateBounds> {
  const query = new URLSearchParams();
  setTimezone(query);
  return getJSON<DateBounds>(
    `/api/v1/activities/bounds?${query.toString()}`,
    fetchFn,
  );
}

export interface ActivityOverviewParams {
  recent?: number;
  timezone?: string | null;
}

export function getActivityOverview(
  params: ActivityOverviewParams = {},
  fetchFn: typeof fetch = fetch,
): Promise<ActivityOverview> {
  const query = new URLSearchParams();
  if (params.recent != null) query.set("recent", String(params.recent));
  setTimezone(query);
  if (params.timezone) query.set("timezone", params.timezone);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return getJSON<ActivityOverview>(
    `/api/v1/activities/overview${suffix}`,
    fetchFn,
  );
}

export function getActivityRoutes(
  fetchFn: typeof fetch = fetch,
): Promise<RouteFeatureCollection> {
  return getJSON<RouteFeatureCollection>("/api/v1/activities/routes", fetchFn);
}
