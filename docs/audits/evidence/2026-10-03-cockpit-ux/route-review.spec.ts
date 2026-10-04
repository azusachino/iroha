// Read-only report collection. All API responses are synthetic; no live backend.
import {
  test,
  expect,
} from "../../../../apps/iroha-web/node_modules/@playwright/test/index.mjs";
import { installPilotFixtures } from "../../../../apps/iroha-web/e2e/pilot-fixtures";
import { measurePilotPage } from "../../../../apps/iroha-web/e2e/pilot-measurements";
test("Overview compact reading order", async ({ page }, info) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
  const fixture = await installPilotFixtures(page, "overview");
  await page.goto("/overview");
  await expect(
    page.getByText("20.50 km", { exact: true }).first(),
  ).toBeVisible();
  const headings = {};
  for (const name of [
    "Activity through the year",
    "Monthly distance",
    "Recent movement",
  ]) {
    headings[name] = await page
      .getByRole("heading", { name, exact: true })
      .boundingBox();
  }
  await info.attach("reading-order.json", {
    body: JSON.stringify({ headings, errors: fixture.unknown }),
    contentType: "application/json",
  });
});
const routes = [
  "/",
  "/patterns",
  "/motion",
  "/motion/synthetic",
  "/night",
  "/night/synthetic",
  "/library",
  "/library/synthetic",
  "/reports?date=2026-08",
  "/manual",
  "/to-go",
  "/admin",
  "/expenses?date=2026-08",
];
for (const route of routes) {
  test(`${route} first failure 320 light`, async ({ page }, info) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "overview", "error");
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route("**/*", (request) => {
      const url = new URL(request.request().url());
      return url.hostname === "127.0.0.1"
        ? request.fallback()
        : request.abort();
    });
    await page.goto(route);
    await expect(page.getByRole("main")).toBeVisible();
    if (route !== "/manual")
      await expect(
        page.getByText(/503 Service Unavailable/).first(),
      ).toBeVisible();
    await info.attach("first-failure.png", {
      body: await page.screenshot({ fullPage: true }),
      contentType: "image/png",
    });
    const record = {
      route,
      scenario: "first-failure",
      errors,
      requests: fixture.requests,
      aria: await page.locator("body").ariaSnapshot(),
      measurement: await page.evaluate(measurePilotPage),
    };
    await info.attach("route.json", {
      body: JSON.stringify(record, null, 2),
      contentType: "application/json",
    });
  });
}
