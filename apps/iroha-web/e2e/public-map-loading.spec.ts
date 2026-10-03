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

function isDetailLibrary(url: string) {
  const path = new URL(url).pathname;
  const entry = manifest["src/lib/components/ActivityDetailView.svelte"];
  return (
    path.endsWith("/src/lib/components/ActivityDetailView.svelte") ||
    (entry?.file && path.endsWith(`/${entry.file}`))
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
    const landingEntry =
      manifest[".svelte-kit/generated/client-optimized/nodes/2.js"];
    const staticKeys = new Set<string>();
    function visit(key: string) {
      if (staticKeys.has(key)) return;
      staticKeys.add(key);
      for (const child of manifest[key]?.imports ?? []) visit(child);
    }
    visit(".svelte-kit/generated/client-optimized/nodes/2.js");
    // Deferred edges can live in shared chunks, not just the route entry.
    const landing = {
      ...landingEntry,
      dynamicImports: [...staticKeys].flatMap(
        (key) => manifest[key]?.dynamicImports ?? [],
      ),
    };
    expect(landing?.dynamicImports).toContain("src/lib/maplibre.ts");
    expect(landing?.imports ?? []).not.toContain("src/lib/maplibre.ts");
    expect([...staticKeys]).not.toContain("src/lib/maplibre.ts");
    expect([...staticKeys]).not.toContain(
      "src/lib/components/ActivityDetailView.svelte",
    );
    expect(landing?.dynamicImports).toContain(
      "src/lib/components/ActivityDetailView.svelte",
    );
  }
  const engineRequests: string[] = [];
  const detailRequests: string[] = [];
  page.on("request", (request) => {
    if (isMapLibrary(request.url())) engineRequests.push(request.url());
    if (isDetailLibrary(request.url())) detailRequests.push(request.url());
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
  expect(detailRequests).toEqual([]);
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

test("public single raster tile failure is partial and leaves the route map usable", async ({
  page,
}) => {
  await installMapFixtures(page);
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
  await page.goto(PUBLIC_BASE_URL);
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
    page.getByRole("table", { name: "Recorded route summaries", exact: true }),
  ).toContainText("Berlin");
});

for (const leavePending of [false, true]) {
  test(`public deferred detail ${leavePending ? "does not resurrect after back navigation" : "keeps selected record visible"}`, async ({
    page,
  }) => {
    await installMapFixtures(page);
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/*", async (route) => {
      if (!isDetailLibrary(route.request().url())) return route.fallback();
      await gate;
      await route.continue();
    });
    try {
      await page.goto(`${PUBLIC_BASE_URL}/?activity=map-record`);
      await expect(page.getByRole("status")).toContainText(
        "Loading activity details…",
      );
      await expect(
        page.getByRole("heading", { name: activity.title, exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Distance: 1.50 km · Duration: 10:00", { exact: true }),
      ).toBeVisible();
      if (leavePending)
        await page
          .getByRole("link", { name: "← Back to archive", exact: true })
          .click();
    } finally {
      release();
    }
    if (leavePending) {
      await expect(
        page.getByRole("heading", { name: "Activities", exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: activity.title, exact: true }),
      ).toHaveCount(0);
      // Wait for the same module on a fresh selection, then verify normal back navigation.
      await page
        .getByRole("region", { name: "Activity records", exact: true })
        .getByRole("link")
        .first()
        .click();
    }
    await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
    await expect(
      page.getByText("Loading activity details…", { exact: true }),
    ).toHaveCount(0);
    await page
      .getByRole("link", { name: "← Back to archive", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "Activities", exact: true }),
    ).toBeVisible();
  });
}

test("public detail module failure retains the record and recovers by keyboard reload", async ({
  page,
}) => {
  await installMapFixtures(page);
  const exceptions: string[] = [];
  page.on("pageerror", (error) => exceptions.push(error.message));
  let fail = true;
  await page.route("**/*", (route) =>
    !isDetailLibrary(route.request().url()) || !fail
      ? route.fallback()
      : route.fulfill({
          status: 503,
          contentType: "text/javascript",
          body: "",
        }),
  );
  await page.goto(`${PUBLIC_BASE_URL}/?activity=map-record`);
  await expect(page.getByRole("alert")).toContainText(
    "Activity detail view unavailable. Record summary remains.",
  );
  await expect(
    page.getByRole("heading", { name: activity.title, exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Distance: 1.50 km · Duration: 10:00", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "← Back to archive", exact: true }),
  ).toBeVisible();
  fail = false;
  await page.getByRole("button", { name: "Reload page", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(page.locator(".maplibregl-canvas")).toHaveCount(1);
  await expect(page.getByRole("alert")).toHaveCount(0);
  expect(exceptions).toEqual([]);
});

test("public selected rich detail still loads its approved view", async ({
  page,
}) => {
  await installMapFixtures(page);
  await page.route("**/public/v1/activities/map-record", (route) =>
    route.fulfill({
      json: {
        activity,
        route: [],
        samplings: [],
        laps: [],
      },
    }),
  );
  await page.goto(`${PUBLIC_BASE_URL}/?activity=map-record`);
  await expect(
    page.getByRole("heading", { name: activity.title, level: 1, exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "← Back to archive", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("public worker failure is visible and leaves route summaries usable", async ({
  page,
}) => {
  await installMapFixtures(page);
  await page.route("**/*", (route) => {
    const url = new URL(route.request().url());
    const worker =
      url.pathname.includes("maplibre-gl-worker") &&
      (url.searchParams.has("worker_file") ||
        url.pathname.includes("/workers/"));
    return worker
      ? route.fulfill({ status: 404, contentType: "text/javascript", body: "" })
      : route.fallback();
  });
  await page.goto(PUBLIC_BASE_URL);
  await expect(page.getByRole("alert")).toContainText(
    "Interactive map unavailable. Route summaries remain below.",
  );
  await expect(page.locator(".map")).toHaveAttribute("aria-busy", "false");
  await page.getByText("View route summaries", { exact: true }).click();
  await expect(
    page.getByRole("table", { name: "Recorded route summaries", exact: true }),
  ).toContainText("Berlin");
});
