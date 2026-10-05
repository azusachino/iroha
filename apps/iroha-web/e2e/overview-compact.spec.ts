import { test, expect } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";
for (const mode of ["light", "dark"] as const)
  for (const dense of [false, true]) {
    test(`Overview ${mode} ${dense ? "dense" : "sparse"} first pattern starts within 900px`, async ({
      page,
    }) => {
      const fixture = await installPilotFixtures(page, "overview");
      if (dense) {
        const totals = {
          activity_count: 5000,
          distance_m: 20500000,
          distance_known_count: 5000,
          distance_unknown_count: 0,
          duration_s: 61200000,
          elevation_gain_m: 500000,
        };
        await page.route("**/api/v1/activities/overview**", (route) =>
          route.fulfill({
            json: {
              summary: {
                totals,
                by_year: [{ key: "2026", ...totals }],
                by_month: [{ key: "2026-08", ...totals }],
                by_sport: [{ key: "run", ...totals }],
              },
              recent: Array.from({ length: 5 }, (_, index) => ({
                id: `dense-${index}`,
                title: `Synthetic long-labelled activity ${index}: ${"bounded detailed route evidence ".repeat(5)}`,
                sport_type: "run",
                started_at: `2026-08-${String(index + 1).padStart(2, "0")}T12:00:00Z`,
                distance_m: 10000,
                duration_s: 3600,
                moving_time_s: 3600,
                avg_hr: null,
                avg_pace_s_per_km: 360,
              })),
              active_days: Array.from({ length: 31 }, (_, index) => ({
                day: `2026-08-${String(index + 1).padStart(2, "0")}`,
                activity_count: 160,
              })),
              current_streak: 125,
            },
          }),
        );
      }
      await page.setViewportSize({ width: 320, height: 900 });
      await page.emulateMedia({ colorScheme: mode });
      await page.goto("/overview");
      await expect(
        page.getByRole("heading", { name: "Overview", exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Loading your long view…", { exact: true }),
      ).toHaveCount(0);
      const calendar = page.getByLabel("Activity calendar", { exact: true });
      await expect(calendar).toBeVisible();
      const y = await calendar.evaluate(
        (element) => element.getBoundingClientRect().top + window.scrollY,
      );
      expect(y).toBeLessThan(900);
      await expect(
        page.getByRole("link", { name: "Explore sleep", exact: false }),
      ).toBeVisible();
      await expect(
        page.getByRole("link", { name: "Explore library", exact: false }),
      ).toBeVisible();
      await expect(page.getByText(/Mixed windows/)).toBeVisible();
      const tiles = page.locator(".grapher-dashboard .stat-tile");
      await expect(tiles).toHaveCount(6);
      const distance = page.getByRole("region", {
        name: "Distance · all time",
        exact: true,
      });
      await expect(distance).toContainText(dense ? "20,500.00 km" : "20.50 km");
      await expect(distance).toContainText("All time");
      const count = page.getByRole("region", {
        name: "Activity records",
        exact: true,
      });
      await expect(count).toContainText(dense ? "5,000" : "2");
      await expect(
        page.getByRole("region", { name: "Main sleep · recent", exact: true }),
      ).toContainText("Recent records (limit 30)");
      await expect(
        page
          .getByRole("region", { name: "Main sleep · recent", exact: true })
          .locator(".stat-value"),
      ).toHaveText("—");
      await expect(
        page
          .getByRole("region", { name: "Library items", exact: true })
          .locator(".stat-value"),
      ).toHaveText("0");
      const fontEvidence = await tiles.evaluateAll((elements) =>
        elements.map((element) => {
          const fonts = () =>
            Array.from(
              element.querySelectorAll(
                ".stat-label,.stat-value,.stat-context,.stat-sub,.stat-actions a",
              ),
            ).map((node) => getComputedStyle(node).fontSize);
          const compact = fonts();
          element.classList.remove("compact");
          const regular = fonts();
          element.classList.add("compact");
          return { compact, regular };
        }),
      );
      for (const tile of fontEvidence)
        expect(tile.compact).toEqual(tile.regular);
      for (const label of ["Explore sleep", "Explore library"]) {
        const target = await page
          .getByRole("link", { name: label, exact: false })
          .boundingBox();
        expect(target).not.toBeNull();
        expect(target!.height).toBeGreaterThanOrEqual(24);
      }
      expect(fixture.unknown).toEqual([]);
    });
  }
