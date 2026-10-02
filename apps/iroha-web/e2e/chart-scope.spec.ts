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

test("public unknown distances remain missing instead of becoming zero", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(
    page,
    "public",
    "missing-distance",
  );
  await page.goto(PUBLIC_BASE_URL);

  const distance = page.locator(".stat-grid .stat-tile").first();
  await expect(distance.getByText("—", { exact: true })).toBeVisible();
  await expect(distance).toContainText("Distance unavailable for 1 activity");
  await expect(
    page.getByRole("img", { name: "Monthly activity count for 2026" }),
  ).toBeVisible();
  await page.getByText("View cumulative distance data").click();
  await expect(
    page.getByRole("cell", { name: "Distance unknown" }).first(),
  ).toBeVisible();

  const snapshot = await page.evaluate(inspectPilotCharts);
  expect(snapshot.verified).toBe(true);
  if (snapshot.verified) {
    const chart = snapshot.charts.find(
      (candidate) => candidate.label === "Cumulative distance for 2026",
    );
    expect(
      chart?.series.find((series) => series.name === "2026")?.data?.[7],
    ).toBeNull();
  }
  expect(fixture.unknown).toEqual([]);
  expect(
    fixture.requests.filter((request) => request.startsWith("/api/")),
  ).toEqual([]);
});
