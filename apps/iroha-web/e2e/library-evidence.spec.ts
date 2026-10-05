import { test, expect } from "@playwright/test";
import { installLibraryFixtures } from "./library-fixtures";
import { inspectPilotCharts } from "./pilot-charts";
for (const mode of ["light", "dark"] as const) {
  for (const preference of ["reduce", "no-preference"] as const) {
    test(`Library ${mode} ${preference} distributions expose named exact category counts`, async ({
      page,
    }) => {
      const fixture = await installLibraryFixtures(page);
      fixture.zeroRating = true;
      await page.emulateMedia({ colorScheme: mode, reducedMotion: preference });
      await page.goto("/library");
      for (const [name, rows] of [
        ["Completions by year", ["2025 1", "2026 1"]],
        ["Score distribution", ["0 1"]],
        ["By kind", ["Anime 1", "Books 1", "Games 1"]],
      ] as const) {
        await expect(
          page.getByRole("img", { name, exact: true }),
        ).toBeVisible();
        const table = page.getByRole("table", {
          name: `${name} — exact counts`,
          exact: true,
        });
        await expect(table).toBeVisible();
        for (const row of rows)
          await expect(
            table.getByRole("row", { name: row, exact: true }),
          ).toBeVisible();
        await expect(table.getByRole("row")).toHaveCount(rows.length + 1);
        const headingGap = await table.evaluate((node) => {
          const headers = node.querySelectorAll("th");
          const bounds = (element: Element) => {
            const range = document.createRange();
            range.selectNodeContents(element);
            return range.getBoundingClientRect();
          };
          return bounds(headers[1]).left - bounds(headers[0]).right;
        });
        expect(headingGap).toBeGreaterThanOrEqual(12);
      }
      await expect
        .poll(async () => {
          const rendered = await page.evaluate(inspectPilotCharts);
          if (!rendered.verified) return [];
          return rendered.charts.map((chart) => ({
            label: chart.label,
            animation: chart.animation,
            duration: chart.animationDuration,
          }));
        })
        .toEqual(
          ["Completions by year", "Score distribution", "By kind"].map(
            (label) => ({
              label,
              animation: preference === "no-preference",
              duration: preference === "reduce" ? 0 : 500,
            }),
          ),
        );
      expect(fixture.unknown).toEqual([]);
    });
  }
}
