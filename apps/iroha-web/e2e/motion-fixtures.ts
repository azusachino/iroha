import type { Page } from "@playwright/test";
import type { Activity, ActivitySummary } from "../src/lib/api";
import type {
  ActivitySummaryBucket,
  ActivitySummaryTotals,
} from "@iroha/shared/domain/activity";
import type { MetricSeriesResponse } from "@iroha/shared/components/metric-series";
import { fakeSession } from "./session";

type Dependency = "summary" | "records" | "series" | "bounds";
const MONTHS = ["2025-08", "2025-09", "2026-08", "2026-09"];

function totals(records: Activity[]): ActivitySummaryTotals {
  const known = records.filter((record) => record.distance_m != null);
  return {
    activity_count: records.length,
    distance_m: known.reduce(
      (sum, record) => sum + (record.distance_m ?? 0),
      0,
    ),
    distance_known_count: known.length,
    distance_unknown_count: records.length - known.length,
    duration_s: records.reduce(
      (sum, record) => sum + (record.duration_s ?? 0),
      0,
    ),
    elevation_gain_m: 0,
  };
}

function buckets(
  records: Activity[],
  key: (record: Activity) => string,
): ActivitySummaryBucket[] {
  const groups = new Map<string, Activity[]>();
  for (const record of records) {
    const value = key(record);
    groups.set(value, [...(groups.get(value) ?? []), record]);
  }
  return [...groups]
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([key, rows]) => ({ key, ...totals(rows) }));
}

export async function installMotionFixtures(page: Page) {
  const failures = new Set<Dependency>();
  const requests: string[] = [];
  const unknown: string[] = [];
  let hold: Promise<void> | null = null;
  let pageHold: Promise<void> | null = null;
  const fixture = {
    missingDistance: false,
    paginated: false,
    failures,
    requests,
    unknown,
    set hold(value: Promise<void> | null) {
      hold = value;
    },
    set pageHold(value: Promise<void> | null) {
      pageHold = value;
    },
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url.pathname + url.search);
    if (url.pathname === "/api/v1/activities/bounds")
      return failures.has("bounds")
        ? route.fulfill({
            status: 503,
            json: { error: "Synthetic range failure" },
          })
        : route.fulfill({ json: { min: "2025-08-14", max: "2026-09-14" } });
    const dependency: Dependency | null =
      url.pathname === "/api/v1/activities/summary"
        ? "summary"
        : url.pathname === "/api/v1/activities"
          ? "records"
          : /^\/api\/v1\/metrics\/movement\.(distance_m|duration_s)\/series$/.test(
                url.pathname,
              )
            ? "series"
            : null;
    if (!dependency) {
      unknown.push(url.pathname);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown synthetic endpoint" },
      });
    }
    if (pageHold && url.searchParams.has("cursor")) await pageHold;
    else if (hold) await hold;
    if (failures.has(dependency))
      return route.fulfill({
        status: 503,
        json: { error: "Synthetic read failure" },
      });

    const records: Activity[] = MONTHS.flatMap((month) => {
      const record: Activity = {
        id: `synthetic-${month}`,
        title: `Synthetic run ${month}`,
        sport_type: "run",
        started_at: `${month}-14T12:00:00Z`,
        timezone: "UTC",
        distance_m: fixture.missingDistance
          ? undefined
          : month.endsWith("-09")
            ? 8000
            : 12500,
        duration_s: 3720,
        source_kind: "synthetic",
        first_raw_file_id: "synthetic-raw",
        created_at: "2026-10-01T12:00:00Z",
        updated_at: "2026-10-01T12:00:00Z",
      };
      return fixture.paginated
        ? [
            record,
            {
              ...record,
              id: `${record.id}-next`,
              title: `Synthetic next run ${month}`,
            },
          ]
        : [record];
    });
    const dimensions: Record<string, string> = Object.fromEntries(
      url.searchParams.getAll("dimension").map((value) => value.split(":", 2)),
    );
    const sport =
      url.searchParams.get("sport_type") ||
      url.searchParams.get("sport") ||
      dimensions.sport;
    const matchesSport = (record: Activity) =>
      !sport ||
      record.sport_type ===
        (sport.toLowerCase() === "running" ? "run" : sport.toLowerCase());
    const date = url.searchParams.get("date") || url.searchParams.get("year");
    if (dependency === "records") {
      const selected = records.filter(
        (record) =>
          matchesSport(record) && (!date || record.started_at.startsWith(date)),
      );
      const next = url.searchParams.has("cursor");
      const items = fixture.paginated
        ? selected.slice(next ? 1 : 0, next ? undefined : 1)
        : selected;
      const hasMore =
        fixture.paginated && !next && selected.length > items.length;
      return route.fulfill({
        json: {
          items,
          has_more: hasMore,
          next_cursor: hasMore ? "synthetic-cursor" : null,
        },
      });
    }
    if (dependency === "summary") {
      // The real HTTP handler resolves month requests to their YEAR. ByMonth
      // is scoped by year/sport, ByYear ignores year, BySport ignores sport.
      const year = date?.slice(0, 4);
      const inYear = (record: Activity) =>
        !year || record.started_at.startsWith(year);
      const selected = records.filter(
        (record) => matchesSport(record) && inYear(record),
      );
      const summary: ActivitySummary = {
        totals: totals(selected),
        by_month: buckets(selected, (record) => record.started_at.slice(0, 7)),
        by_year: buckets(records.filter(matchesSport), (record) =>
          record.started_at.slice(0, 4),
        ),
        by_sport: buckets(
          records.filter(inYear),
          (record) => record.sport_type,
        ),
      };
      return route.fulfill({ json: summary });
    }
    const metric = decodeURIComponent(url.pathname.split("/")[4]);
    const from = url.searchParams.get("from")!;
    const to = url.searchParams.get("to")!;
    const grain = url.searchParams.get("grain") === "month" ? "month" : "day";
    const first = new Date(from);
    const last = new Date(to);
    const expectedPeriods =
      grain === "month"
        ? (last.getUTCFullYear() - first.getUTCFullYear()) * 12 +
          last.getUTCMonth() -
          first.getUTCMonth()
        : Math.round((last.getTime() - first.getTime()) / 86400000);
    const selected = records.filter(
      (record) =>
        matchesSport(record) &&
        record.started_at.slice(0, 10) >= from &&
        record.started_at.slice(0, 10) < to,
    );
    const grouped = new Map<string, Activity[]>();
    for (const record of selected) {
      const period = record.started_at.slice(0, grain === "month" ? 7 : 10);
      grouped.set(period, [...(grouped.get(period) ?? []), record]);
    }
    const points = [...grouped]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([period, rows]) => {
        const values = rows.filter((record) =>
          metric.endsWith("distance_m")
            ? record.distance_m != null
            : record.duration_s != null,
        );
        return {
          period,
          value: values.length
            ? values.reduce(
                (sum, record) =>
                  sum +
                  (metric.endsWith("distance_m")
                    ? (record.distance_m ?? 0)
                    : (record.duration_s ?? 0)),
                0,
              )
            : null,
          observed_days: new Set(
            values.map((record) => record.started_at.slice(0, 10)),
          ).size,
        };
      });
    const observed = points.filter((point) => point.value != null).length;
    const series: MetricSeriesResponse = {
      schema: "metric-series.v1",
      metric_id: metric,
      label: metric,
      unit: metric.endsWith("distance_m") ? "m" : "s",
      value_type: "number",
      period: {
        grain,
        from,
        to,
        timezone: url.searchParams.get("timezone") ?? "UTC",
      },
      series: [
        {
          dimensions,
          points,
          coverage: {
            expected_periods: expectedPeriods,
            observed_periods: observed,
            observation_state: observed ? "partial" : "empty",
            collection_completeness: "unknown",
          },
          source: {
            kind: "canonical",
            method: "synthetic.v1",
            source_kinds: ["synthetic"],
          },
        },
      ],
    };
    return route.fulfill({ json: series });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
  });
  return fixture;
}
