import type { Page } from "@playwright/test";
import type { DailyRow, DailyAggregateBucket } from "../src/lib/api";
import { fakeSession } from "./session";

const rows: DailyRow[] = [
  ["2026-09-14", 9000],
  ["2026-08-14", 8000],
  ["2025-03-11", 2000],
].map(([day, steps]) => ({
  id: `synthetic-${day}`,
  day: String(day),
  steps: Number(steps),
  ring: {
    move_kcal: 100,
    move_goal_kcal: 500,
    exercise_min: 10,
    exercise_goal_min: 30,
    stand_hours: 4,
    stand_goal_hours: 12,
  },
  distance_km: 1.5,
  source: "manual",
  first_raw_file_id: "synthetic-raw",
  created_at: "2026-09-14T12:00:00Z",
  updated_at: "2026-09-14T12:00:00Z",
}));

export async function installPatternsFixtures(page: Page) {
  const fixture = {
    failures: new Set<string>(),
    holds: new Map<string, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    empty: false,
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    const gran = url.searchParams.get("granularity");
    const scope = url.searchParams.get("date") ?? "lifetime";
    const dependency = path.endsWith("/bounds")
      ? "bounds"
      : gran === "month"
        ? "monthly"
        : gran === "year"
          ? "yearly"
          : path === "/api/v1/daily"
            ? url.searchParams.get("limit") === "1"
              ? "latest"
              : "days"
            : "unknown";
    fixture.requests.push(`${dependency}:${scope}`);
    await (fixture.holds.get(`${dependency}:${scope}`) ??
      fixture.holds.get(dependency));
    if (
      fixture.failures.has(dependency) ||
      fixture.failures.has(`${dependency}:${scope}`)
    )
      return route.fulfill({
        status: 503,
        json: { error: `Synthetic ${dependency} failure` },
      });
    const records = fixture.empty ? [] : rows;
    if (dependency === "bounds")
      return route.fulfill({
        json: records.length ? { min: "2025-03-11", max: "2026-09-14" } : {},
      });
    if (dependency === "latest" || dependency === "days")
      return route.fulfill({
        json: {
          items: records
            .filter((row) => scope === "lifetime" || row.day.startsWith(scope))
            .slice(0, Number(url.searchParams.get("limit") ?? 100)),
          has_more: false,
          next_cursor: null,
        },
      });
    if (gran) {
      const selected = records.filter(
        (row) => scope === "lifetime" || row.day.startsWith(scope),
      );
      const periods = [
        ...new Set(
          selected.map((row) => row.day.slice(0, gran === "year" ? 4 : 7)),
        ),
      ].sort();
      const buckets: DailyAggregateBucket[] = periods.map((period) => {
        const matching = selected.filter((row) => row.day.startsWith(period));
        return {
          period: `${period}${gran === "year" ? "-01-01" : "-01"}T00:00:00Z`,
          days: matching.length,
          move_kcal_avg: 100,
          exercise_min_avg: 10,
          stand_hours_avg: 4,
          move_closed_pct: 0,
          metrics: [
            {
              metric: "steps",
              value:
                matching.reduce((sum, row) => sum + row.steps!, 0) /
                matching.length,
              unit: "count",
              observed_days: matching.length,
            },
            {
              metric: "distance_km",
              value: 1.5,
              unit: "km",
              observed_days: matching.length,
            },
          ],
        };
      });
      return route.fulfill({ json: { granularity: gran, buckets } });
    }
    fixture.unknown.push(path);
    return route.fulfill({
      status: 404,
      json: { error: "Unmodelled endpoint" },
    });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
    display_name: "Synthetic owner",
  });
  return fixture;
}
