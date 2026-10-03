import { existsSync, readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";

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
        activity_id: "private-map-fixture",
        sport_type: "run",
        year: "2026",
        city: "Berlin",
        location_status: "resolved",
      },
    },
  ],
};
const manifestUrl = new URL(
  "../.svelte-kit/output/client/.vite/manifest.json",
  import.meta.url,
);
const manifest = existsSync(manifestUrl)
  ? JSON.parse(readFileSync(manifestUrl, "utf8"))
  : {};
const productionMapFiles = Object.entries(manifest)
  .filter(
    ([key, entry]) =>
      key.endsWith("src/lib/maplibre.ts") ||
      (entry as { name?: string }).name === "maplibre",
  )
  .map(([, entry]) => (entry as { file: string }).file);
function isMapLibrary(url: string) {
  const path = new URL(url).pathname;
  return (
    /\/src\/lib\/maplibre\.ts$/.test(path) ||
    productionMapFiles.some((file) => path.endsWith(`/${file}`))
  );
}

function isWorker(url: string) {
  const parsed = new URL(url);
  return (
    parsed.pathname.includes("maplibre-gl-worker") &&
    (parsed.searchParams.has("worker_file") ||
      parsed.pathname.includes("/workers/"))
  );
}

test("private single raster tile failure is partial and leaves the route map usable", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "overview");
  await page.route("**/api/v1/activities/routes*", (route) =>
    route.fulfill({ json: routes }),
  );
  let failedTiles = 0;
  let loadedTiles = 0;
  await page.route("**/tile.openstreetmap.org/**", async (route) => {
    if (failedTiles === 0) {
      failedTiles++;
      return route.fulfill({ status: 503, body: "tile unavailable" });
    }
    loadedTiles++;
    return route.fulfill({
      contentType: "image/png",
      body: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR4nGP4DwQACfsD/fteaysAAAAASUVORK5CYII=",
        "base64",
      ),
    });
  });
  await page.goto("/overview");
  await expect.poll(() => loadedTiles).toBeGreaterThan(0);
  expect(failedTiles).toBe(1);
  await expect(page.getByRole("status")).toContainText(
    "Basemap tile loading failed. Route data remains available.",
  );
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
  await page.getByText("View route summaries", { exact: true }).click();
  await expect(
    page.getByRole("table", {
      name: "Recorded route summaries",
      exact: true,
    }),
  ).toContainText("Berlin");
  expect(fixture.unknown).toEqual([]);
});

test("private no-route overview does not request the map engine", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "overview");
  if (process.env.E2E_BASE_URL) expect(productionMapFiles).toHaveLength(1);
  const engines: string[] = [];
  page.on("request", (request) => {
    if (isMapLibrary(request.url())) engines.push(request.url());
  });
  await page.goto("/overview");
  await expect(
    page.getByText("No routes recorded yet.", { exact: true }),
  ).toBeVisible();
  expect(engines).toEqual([]);
  expect(fixture.unknown).toEqual([]);
});

for (const failLibrary of [false, true]) {
  test(`private deferred map library ${failLibrary ? "failure recovers by reload" : "keeps route summaries available"}`, async ({
    page,
  }) => {
    const fixture = await installPilotFixtures(page, "overview");
    await page.route("**/api/v1/activities/routes*", (route) =>
      route.fulfill({ json: routes }),
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
    const exceptions: string[] = [];
    page.on("pageerror", (error) => exceptions.push(error.message));
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    let fail = failLibrary;
    await page.route("**/*", async (route) => {
      if (!isMapLibrary(route.request().url())) return route.fallback();
      await gate;
      return fail
        ? route.fulfill({
            status: 503,
            contentType: "text/javascript",
            body: "",
          })
        : route.continue();
    });
    try {
      await page.goto("/overview");
      await expect(
        page.getByText("Loading interactive map…", { exact: true }),
      ).toBeVisible();
      await expect(page.locator(".map")).toHaveAttribute("aria-busy", "true");
      const chartRange = page
        .getByRole("group", { name: "Distance chart range", exact: true })
        .getByRole("button")
        .first();
      await chartRange.click();
      await expect(chartRange).toHaveAttribute("aria-pressed", "true");
      await expect(page.getByRole("img").first()).toBeVisible();
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
    if (failLibrary) {
      await expect(page.getByRole("alert")).toContainText(
        "Interactive map unavailable.",
      );
      await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
      fail = false;
      await page
        .getByRole("button", { name: "Reload page", exact: true })
        .focus();
      await page.keyboard.press("Enter");
    }
    await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
    await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
    expect(exceptions).toEqual([]);
    expect(fixture.unknown).toEqual([]);
  });
}

for (const failWorker of [false, true]) {
  test(`private map worker ${failWorker ? "failure is visible" : "loads the GeoJSON source"}`, async ({
    page,
  }) => {
    const fixture = await installPilotFixtures(page, "overview");
    await page.route("**/api/v1/activities/routes*", (route) =>
      route.fulfill({ json: routes }),
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
    const responses: number[] = [];
    page.on("response", (response) => {
      if (isWorker(response.url())) responses.push(response.status());
    });
    if (failWorker)
      await page.route("**/*", (route) =>
        isWorker(route.request().url())
          ? route.fulfill({
              status: 404,
              contentType: "text/javascript",
              body: "",
            })
          : route.fallback(),
      );
    await page.goto("/overview");
    await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
    if (failWorker) {
      await expect(page.getByRole("alert")).toContainText(
        "Interactive map unavailable. Route summaries remain below.",
      );
      await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
    } else {
      await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
      await expect.poll(() => responses.length).toBeGreaterThan(0);
      expect(responses.every((status) => status === 200)).toBe(true);
    }
    await page.getByText("View route summaries", { exact: true }).click();
    await expect(
      page.getByRole("table", {
        name: "Recorded route summaries",
        exact: true,
      }),
    ).toContainText("Berlin");
    expect(fixture.unknown).toEqual([]);
  });
}
