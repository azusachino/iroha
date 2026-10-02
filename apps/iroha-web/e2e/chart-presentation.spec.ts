import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

for (const pilot of ["overview", "expenses", "metrics", "public"] as const) {
  test(`${pilot} plots retain data while live motion and mode preferences change`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, pilot);
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.stack ?? error.message));
    await page.goto(
      pilot === "public"
        ? PUBLIC_BASE_URL
        : `/${pilot}${pilot === "expenses" ? "?date=2026-08&currency=JPY" : ""}`,
    );
    await expect
      .poll(async () => {
        const snapshot = await page.evaluate(inspectPilotCharts);
        return snapshot.verified ? snapshot.charts.length : 0;
      })
      .toBeGreaterThan(0);
    const read = () => page.evaluate(inspectPilotCharts);
    const original = await read();
    expect(original.verified).toBe(true);
    const typeRoles = await page.evaluate((currentPilot) => {
      const checks =
        currentPilot === "public"
          ? [
              [".eyebrow", "--type-caption"],
              [".hero-summary", "--type-body"],
              [".section-heading h2", "--type-title"],
            ]
          : currentPilot === "overview"
            ? [
                [".kicker", "--type-caption"],
                [".chart-data", "--type-label"],
              ]
            : [[".chart-data", "--type-label"]];
      return checks.map(([selector, token]) => {
        const element = document.querySelector(selector);
        if (!element) throw new Error(`Missing typography probe: ${selector}`);
        const probe = document.createElement("span");
        probe.style.fontSize = `var(${token})`;
        document.body.append(probe);
        const expected = getComputedStyle(probe).fontSize;
        probe.remove();
        return getComputedStyle(element).fontSize === expected;
      });
    }, pilot);
    expect(typeRoles.every(Boolean)).toBe(true);
    const values = (
      snapshot: Awaited<ReturnType<typeof inspectPilotCharts>>,
    ) =>
      snapshot.verified
        ? snapshot.charts.map((chart) =>
            chart.series.map((series) =>
              series.data?.map((point) =>
                point != null && typeof point === "object" && "value" in point
                  ? point.value
                  : point,
              ),
            ),
          )
        : [];
    const data = values(original);
    const paint = original.verified
      ? original.charts.map((chart) =>
          chart.series.map((series) => series.color),
        )
      : [];
    for (const preference of ["no-preference", "reduce"] as const) {
      await page.emulateMedia({ reducedMotion: preference });
      await expect
        .poll(async () => {
          const snapshot = await read();
          return (
            snapshot.verified &&
            snapshot.charts.every(
              (chart) => chart.animation === (preference === "no-preference"),
            )
          );
        })
        .toBe(true);
    }
    await page.evaluate(() => {
      document.documentElement.dataset.theme = "dark";
    });
    await expect
      .poll(async () => {
        const snapshot = await read();
        return snapshot.verified
          ? snapshot.charts.map((chart) =>
              chart.series.map((series) => series.color),
            )
          : [];
      })
      .not.toEqual(paint);
    const changed = await read();
    expect(values(changed)).toEqual(data);
    await page.evaluate(() => {
      document.documentElement.style.fontSize = "20px";
    });
    await expect
      .poll(async () => {
        const snapshot = await read();
        return (
          snapshot.verified &&
          snapshot.charts.every((chart) =>
            [...chart.xAxis, ...chart.yAxis].every(
              (axis) => axis.fontSize === 15,
            ),
          )
        );
      })
      .toBe(true);
    expect(errors).toEqual([]);
    expect(fixture.unknown).toEqual([]);
  });
}
