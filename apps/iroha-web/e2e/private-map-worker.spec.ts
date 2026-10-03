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
function isWorker(url: string) {
  const parsed = new URL(url);
  return (
    parsed.pathname.includes("maplibre-gl-worker") &&
    (parsed.searchParams.has("worker_file") ||
      parsed.pathname.includes("/workers/"))
  );
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
