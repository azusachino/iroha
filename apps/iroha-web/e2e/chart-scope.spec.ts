import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

for (const mode of ["light", "dark"] as const) {
  test(`public ${mode} cumulative chart removes years outside the selected comparison`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode });
    const fixture = await installPilotFixtures(page, "public");
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.stack ?? error.message));
    await page.goto(PUBLIC_BASE_URL);

    for (const year of ["2026", "2025", "2026", "2025"]) {
      await page.getByRole("button", { name: year, exact: true }).click();
      const label = `Cumulative distance for ${year}`;
      await expect(
        page.getByRole("img", { name: label, exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("cell", { name: new RegExp(`${year}-08-14`) }),
      ).toBeVisible();
      // Read the rendered chart's public API, not just its updated ARIA label
      // or HTML table: those already changed while the plot stayed stale.
      await expect
        .poll(async () => {
          const option = await page.evaluate(inspectPilotCharts);
          expect(option.verified).toBe(true);
          return option.verified
            ? option.charts
                .find((chart) => chart.label === label)
                ?.series.map((series) => ({
                  year: series.name,
                  august: series.data?.[7],
                }))
            : [];
        })
        .toEqual(
          year === "2025"
            ? [{ year: "2025", august: 8000 }]
            : [
                { year: "2025", august: 8000 },
                { year: "2026", august: 12500 },
              ],
        );
    }
    expect(errors).toEqual([]);
    expect(fixture.unknown).toEqual([]);
    expect(
      fixture.requests.filter((request) => request.startsWith("/api/")),
    ).toEqual([]);
  });
}
