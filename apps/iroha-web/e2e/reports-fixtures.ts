import type { Page } from "@playwright/test";
import type {
  MonthlyReport,
  MonthlyReportSeries,
} from "@iroha/shared/domain/report";
import { fakeSession } from "./session";

function reportFor(
  month: string,
  timezone: string,
  empty: boolean,
): MonthlyReport {
  const [year, number] = month.split("-").map(Number);
  const distance = month === "2026-08" ? 8000 : 9000;
  return {
    schema: "monthly-report.v1",
    period: {
      kind: "month",
      month,
      from: `${month}-01`,
      to: new Date(Date.UTC(year, number, 1)).toISOString().slice(0, 10),
      timezone,
    },
    generated_at: "2026-10-04T12:00:00Z",
    status: {
      calendar_completeness: "complete",
      observation_state: empty ? "empty" : "observed",
      collection_completeness: "unknown",
    },
    sections: {
      movement: {
        schema: "monthly-report.movement.v1",
        state: empty ? "empty" : "available",
        data: empty
          ? null
          : {
              activity_count: 1,
              distance_m: distance,
              distance_activity_count: 1,
              duration_s: 3600,
              by_sport: [
                {
                  sport: "running",
                  activity_count: 1,
                  distance_m: distance,
                  distance_activity_count: 1,
                  duration_s: 3600,
                },
              ],
            },
      },
      sleep: { schema: "monthly-report.sleep.v1", state: "empty", data: null },
      daily_health: {
        schema: "monthly-report.daily-health.v1",
        state: "empty",
        data: null,
      },
      media: { schema: "monthly-report.media.v1", state: "empty", data: null },
      expenses: {
        schema: "monthly-report.expenses.v1",
        state: "empty",
        data: null,
      },
    },
  };
}
export async function installReportsFixtures(page: Page) {
  const fixture = {
    failures: new Set<string>(),
    holds: new Map<string, Promise<void>>(),
    requests: [] as string[],
    unknown: [] as string[],
    empty: false,
    boundsFailure: false,
    boundsHold: null as Promise<void> | null,
  };
  await page.route("**/api/v1/**", async (route) => {
    const url = new URL(route.request().url());
    fixture.requests.push(url.pathname + url.search);
    if (url.pathname === "/api/v1/daily/bounds") {
      if (fixture.boundsHold) await fixture.boundsHold;
      return route.fulfill(
        fixture.boundsFailure
          ? { status: 503, json: { error: "Synthetic bounds unavailable" } }
          : { json: { min: "2025-08-01", max: "2026-09-30" } },
      );
    }
    if (url.pathname !== "/api/v1/reports/monthly-series") {
      fixture.unknown.push(url.pathname + url.search);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown synthetic endpoint" },
      });
    }
    const month = url.searchParams.get("date") ?? "";
    const timezone = url.searchParams.get("timezone") ?? "Asia/Tokyo";
    const months = Number(url.searchParams.get("months"));
    if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(month) || months < 1 || months > 24)
      return route.fulfill({
        status: 400,
        json: { error: "Invalid synthetic month/series" },
      });
    if (fixture.holds.has(month)) await fixture.holds.get(month);
    if (fixture.failures.has(month))
      return route.fulfill({
        status: 503,
        json: {
          error: "Synthetic report unavailable",
          code: "internal_error",
          request_id: "synthetic-report-request",
        },
      });
    const current = reportFor(
      month,
      timezone,
      fixture.empty || !["2026-08", "2026-09"].includes(month),
    );
    const [year, number] = month.split("-").map(Number);
    const range = Array.from({ length: months }, (_, index) =>
      new Date(Date.UTC(year, number - months + index, 1))
        .toISOString()
        .slice(0, 7),
    );
    const observed = fixture.empty
      ? []
      : range.filter((value) => ["2026-08", "2026-09"].includes(value));
    const value: MonthlyReportSeries = {
      schema: "monthly-report-series.v2",
      end_month: month,
      requested_months: months,
      from_month: range[0],
      to_month: month,
      generated_at: current.generated_at,
      current_report: current,
      reports: observed.map((value) => ({
        month: value,
        completeness: "complete",
        collection_completeness: "unknown",
        movement: { distance_m: value === "2026-08" ? 8000 : 9000 },
        sleep: null,
        daily_health: null,
        media: null,
        expenses: null,
      })),
      empty_months: range.filter((value) => !observed.includes(value)),
    };
    return route.fulfill({ json: value });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
  });
  return fixture;
}
