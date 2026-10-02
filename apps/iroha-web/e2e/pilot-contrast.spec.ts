import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { measureRenderedContrast } from "./rendered-contrast";
import { inspectPilotCharts } from "./pilot-charts";

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
        expect(measured.canvasText.length).toBeGreaterThan(0);
        const charts = await page.evaluate(inspectPilotCharts);
        expect(charts.verified).toBe(true);
        if (charts.verified) {
          for (const chart of charts.charts) {
            for (const axis of [...chart.xAxis, ...chart.yAxis]) {
              if (!axis.name) continue;
              expect(
                measured.canvasText.some(
                  (text) =>
                    text.chart === chart.label && text.text === axis.name,
                ),
                `Rendered axis title: ${chart.label} / ${axis.name}`,
              ).toBe(true);
            }
          }
        }
        for (const text of measured.canvasText) {
          const label = `${text.chart}: ${text.text}`;
          expect(text.ratio, label).not.toBeNull();
          expect(text.ratio!, label).toBeGreaterThanOrEqual(4.5);
        }
        for (const mark of measured.marks) {
          expect(mark.ratio, `${mark.chart}: ${mark.series}`).not.toBeNull();
          expect(
            mark.ratio!,
            `${mark.chart}: ${mark.series}`,
          ).toBeGreaterThanOrEqual(3);
        }
        if (pilot === "public" && mode === "light" && width === 320) {
          // Negative control: the option still declares its original color, but
          // the renderer paints an invisible axis title. The probe must notice.
          const damaged = await page.evaluate(async () => {
            const entry = performance
              .getEntriesByType("resource")
              .find((resource) =>
                /\/echarts_core\.js(?:\?|$)/.test(resource.name),
              )!;
            const core = await import(entry.name);
            const el = Array.from(
              document.querySelectorAll('[role="img"]'),
            ).find((el) => core.getInstanceByDom(el))!;
            const chart = core.getInstanceByDom(el);
            const title = chart
              .getOption()
              .yAxis.find((axis: { name?: string }) => axis.name).name;
            const span = chart
              .getZr()
              .storage.getDisplayList()
              .find(
                (span: { type: string; style: { text?: string } }) =>
                  span.type === "tspan" && span.style.text === title,
              );
            const color = span.style.fill;
            span.setStyle({ fill: "rgba(0,0,0,0)" });
            chart.getZr().refreshImmediately();
            return { chart: el.getAttribute("aria-label"), text: title, color };
          });
          try {
            const poisoned = await page.evaluate(measureRenderedContrast, true);
            expect(
              poisoned.canvasText.find(
                (text) =>
                  text.chart === damaged.chart && text.text === damaged.text,
              )?.ratio,
            ).toBe(1);
          } finally {
            await page.evaluate(async (original) => {
              const entry = performance
                .getEntriesByType("resource")
                .find((resource) =>
                  /\/echarts_core\.js(?:\?|$)/.test(resource.name),
                )!;
              const core = await import(entry.name);
              const el = Array.from(
                document.querySelectorAll('[role="img"]'),
              ).find((el) => el.getAttribute("aria-label") === original.chart)!;
              const chart = core.getInstanceByDom(el);
              const span = chart
                .getZr()
                .storage.getDisplayList()
                .find(
                  (span: { type: string; style: { text?: string } }) =>
                    span.type === "tspan" && span.style.text === original.text,
                );
              span.setStyle({ fill: original.color });
              chart.getZr().refreshImmediately();
            }, damaged);
          }
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
