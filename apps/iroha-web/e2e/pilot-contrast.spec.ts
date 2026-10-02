import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { measureRenderedContrast } from "./rendered-contrast";

for (const pilot of ["overview", "expenses", "metrics", "public"] as const) {
  for (const mode of ["light", "dark"] as const) {
    for (const width of [320, 768, 1280]) {
      test(`${pilot} ${mode} ${width} rendered text and chart paint meet contrast`, async ({
        page,
      }, info) => {
        await page.setViewportSize({ width, height: 900 });
        await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
        const fixture = await installPilotFixtures(page, pilot);
        await page.goto(
          pilot === "public" ? PUBLIC_BASE_URL : `/${pilot}?date=2026-08`,
        );
        await expect
          .poll(
            async () =>
              (await page.evaluate(measureRenderedContrast, false)).marks
                .length,
          )
          .toBeGreaterThan(0);
        const measured = await page.evaluate(measureRenderedContrast, true);
        expect(measured.textPairs.length).toBeGreaterThan(0);
        for (const pair of measured.textPairs) {
          expect(pair.ratio, pair.text).not.toBeNull();
          expect(pair.ratio!, pair.text).toBeGreaterThanOrEqual(pair.required);
        }
        for (const mark of measured.marks) {
          expect(mark.ratio, `${mark.chart}: ${mark.series}`).not.toBeNull();
          expect(
            mark.ratio!,
            `${mark.chart}: ${mark.series}`,
          ).toBeGreaterThanOrEqual(3);
        }
        await info.attach("rendered-contrast.json", {
          body: JSON.stringify(measured),
          contentType: "application/json",
        });
        await info.attach("contrast.png", {
          body: await page.screenshot({ fullPage: true }),
          contentType: "image/png",
        });
        expect(fixture.unknown).toEqual([]);
      });
    }
  }
}
