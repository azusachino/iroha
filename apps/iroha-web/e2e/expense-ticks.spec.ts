import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

const cases = [
  {
    currency: "JPY",
    date: "2026-08",
    exact: "¥12,345",
    merchant: "Synthetic cafe",
  },
  {
    currency: "USD",
    date: "2026-09",
    exact: "$1,234.56",
    merchant: "Synthetic shop",
  },
];
for (const mode of ["light", "dark"] as const) {
  for (const width of [320, 1280]) {
    for (const scenario of cases) {
      test(`Expenses ${mode} ${width} keeps ${scenario.currency} ticks readable and exact data intact`, async ({
        page,
      }, info) => {
        await page.setViewportSize({ width, height: 900 });
        await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
        const fixture = await installPilotFixtures(page, "expenses");
        const errors: string[] = [];
        page.on("pageerror", (error) =>
          errors.push(error.stack ?? error.message),
        );
        await page.goto(
          `/expenses?date=${scenario.date}&currency=${scenario.currency}`,
        );
        await expect(
          page.getByRole("heading", { name: scenario.merchant, exact: true }),
        ).toBeVisible();
        const composition = page.getByRole("region", {
          name: "Spend by category",
          exact: true,
        });
        await expect(composition).toBeVisible();
        await expect
          .poll(async () => {
            const snapshot = await page.evaluate(inspectPilotCharts);
            expect(snapshot.verified).toBe(true);
            const chart = snapshot.verified
              ? snapshot.charts.find((chart) =>
                  chart.xAxis.some((axis) => axis.type === "value"),
                )
              : undefined;
            return {
              hideOverlap: chart?.xAxis[0]?.hideOverlap,
              unit: chart?.xAxis[0]?.name,
              bottom: chart?.grid[0]?.bottom,
            };
          })
          .toEqual({ hideOverlap: true, unit: scenario.currency, bottom: 32 });
        const snapshot = await page.evaluate(inspectPilotCharts);
        expect(snapshot.verified).toBe(true);
        const axis = snapshot.verified
          ? snapshot.charts.find((chart) =>
              chart.xAxis.some((axis) => axis.type === "value"),
            )?.xAxis[0]
          : undefined;
        expect(axis?.min).toBe(0);
        expect(axis?.max).toBeGreaterThanOrEqual(
          scenario.currency === "JPY" ? 12345 : 123456,
        );
        expect(axis?.tickLabels.length).toBeGreaterThan(1);
        expect(new Set(axis?.tickLabels).size).toBe(axis?.tickLabels.length);
        for (const label of axis?.tickLabels ?? [])
          expect(label).toMatch(/^[\d,.-]+$/);
        await info.attach("composition.png", {
          body: await composition.screenshot(),
          contentType: "image/png",
        });
        await composition.getByText("View chart data", { exact: true }).click();
        await expect(
          composition.getByRole("cell", { name: scenario.exact, exact: true }),
        ).toBeVisible();
        expect(errors).toEqual([]);
        expect(fixture.unknown).toEqual([]);
      });
    }
  }
}
