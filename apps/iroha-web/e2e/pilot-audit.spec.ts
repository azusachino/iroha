// Report-only, owner-authorized pilot baseline. Violations are recorded, not
// turned into claims of conformance. Existing chromium regression checks stay blocking.
import { expect, test } from "@playwright/test";
import {
  installPilotFixtures,
  type Pilot,
  type Scenario,
} from "./pilot-fixtures";
import { measurePilotPage } from "./pilot-measurements";
import { inspectPilotCharts } from "./pilot-charts";
import { PUBLIC_BASE_URL } from "../playwright.config";

const pilots: Pilot[] = ["overview", "expenses", "public"];
const paths: Record<Pilot, string> = {
  overview: "/overview",
  expenses: "/expenses?date=2026-08",
  public: PUBLIC_BASE_URL,
};
const titles: Record<Pilot, string> = {
  overview: "Overview",
  expenses: "Expenses",
  public: "harus",
};

for (const pilot of pilots) {
  for (const mode of ["light", "dark"] as const) {
    for (const width of [320, 768, 1280]) {
      test(`${pilot} ${mode} ${width}`, async ({ page }, info) => {
        await page.setViewportSize({ width, height: 900 });
        await page.emulateMedia({ colorScheme: mode });
        const fixture = await installPilotFixtures(page, pilot);
        const errors: string[] = [];
        page.on("pageerror", (error) =>
          errors.push(error.stack ?? error.message),
        );
        await page.goto(paths[pilot]);
        await expect(page).toHaveTitle(new RegExp(titles[pilot], "i"));
        if (pilot === "expenses")
          await expect(
            page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
          ).toBeVisible();
        if (pilot === "overview")
          await expect(
            page.getByRole("heading", {
              name: "Overview",
            }),
          ).toBeVisible();
        if (pilot === "public")
          await expect(page.getByRole("table").last()).toBeVisible();
        const normal = await page.evaluate(measurePilotPage);
        const charts = await page.evaluate(inspectPilotCharts);
        const aria = await page.locator("body").ariaSnapshot();
        await info.attach("populated.png", {
          body: await page.screenshot({ fullPage: true }),
          contentType: "image/png",
        });
        const keyboard = [];
        for (let stop = 0; stop < 12; stop++) {
          await page.keyboard.press("Tab");
          keyboard.push(
            await page.evaluate(() => {
              const el = document.activeElement!;
              const s = getComputedStyle(el);
              const r = el.getBoundingClientRect();
              return {
                tag: el.tagName,
                name:
                  el.getAttribute("aria-label") ||
                  el.textContent?.trim().slice(0, 100),
                outline: s.outline,
                shadow: s.boxShadow,
                x: r.x,
                right: r.right,
                ariaHidden: Boolean(el.closest("[aria-hidden='true'],[inert]")),
              };
            }),
          );
        }
        // CSS zoom is a diagnostic surrogate, not native browser-zoom evidence.
        await page.evaluate(() => {
          document.body.style.zoom = "2";
        });
        const cssZoom = await page.evaluate(measurePilotPage);
        await page.evaluate(() => {
          document.body.style.zoom = "";
        });
        await page.emulateMedia({ reducedMotion: "reduce" });
        await page.reload();
        if (pilot === "expenses")
          await expect(
            page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
          ).toBeVisible();
        if (pilot === "overview")
          await expect(
            page.getByText("20.50 km", { exact: true }).first(),
          ).toBeVisible();
        if (pilot === "public")
          await expect(page.getByRole("table").last()).toBeVisible();
        const reducedMotion = await page.evaluate(measurePilotPage);
        const reducedCharts = await page.evaluate(inspectPilotCharts);
        const record = {
          pilot,
          mode,
          width,
          normal,
          charts,
          reducedCharts,
          keyboard,
          cssZoom,
          reducedMotion,
          errors,
          requests: fixture.requests,
          unknownRequests: fixture.unknown,
          aria,
        };
        await info.attach("baseline.json", {
          body: JSON.stringify(record, null, 2),
          contentType: "application/json",
        });
        console.log(
          JSON.stringify({
            pilot,
            mode,
            width,
            runtimeErrors: errors.length,
            documentOverflow: normal.scrollWidth - width,
            clippedControls: normal.controls
              .filter((c) => c.clipped)
              .map((c) => c.name),
            contrastCandidates: normal.textPairs
              .filter((p) => p.ratio < p.required)
              .map((p) => ({
                text: p.text,
                ratio: p.ratio,
                required: p.required,
              })),
            unmeasuredText: normal.unmeasuredText,
            unknownRequests: fixture.unknown,
          }),
        );
      });
    }
  }
}

for (const pilot of pilots) {
  for (const scenario of ["empty", "error"] as Scenario[]) {
    test(`${pilot} ${scenario}`, async ({ page }, info) => {
      const fixture = await installPilotFixtures(page, pilot, scenario);
      const errors: string[] = [];
      page.on("pageerror", (error) =>
        errors.push(error.stack ?? error.message),
      );
      await page.goto(paths[pilot]);
      await expect(page.locator("body")).toHaveText(/\S/, {
        useInnerText: true,
      });
      if (scenario === "error" && pilot !== "public")
        await expect(page.getByRole("alert").first()).toBeVisible();
      if (scenario === "empty" && pilot === "expenses")
        await expect(
          page.getByText(/No .*expenses|No records|No entries/i).first(),
        ).toBeVisible();
      const aria = await page.locator("body").ariaSnapshot();
      await info.attach("state.json", {
        body: JSON.stringify(
          {
            pilot,
            scenario,
            errors,
            requests: fixture.requests,
            unknownRequests: fixture.unknown,
            aria,
          },
          null,
          2,
        ),
        contentType: "application/json",
      });
      console.log(JSON.stringify({ pilot, scenario, errors, aria }));
    });
  }
}

test("scope controls exercise real request changes", async ({ page }, info) => {
  const fixture = await installPilotFixtures(page, "expenses");
  await page.goto(paths.expenses);
  await expect(
    page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("combobox", { name: "Filter by month" })
    .selectOption("");
  await expect(page).toHaveURL(/date=2026(?:&|$)/);
  await expect(
    page.getByRole("button", { name: /Synthetic shop/ }),
  ).toBeVisible();
  await page
    .getByRole("combobox", { name: "Filter by month" })
    .selectOption("9");
  await expect(page).toHaveURL(/date=2026-09/);
  await expect(
    page.getByRole("heading", { name: "Synthetic shop", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /Synthetic cafe/ }),
  ).toHaveCount(0);
  await info.attach("expense-scope.json", {
    body: JSON.stringify(fixture, null, 2),
    contentType: "application/json",
  });
});

test("public year changes keep prior-year chart lifecycle observable", async ({
  page,
}, info) => {
  const fixture = await installPilotFixtures(page, "public");
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.stack ?? error.message));
  await page.goto(PUBLIC_BASE_URL);
  await expect(
    page.getByRole("button", { name: "2025", exact: true }),
  ).toBeVisible();
  const states = [];
  for (const year of ["2025", "2026", "2025", "2026"]) {
    await page.getByRole("button", { name: year, exact: true }).click();
    await expect(
      page.getByRole("img", {
        name: `Cumulative distance for ${year}`,
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("cell", { name: new RegExp(`${year}-08-14`) }),
    ).toBeVisible();
    states.push({ year, option: await page.evaluate(inspectPilotCharts) });
  }
  await info.attach("chart-lifecycle.json", {
    body: JSON.stringify({ errors, states, ...fixture }, null, 2),
    contentType: "application/json",
  });
  console.log(JSON.stringify({ test: "public-chart-lifecycle", errors }));
});
