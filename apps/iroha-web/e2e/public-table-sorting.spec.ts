import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

const activities = Array.from({ length: 120 }, (_, index) => ({
  id: `sort-${index}`,
  sport_type: index % 2 ? "walk" : "run",
  started_at: new Date(Date.UTC(2026, 7, 1, 0, index)).toISOString(),
  timezone: "UTC",
  title: `Sort record ${index}`,
  distance_m: index + 1,
  duration_s: 600,
  avg_pace_s_per_km: 240 + index,
}));

for (const mode of ["light", "dark"] as const) {
  test(`public ${mode} retains sort cycles, shift state, filters and progressive rows`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "public");
    await page.route("**/public/v1/activities", (route) =>
      route.fulfill({ json: activities }),
    );
    await page.goto(PUBLIC_BASE_URL);
    const records = page.getByRole("region", {
      name: "Activity records",
      exact: true,
    });
    const links = records.locator("tbody .activity-link");
    const firstId = () => links.first().getAttribute("href");
    await expect(links).toHaveCount(50);
    await expect.poll(firstId).toContain("activity=sort-119");
    const date = records.getByRole("button", { name: /^Date/ });
    await date.focus();
    await page.keyboard.press("Enter");
    await expect(date).toHaveText("Date");
    await expect.poll(firstId).toContain("activity=sort-0");
    await page.keyboard.press("Enter");
    await expect(date).toHaveText("Date ▲");
    await page.keyboard.press("Enter");
    await expect(date).toHaveText("Date ▼");

    const distance = records.getByRole("button", {
      name: /^Distance \/ Duration/,
    });
    await distance.click();
    await expect(distance).toHaveText("Distance / Duration ▼");
    await expect.poll(firstId).toContain("activity=sort-119");
    await distance.click();
    await expect(distance).toHaveText("Distance / Duration ▲");
    await expect.poll(firstId).toContain("activity=sort-0");
    await distance.click();
    await expect(distance).toHaveText("Distance / Duration");
    await expect.poll(firstId).toContain("activity=sort-0");

    const sport = records.getByRole("button", { name: /^Activity/ });
    await sport.click();
    await expect(sport).toHaveText("Activity ▲");
    await distance.click({ modifiers: ["Shift"] });
    await expect(sport).toHaveText("Activity ▲");
    await expect(distance).toHaveText("Distance / Duration ▼");
    // Existing page semantics sort by the leading header only, even with Shift.
    await expect.poll(firstId).toContain("activity=sort-0");
    await distance.click({ modifiers: ["Shift"] });
    await expect(distance).toHaveText("Distance / Duration ▲");
    await distance.click({ modifiers: ["Shift"] });
    await expect(distance).toHaveText("Distance / Duration");
    await expect(sport).toHaveText("Activity ▲");
    await expect(links).toHaveCount(50);
    await records
      .getByRole("button", { name: "Load more", exact: true })
      .click();
    await expect.poll(() => links.count()).toBeGreaterThan(50);
    await page.locator(".sport-row").filter({ hasText: "Run" }).click();
    await expect(records.locator("tbody tr")).toHaveCount(50);
    await expect(
      records.locator("tbody").getByText("Walk", { exact: true }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
    expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
      [],
    );
  });
}
