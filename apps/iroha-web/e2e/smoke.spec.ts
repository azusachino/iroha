// Smoke check cloned from the playwright-verify skill: the first route renders
// its main landmark without console errors. Signed out, that is the sign-in
// screen; behavior checks live in their own specs.
import { expect, test } from "@playwright/test";
import { fakeSession, signedOut } from "./session";

const ROUTE = "/";

test("first route renders without console errors", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });
  page.on("pageerror", (error) => errors.push(error.message));
  await fakeSession(page, signedOut);
  await page.route("**/public/v1/summary", (route) =>
    route.fulfill({
      json: {
        totals: {
          activity_count: 0,
          distance_m: 0,
          duration_s: 0,
          elevation_gain_m: 0,
        },
        by_year: [],
      },
    }),
  );
  await page.route("**/public/v1/activities", (route) =>
    route.fulfill({ json: [] }),
  );
  await page.route("**/public/v1/meta", (route) =>
    route.fulfill({ json: { generated_at: "2026-11-01T00:00:00Z" } }),
  );
  await page.route("**/public/v1/routes", (route) =>
    route.fulfill({ json: { type: "FeatureCollection", features: [] } }),
  );

  await page.goto(ROUTE);

  await expect(page.getByRole("main")).toBeVisible();
  expect(errors).toEqual([]);
});
