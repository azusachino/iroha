import { expect, test, type Page } from "@playwright/test";
import type { Activity, ActivitySummary } from "@iroha/shared/domain/activity";
import type { PublicActivity } from "@iroha/shared/domain/public-activity";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { fakeSession } from "./session";

// Synthetic projection: no raw-file IDs, source payloads or private samples
// enter the anonymous API fixture.
const activity: PublicActivity = {
  id: "synthetic-run",
  title: "Synthetic run",
  sport_type: "run",
  started_at: "2026-08-14T12:00:00Z",
  timezone: "UTC",
  distance_m: 12500,
  duration_s: 3720,
  avg_pace_s_per_km: 59.6,
};
const routes = { type: "FeatureCollection", features: [] };
const summary: ActivitySummary = {
  totals: {
    activity_count: 1,
    distance_m: 12500,
    distance_known_count: 1,
    distance_unknown_count: 0,
    duration_s: 3720,
    elevation_gain_m: 0,
  },
  by_year: [],
  by_month: [],
  by_sport: [],
};

async function mockPrivate(page: Page, count = 1) {
  const recent: Activity = {
    ...activity,
    source_kind: "synthetic",
    first_raw_file_id: "synthetic-raw",
    created_at: activity.started_at,
    updated_at: activity.started_at,
  };
  const responses: Record<string, unknown> = {
    "/api/v1/activities/overview": {
      summary: {
        ...summary,
        totals: { ...summary.totals, activity_count: count },
      },
      recent: [recent],
      active_days: [],
      current_streak: 0,
    },
    "/api/v1/activities/routes": routes,
    "/api/v1/sleep/overview": {
      session_count: 1,
      main_sleep_count: 1,
      average_asleep_s: 2700,
      average_efficiency: 90,
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
  await page.route("**/api/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    expect(responses[path], path).toBeDefined();
    return route.fulfill({ json: responses[path] });
  });
  await fakeSession(page, { setup_required: false, authenticated: true });
}

for (const host of ["private", "public"] as const) {
  for (const mode of ["light", "dark"] as const) {
    test(`${host} ${mode} summaries use human time, exact tables and pace keep clock notation`, async ({
      page,
    }) => {
      await page.emulateMedia({ colorScheme: mode });
      const errors: string[] = [];
      page.on("pageerror", (error) =>
        errors.push(error.stack ?? error.message),
      );
      const privateRequests: string[] = [];
      if (host === "private") {
        await mockPrivate(page);
      } else {
        page.on("request", (request) => {
          if (new URL(request.url()).pathname.startsWith("/api/")) {
            privateRequests.push(request.url());
          }
        });
        const responses: Record<string, unknown> = {
          summary,
          activities: [activity],
          routes,
          meta: { generated_at: "2026-08-14T12:00:00Z", activity_count: 1 },
        };
        await page.route("**/public/v1/*", (route) => {
          const key = new URL(route.request().url()).pathname.split("/").pop()!;
          expect(responses[key], key).toBeDefined();
          return route.fulfill({ json: responses[key] });
        });
      }
      await page.goto(host === "private" ? "/overview" : PUBLIC_BASE_URL);
      const grid = page.locator(
        host === "private"
          ? ".grapher-dashboard .stat-grid"
          : ".dashboard .stat-grid",
      );
      await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
      await expect(grid).toContainText("12.50 km");
      await expect(grid).toContainText("1 h 02 min");
      if (host === "private") {
        await expect(
          page.getByRole("cell").filter({ hasText: "1:02:00" }),
        ).toBeVisible();
        await expect(grid).toContainText("45 min");
        await expect(
          grid
            .getByRole("region", { name: "Library items", exact: true })
            .getByText("0", { exact: true }),
        ).toHaveText("0");
      } else {
        await expect(page.getByText("1:00 /km", { exact: true })).toBeVisible();
      }
      expect(errors).toEqual([]);
      expect(privateRequests).toEqual([]);
    });
  }
}

test("Overview groups large counts without changing units", async ({
  page,
}) => {
  await mockPrivate(page, 12345);
  await page.goto("/overview");
  await expect(page.locator(".grapher-dashboard .stat-grid")).toContainText(
    "12,345",
  );
});
