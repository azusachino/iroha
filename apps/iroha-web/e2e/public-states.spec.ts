import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

for (const mode of ["light", "dark"] as const) {
  test(`public ${mode} monthly chart honors reduced motion`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "public");
    await page.goto(PUBLIC_BASE_URL);
    await expect(
      page.getByRole("img", { name: "Monthly distance for 2026", exact: true }),
    ).toBeVisible();
    const readMotion = () =>
      page.evaluate(inspectPilotCharts).then((snapshot) => {
        expect(snapshot.verified).toBe(true);
        const chart = snapshot.verified
          ? snapshot.charts.find(
              (chart) => chart.label === "Monthly distance for 2026",
            )
          : undefined;
        return {
          animation: chart?.animation,
          duration: chart?.animationDuration,
        };
      });
    await expect.poll(readMotion).toEqual({ animation: false, duration: 0 });
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await expect.poll(readMotion).toEqual({ animation: true, duration: 550 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await expect.poll(readMotion).toEqual({ animation: false, duration: 0 });
    for (const theme of [mode === "light" ? "dark" : "light", mode]) {
      await page.evaluate((theme) => {
        document.documentElement.dataset.theme = theme;
      }, theme);
      const paint = await page.evaluate(() =>
        getComputedStyle(document.documentElement)
          .getPropertyValue("--accent")
          .trim(),
      );
      await expect
        .poll(async () => {
          const snapshot = await page.evaluate(inspectPilotCharts);
          return snapshot.verified
            ? snapshot.charts.find(
                (chart) => chart.label === "Monthly distance for 2026",
              )?.series[0].color
            : undefined;
        })
        .toBe(paint);
    }
    expect(fixture.unknown).toEqual([]);
    expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
      [],
    );
  });

  test(`public ${mode} compact error shell retries sanitized data by keyboard`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const failed = await installPilotFixtures(page, "public", "error");
    await page.goto(PUBLIC_BASE_URL);
    await expect(page.getByRole("main")).toBeVisible();
    await expect(page.getByRole("alert")).toContainText(
      "Public activity data is temporarily unavailable.",
    );
    await page.keyboard.press("Tab");
    const retry = page.getByRole("button", { name: "Try again", exact: true });
    await expect(retry).toBeFocused();
    await page.unrouteAll({ behavior: "wait" });
    const recovered = await installPilotFixtures(page, "public");
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Activities", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(failed.unknown).toEqual([]);
    expect(recovered.unknown).toEqual([]);
    expect(
      [...failed.requests, ...recovered.requests].filter((path) =>
        path.startsWith("/api/"),
      ),
    ).toEqual([]);
  });
}
