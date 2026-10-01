import { expect, test } from "@playwright/test";
import type { MetricSeriesResponse } from "../src/lib/api";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

const cases = [
  {
    name: "one observation",
    values: [null, 12345, null],
    showSymbol: true,
    sizes: [6, 6, 6],
  },
  {
    name: "observed zero",
    values: [null, 0, null],
    showSymbol: true,
    sizes: [6, 6, 6],
  },
  {
    name: "separated observations",
    values: [12345, null, 0],
    showSymbol: true,
    sizes: [6, 6, 6],
  },
  {
    name: "mixed connected and isolated observations",
    values: [12345, 0, null, 7000],
    showSymbol: true,
    sizes: [6, 6, 6, 6],
  },
  {
    name: "connected observations",
    values: [12345, 0, 7000],
    showSymbol: false,
    sizes: [6, 6, 6],
  },
];

for (const mode of ["light", "dark"] as const) {
  for (const scenario of cases) {
    test(`Metrics ${mode} keeps ${scenario.name} visible without joining gaps`, async ({
      page,
    }, info) => {
      const periods = ["2026-05", "2026-06", "2026-07", "2026-08"].slice(
        -scenario.values.length,
      );
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      const fixture = await installPilotFixtures(page, "metrics");
      const errors: string[] = [];
      page.on("pageerror", (error) =>
        errors.push(error.stack ?? error.message),
      );
      await page.route("**/api/v1/metrics/daily.steps/series**", (route) => {
        const url = new URL(route.request().url());
        const observed = scenario.values.filter(Number.isFinite).length;
        const response: MetricSeriesResponse = {
          schema: "metric-series.v1",
          metric_id: "daily.steps",
          label: "Steps",
          unit: "count",
          value_type: "number",
          period: {
            grain: "month",
            from: url.searchParams.get("from")!,
            to: url.searchParams.get("to")!,
            timezone: "Asia/Tokyo",
          },
          series: [
            {
              dimensions: {},
              points: scenario.values.map((value, index) => ({
                period: periods[index],
                value,
                observed_days: value === null ? 0 : 1,
              })),
              coverage: {
                expected_periods: 12,
                observed_periods: observed,
                observation_state: "observed",
                collection_completeness: "unknown",
              },
              source: {
                kind: "canonical",
                method: "synthetic.v1",
                source_kinds: ["synthetic"],
              },
            },
          ],
        };
        return route.fulfill({ json: response });
      });
      await page.goto("/metrics?date=2026-08");
      await expect(
        page.getByRole("group", { name: "Steps view", exact: true }),
      ).toBeVisible();
      await expect
        .poll(async () => {
          const snapshot = await page.evaluate(inspectPilotCharts);
          expect(snapshot.verified).toBe(true);
          const series = snapshot.verified
            ? snapshot.charts.find((chart) =>
                chart.label?.startsWith("Daily trends"),
              )?.series[0]
            : undefined;
          return {
            data: series?.data,
            showSymbol: series?.showSymbol,
            showAllSymbol: series?.showAllSymbol,
            connectNulls: series?.connectNulls,
            sizes: series?.symbolSizes,
          };
        })
        .toEqual({
          data: scenario.values,
          showSymbol: scenario.showSymbol,
          showAllSymbol: true,
          connectNulls: false,
          sizes: scenario.sizes,
        });
      const expectedColor = await page
        .getByRole("img", { name: /^Daily trends/ })
        .evaluate((el) =>
          getComputedStyle(el).getPropertyValue("--accent").trim(),
        );
      await expect
        .poll(async () => {
          const snapshot = await page.evaluate(inspectPilotCharts);
          return snapshot.verified
            ? snapshot.charts.find((chart) =>
                chart.label?.startsWith("Daily trends"),
              )?.series[0]?.color
            : undefined;
        })
        .toBe(expectedColor);
      await page.getByText("View trend data", { exact: true }).click();
      const table = page.getByRole("table", {
        name: "Daily trend values",
        exact: true,
      });
      for (let index = 0; index < periods.length; index++) {
        await expect(
          table.getByRole("row").filter({ hasText: periods[index] }),
        ).toContainText(
          scenario.values[index] === null
            ? "No observation"
            : `${scenario.values[index]} count`,
        );
      }
      await info.attach("metric-points.png", {
        body: await page.screenshot({ fullPage: true }),
        contentType: "image/png",
      });
      expect(errors).toEqual([]);
      expect(fixture.unknown).toEqual([]);
    });
  }
}
