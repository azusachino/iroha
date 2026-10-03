import { existsSync, readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

const manifestUrl = new URL(
  "../../iroha-public-site/.svelte-kit/output/client/.vite/manifest.json",
  import.meta.url,
);
const manifest = existsSync(manifestUrl)
  ? JSON.parse(readFileSync(manifestUrl, "utf8"))
  : {};
const productionMapFiles = Object.entries(manifest)
  .filter(([key]) => key.endsWith("src/lib/maplibre.ts"))
  .map(([, entry]) => (entry as { file: string }).file);
function isMapLibrary(url: string) {
  const path = new URL(url).pathname;
  return (
    /\/src\/lib\/maplibre\.ts$/.test(path) ||
    productionMapFiles.some((file) => path.endsWith(`/${file}`))
  );
}

const routes = {
  type: "FeatureCollection",
  features: [
    {
      type: "Feature",
      geometry: {
        type: "LineString",
        coordinates: [
          [13.4, 52.52],
          [13.41, 52.53],
        ],
      },
      properties: {
        activity_id: "map-record",
        sport_type: "run",
        year: "2026",
        city: "Berlin",
        location_status: "resolved",
      },
    },
  ],
};
const activity = {
  id: "map-record",
  sport_type: "run",
  title: "Map loading fixture",
  started_at: "2026-08-01T00:00:00Z",
  timezone: "UTC",
  distance_m: 1500,
  duration_s: 600,
};

async function installMapFixtures(page: import("@playwright/test").Page) {
  const fixture = await installPilotFixtures(page, "public");
  await page.route("**/public/v1/routes", (route) =>
    route.fulfill({ json: routes }),
  );
  await page.route("**/public/v1/activities", (route) =>
    route.fulfill({ json: [activity] }),
  );
  await page.route("**/public/v1/activities/map-record", (route) =>
    route.fulfill({ status: 503, json: { error: "summary-only fixture" } }),
  );
  await page.route("**/tile.openstreetmap.org/**", (route) =>
    route.fulfill({
      contentType: "image/png",
      body: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR4nGP4DwQACfsD/fteaysAAAAASUVORK5CYII=",
        "base64",
      ),
    }),
  );
  return fixture;
}

test("public no-route landing does not request the map engine", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "public");
  if (process.env.E2E_PUBLIC_BASE_URL) {
    const entry = manifest["src/lib/maplibre.ts"];
    expect(entry?.isDynamicEntry).toBe(true);
    const landing =
      manifest[".svelte-kit/generated/client-optimized/nodes/2.js"];
    expect(landing?.dynamicImports).toContain("src/lib/maplibre.ts");
    expect(landing?.imports ?? []).not.toContain("src/lib/maplibre.ts");
  }
  const engineRequests: string[] = [];
  page.on("request", (request) => {
    if (isMapLibrary(request.url())) engineRequests.push(request.url());
  });
  await page.goto(PUBLIC_BASE_URL);
  await expect(
    page.getByRole("heading", { name: "Activities", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("No routes recorded yet.", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("img", { name: "Monthly distance for 2026", exact: true }),
  ).toBeVisible();
  expect(engineRequests).toEqual([]);
  expect(fixture.unknown).toEqual([]);
});

test("public deferred map leaves charts, records and route summaries usable", async ({
  page,
}) => {
  const fixture = await installMapFixtures(page);
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  let engineRequests = 0;
  await page.route("**/*", async (route) => {
    if (!isMapLibrary(route.request().url())) return route.fallback();
    engineRequests++;
    await gate;
    await route.continue();
  });
  try {
    await page.goto(PUBLIC_BASE_URL);
    await expect(page.getByRole("status")).toContainText(
      "Loading interactive map…",
    );
    await expect.poll(() => engineRequests).toBe(1);
    await expect(
      page.getByRole("img", { name: "Monthly distance for 2026", exact: true }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Activity records", exact: true })
        .locator("tbody tr"),
    ).toHaveCount(1);
    await page.getByText("View route summaries", { exact: true }).click();
    await expect(
      page.getByRole("table", {
        name: "Recorded route summaries",
        exact: true,
      }),
    ).toContainText("Berlin");
  } finally {
    release();
  }
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await expect(
    page.getByText("Loading interactive map…", { exact: true }),
  ).toHaveCount(0);
  expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
    [],
  );
});

test("public map load failure keeps records usable and recovers by explicit reload", async ({
  page,
}) => {
  const fixture = await installMapFixtures(page);
  const exceptions: string[] = [];
  page.on("pageerror", (error) => exceptions.push(error.message));
  let fail = true;
  await page.route("**/*", (route) => {
    if (!isMapLibrary(route.request().url()) || !fail) return route.fallback();
    return route.fulfill({
      status: 503,
      contentType: "text/javascript",
      body: "",
    });
  });
  await page.goto(PUBLIC_BASE_URL);
  await expect(page.getByRole("alert")).toContainText(
    "Interactive map unavailable. Route summaries remain below.",
  );
  await expect(
    page
      .getByRole("region", { name: "Activity records", exact: true })
      .locator("tbody tr"),
  ).toHaveCount(1);
  await page.getByText("View route summaries", { exact: true }).click();
  await expect(
    page.getByRole("table", { name: "Recorded route summaries", exact: true }),
  ).toContainText("Berlin");
  fail = false;
  const retry = page.getByRole("button", { name: "Reload page", exact: true });
  await retry.focus();
  await page.keyboard.press("Enter");
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
  expect(exceptions).toEqual([]);
  expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
    [],
  );
});

test("public detail deep link loads its map and back navigation destroys the old canvas", async ({
  page,
}) => {
  await installMapFixtures(page);
  await page.goto(`${PUBLIC_BASE_URL}/?activity=map-record`);
  await expect(
    page.getByRole("heading", { name: "Map loading fixture", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await page
    .getByRole("link", { name: "← Back to archive", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Activities", exact: true }),
  ).toBeVisible();
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
  await page.getByRole("button", { name: "2026", exact: true }).focus();
});
