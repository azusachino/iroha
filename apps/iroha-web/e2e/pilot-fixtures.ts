import type { Page } from "@playwright/test";
import type {
  Expense,
  MetricDefinition,
  MetricSeriesResponse,
} from "../src/lib/api";
import type { PublicActivity } from "@iroha/shared/domain/public-activity";
import { fakeSession } from "./session";

export type Pilot = "overview" | "expenses" | "metrics" | "public";
export type Scenario =
  "populated" | "empty" | "error" | "missing-distance" | "partial-distance";
const instant = "2026-08-14T12:00:00Z";
const activities: PublicActivity[] = [2025, 2026].map((year) => ({
  id: `synthetic-${year}`,
  title: `Synthetic run ${year}`,
  sport_type: "run",
  started_at: `${year}-08-14T12:00:00Z`,
  timezone: "UTC",
  distance_m: year === 2026 ? 12500 : 8000,
  duration_s: year === 2026 ? 3720 : 2400,
  avg_pace_s_per_km: 59.6,
}));
const buckets = activities.map((activity) => ({
  key: activity.started_at.slice(0, 7),
  activity_count: 1,
  distance_m: activity.distance_m ?? 0,
  distance_known_count: 1,
  distance_unknown_count: 0,
  duration_s: activity.duration_s,
  elevation_gain_m: 0,
}));
const summary = {
  totals: {
    activity_count: 2,
    distance_m: 20500,
    distance_known_count: 2,
    distance_unknown_count: 0,
    duration_s: 6120,
    elevation_gain_m: 0,
  },
  by_year: buckets.map((bucket) => ({
    ...bucket,
    key: bucket.key.slice(0, 4),
  })),
  by_month: buckets,
  by_sport: [
    {
      key: "run",
      activity_count: 2,
      distance_m: 20500,
      distance_known_count: 2,
      distance_unknown_count: 0,
      duration_s: 6120,
      elevation_gain_m: 0,
    },
  ],
};
const missingDistanceBuckets = buckets.map((bucket) =>
  bucket.key.startsWith("2026-")
    ? {
        ...bucket,
        distance_m: 0,
        distance_known_count: 0,
        distance_unknown_count: 1,
      }
    : bucket,
);
const missingDistanceSummary = {
  ...summary,
  totals: {
    ...summary.totals,
    distance_m: 8000,
    distance_known_count: 1,
    distance_unknown_count: 1,
  },
  by_year: missingDistanceBuckets.map((bucket) => ({
    ...bucket,
    key: bucket.key.slice(0, 4),
  })),
  by_month: missingDistanceBuckets,
  by_sport: [
    {
      ...summary.by_sport[0],
      distance_m: 8000,
      distance_known_count: 1,
      distance_unknown_count: 1,
    },
  ],
};
const emptySummary = {
  ...summary,
  totals: {
    ...summary.totals,
    activity_count: 0,
    distance_m: 0,
    distance_known_count: 0,
    duration_s: 0,
  },
  by_year: [],
  by_month: [],
  by_sport: [],
};
const routes = { type: "FeatureCollection", features: [] };
const expenses: Expense[] = [
  {
    id: "synthetic-jpy",
    occurred_on: "2026-08-14",
    account_key: "synthetic",
    kind: "expense",
    currency: "JPY",
    currency_exponent: 0,
    amount_minor: 12345,
    category: "food",
    merchant: "Synthetic cafe",
    note: "Synthetic fixture only",
    items: [],
    source: { kind: "manual", ref: "synthetic" },
    created_at: instant,
    updated_at: instant,
  },
  {
    id: "synthetic-usd",
    occurred_on: "2026-09-14",
    account_key: "synthetic",
    kind: "expense",
    currency: "USD",
    currency_exponent: 2,
    amount_minor: 123456,
    category: "shopping",
    merchant: "Synthetic shop",
    note: "Synthetic fixture only",
    items: [],
    source: { kind: "manual", ref: "synthetic" },
    created_at: instant,
    updated_at: instant,
  },
];
const definition: MetricDefinition = {
  id: "daily.steps",
  domain: "daily",
  label: "Steps",
  description: "Synthetic daily steps",
  kind: "canonical",
  value_type: "number",
  unit: "count",
  short_unit: "steps",
  supported_grains: ["day", "month", "year"],
  dimensions: [],
  reducer: "sum",
  rollup: "sum",
  aggregation_version: "synthetic.v1",
  coverage_kind: "observed",
  semantic_color_token: "--accent",
  preferred_view: "line",
};

function metricResponse(
  url: URL,
  rows: Expense[],
  scenario: Scenario,
): MetricSeriesResponse {
  const metricId = decodeURIComponent(url.pathname.split("/")[4]);
  const dimensions = url.searchParams.getAll("dimension");
  const categories = dimensions
    .filter((v) => v.startsWith("category:"))
    .map((v) => v.slice(9));
  const currencies = dimensions
    .filter((v) => v.startsWith("currency:"))
    .map((v) => v.slice(9));
  const from = url.searchParams.get("from")!;
  const to = url.searchParams.get("to")!;
  const grain = url.searchParams.get("grain") as "day" | "month" | "year";
  const selected = rows.filter(
    (row) => row.occurred_on >= from && row.occurred_on < to,
  );
  const groups: Record<string, string>[] = metricId.startsWith("expenses.")
    ? currencies.flatMap<Record<string, string>>((currency) =>
        categories.length
          ? categories.map((category) => ({ currency, category }))
          : [{ currency }],
      )
    : [{}];
  return {
    schema: "metric-series.v1",
    metric_id: metricId,
    label: metricId,
    unit: metricId.endsWith("amount_minor") ? "minor" : "count",
    value_type: "number",
    period: { grain, from, to, timezone: "Asia/Tokyo" },
    series: groups.map((group) => {
      const matching = selected.filter(
        (row) =>
          (!group.currency || row.currency === group.currency) &&
          (!("category" in group) || row.category === group.category),
      );
      const amount = matching.reduce((sum, row) => sum + row.amount_minor, 0);
      const points = metricId.startsWith("expenses.")
        ? [
            {
              period: from,
              observed_days: matching.length,
              ...(metricId.endsWith("amount_minor")
                ? { value_minor: amount }
                : { value: matching.length }),
            },
          ]
        : [
            {
              period: from.slice(0, 7),
              observed_days: scenario === "empty" ? 0 : 1,
              value: scenario === "empty" ? null : 12345,
            },
          ];
      return {
        dimensions: group,
        points,
        coverage: {
          expected_periods: 1,
          observed_periods: scenario === "empty" ? 0 : 1,
          observation_state: scenario === "empty" ? "empty" : "observed",
          collection_completeness: "unknown",
        },
        source: {
          kind: "canonical",
          method: "synthetic.v1",
          source_kinds: ["synthetic"],
        },
      };
    }),
  };
}

export async function installPilotFixtures(
  page: Page,
  pilot: Pilot,
  scenario: Scenario = "populated",
) {
  const requests: string[] = [];
  const unknown: string[] = [];
  const distanceMissing =
    scenario === "missing-distance" || scenario === "partial-distance";
  const records =
    scenario === "empty"
      ? []
      : distanceMissing
        ? activities.map((activity) => {
            if (activity.id !== "synthetic-2026") return activity;
            const { distance_m: _distance, ...withoutDistance } = activity;
            return withoutDistance;
          })
        : activities;
  if (scenario === "partial-distance")
    records.push({ ...activities[1], id: "synthetic-known-2026" });
  const partialBuckets = missingDistanceSummary.by_month.map((bucket) =>
    bucket.key.startsWith("2026-")
      ? {
          ...bucket,
          activity_count: 2,
          distance_m: 12500,
          distance_known_count: 1,
          duration_s: 7440,
        }
      : bucket,
  );
  const partialTotals = {
    ...summary.totals,
    activity_count: 3,
    distance_unknown_count: 1,
    duration_s: 9840,
  };
  const distanceSummary =
    scenario === "partial-distance"
      ? {
          totals: partialTotals,
          by_month: partialBuckets,
          by_year: partialBuckets.map((bucket) => ({
            ...bucket,
            key: bucket.key.slice(0, 4),
          })),
          by_sport: [{ key: "run", ...partialTotals }],
        }
      : missingDistanceSummary;
  const rows = scenario === "empty" ? [] : expenses;
  await page.route("**/public/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    requests.push(path);
    if (scenario === "error")
      return route.fulfill({
        status: 503,
        json: { error: "synthetic failure" },
      });
    const response: Record<string, unknown> = {
      summary:
        scenario === "empty"
          ? emptySummary
          : distanceMissing
            ? distanceSummary
            : summary,
      activities: records,
      routes,
      meta: { generated_at: instant, activity_count: records.length },
    };
    const key = path.split("/").pop()!;
    if (!(key in response)) unknown.push(path);
    return route.fulfill({
      status: key in response ? 200 : 404,
      json: response[key] ?? {},
    });
  });
  await page.route("**/api/v1/**", (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    requests.push(path + url.search);
    if (pilot === "public") {
      unknown.push(path);
      return route.fulfill({
        status: 403,
        json: { error: "private API forbidden in public fixture" },
      });
    }
    if (path.endsWith("/bounds"))
      return route.fulfill({ json: { min: "2025-01-01", max: "2026-09-30" } });
    if (scenario === "error")
      return route.fulfill({
        status: 503,
        json: { error: "synthetic failure" },
      });
    if (path.endsWith("/series"))
      return route.fulfill({ json: metricResponse(url, rows, scenario) });
    const response: Record<string, unknown> = {
      "/api/v1/metrics": { schema: "metric-catalog.v1", metrics: [definition] },
      "/api/v1/expenses": {
        items: rows.filter((row) => {
          const date = url.searchParams.get("date") ?? "";
          return (
            row.occurred_on.startsWith(date) &&
            (!url.searchParams.get("currency") ||
              row.currency === url.searchParams.get("currency"))
          );
        }),
        has_more: false,
        next_cursor: null,
      },
      "/api/v1/activities/overview": {
        summary: scenario === "empty" ? emptySummary : summary,
        recent: records.map((a) => ({
          ...a,
          source_kind: "synthetic",
          first_raw_file_id: "synthetic-raw",
          created_at: instant,
          updated_at: instant,
        })),
        active_days: records.map((activity) => ({
          day: activity.started_at.slice(0, 10),
          activity_count: 1,
        })),
        current_streak: 0,
      },
      "/api/v1/activities/routes": routes,
      "/api/v1/sleep/overview": {
        session_count: 0,
        main_sleep_count: 0,
        average_asleep_s: 0,
        average_efficiency: 0,
      },
      "/api/v1/media/aggregates": {
        totals: {
          item_count: 0,
          completed_count: 0,
          current_completed_count: 0,
          this_year_completed: 0,
          average_rating: 0,
        },
        completions_by_year: [],
        score_distribution: [],
        type_split: [],
      },
    };
    if (!(path in response)) unknown.push(path);
    return route.fulfill({
      status: path in response ? 200 : 404,
      json: response[path] ?? {},
    });
  });
  if (pilot !== "public")
    await fakeSession(page, {
      setup_required: false,
      authenticated: true,
      username: "synthetic-owner",
      display_name: "Synthetic owner",
    });
  return { requests, unknown };
}
