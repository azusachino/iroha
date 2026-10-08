import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

for (const mode of ["light", "dark"] as const) {
  test(`public ${mode} shows truthful feedback while initial data is deferred`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "public");
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/public/v1/**", async (route) => {
      await gate;
      await route.fallback();
    });
    try {
      await page.goto(PUBLIC_BASE_URL);
      await expect(page.getByRole("status")).toContainText(
        "Loading public activity data…",
      );
      await expect(
        page.getByRole("heading", { name: "Activities", exact: true }),
      ).toHaveCount(0);
    } finally {
      release();
    }
    await expect(
      page.getByRole("heading", { name: "Activities", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Loading public activity data…", { exact: true }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
    expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
      [],
    );
  });
  test(`public ${mode} monthly chart honors reduced motion`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "public");
    await page.goto(PUBLIC_BASE_URL);
    await expect(
      page.getByRole("img", { name: "Monthly distance for 2026", exact: true }),
    ).toBeVisible();
    const readMotion = () =>
      page.evaluate(inspectPilotCharts).then((snapshot) => {
        expect(snapshot.verified).toBe(true);
        const chart = snapshot.verified
          ? snapshot.charts.find(
              (chart) => chart.label === "Monthly distance for 2026",
            )
          : undefined;
        return {
          animation: chart?.animation,
          duration: chart?.animationDuration,
        };
      });
    await expect.poll(readMotion).toEqual({ animation: false, duration: 0 });
    await page.emulateMedia({ reducedMotion: "no-preference" });
    await expect.poll(readMotion).toEqual({ animation: true, duration: 550 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await expect.poll(readMotion).toEqual({ animation: false, duration: 0 });
    for (const theme of [mode === "light" ? "dark" : "light", mode]) {
      await page.evaluate((theme) => {
        document.documentElement.dataset.theme = theme;
      }, theme);
      const paint = await page.evaluate(() =>
        getComputedStyle(document.documentElement)
          .getPropertyValue("--accent")
          .trim(),
      );
      await expect
        .poll(async () => {
          const snapshot = await page.evaluate(inspectPilotCharts);
          return snapshot.verified
            ? snapshot.charts.find(
                (chart) => chart.label === "Monthly distance for 2026",
              )?.series[0].color
            : undefined;
        })
        .toBe(paint);
    }
    expect(fixture.unknown).toEqual([]);
    expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
      [],
    );
  });

  test(`public ${mode} compact error shell retries sanitized data by keyboard`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const failed = await installPilotFixtures(page, "public", "error");
    await page.goto(PUBLIC_BASE_URL);
    await expect(page.getByRole("main")).toBeVisible();
    await expect(page.getByRole("alert")).toContainText(
      "Public activity data is temporarily unavailable.",
    );
    await page.keyboard.press("Tab");
    const retry = page.getByRole("button", { name: "Try again", exact: true });
    await expect(retry).toBeFocused();
    let failedRetryResponses = 0;
    page.on("response", (response) => {
      if (response.url().includes("/public/v1/") && response.status() === 503)
        failedRetryResponses++;
    });
    await page.keyboard.press("Enter");
    await expect.poll(() => failedRetryResponses).toBeGreaterThan(0);
    await expect(page.getByRole("alert")).toContainText(
      "Public activity data is temporarily unavailable.",
    );
    await expect(retry).toBeVisible();
    await page.unrouteAll({ behavior: "wait" });
    const recovered = await installPilotFixtures(page, "public");
    await retry.focus();
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Activities", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(failed.unknown).toEqual([]);
    expect(recovered.unknown).toEqual([]);
    expect(
      [...failed.requests, ...recovered.requests].filter((path) =>
        path.startsWith("/api/"),
      ),
    ).toEqual([]);
  });
}

// Unknown ?activity= links (issue #116): the archive must say clearly that the
// linked record is not in the public projection, while the archive itself
// stays usable. Pending or failing reads are never "not found".
const notFoundNotice = (page: Parameters<typeof installPilotFixtures>[0]) =>
  page.getByRole("status", { name: "Activity not found" });

const archiveVisible = async (
  page: Parameters<typeof installPilotFixtures>[0],
) => {
  await expect(
    page.getByRole("heading", { name: /harus track/i }),
  ).toBeVisible();
  await expect(
    page.getByRole("table", { name: "Public activity records" }),
  ).toBeVisible();
};

test("unknown activity id shows a clear not-found notice and keeps the archive usable", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "public");
  // The notice must be confirmed by an actual GET 404 for the selected id,
  // not by a fixture's default unknown sentinel, so intercept the exact
  // target explicitly and keep base-fixture unknowns empty.
  const targetPath = "/public/v1/activities/does-not-exist";
  let targetReads = 0;
  await page.route(`**${targetPath}`, async (route) => {
    const request = route.request();
    targetReads += 1;
    expect(request.method()).toBe("GET");
    expect(new URL(request.url()).pathname).toBe(targetPath);
    fixture.requests.push(targetPath);
    await route.fulfill({
      status: 404,
      json: { error: "activity not found in public archive" },
    });
  });
  const targetGone = page.waitForResponse(
    (response) =>
      response.url().endsWith(targetPath) && response.status() === 404,
  );
  await page.goto(`${PUBLIC_BASE_URL}/?activity=does-not-exist`);
  await targetGone;
  const targetPayload = await (await targetGone).json();
  expect(targetPayload).toEqual({
    error: "activity not found in public archive",
  });
  await expect(notFoundNotice(page)).toContainText("not in the public archive");
  await expect(
    notFoundNotice(page).getByRole("link", { name: "View the full archive" }),
  ).toBeVisible();
  // The archive fallback stays fully usable next to the notice.
  await archiveVisible(page);
  await expect(page.getByText("Synthetic run 2026").first()).toBeVisible();
  // Exactly one detail read happened, and it got the explicit GET 404.
  await expect.poll(() => targetReads).toBe(1);
  expect(
    fixture.requests.filter((path) =>
      path.startsWith("/public/v1/activities/"),
    ),
  ).toEqual([targetPath]);
  expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
    [],
  );
  expect(fixture.unknown).toEqual([]);
  // Selecting a known id from the archive clears the confirmed not-found
  // instead of carrying the stale 404 into the new selection.
  await page.locator("a.activity-link").first().click();
  await expect(
    page.getByRole("heading", { name: "Synthetic run 2026" }),
  ).toBeVisible();
  await expect(notFoundNotice(page)).toHaveCount(0);
  await expect.poll(() => targetReads).toBe(1);
});

test("unknown id with pending or failing detail read is not reported as not found", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "public");
  let releaseTarget!: () => void;
  const gate = new Promise<void>((resolve) => {
    releaseTarget = resolve;
  });
  let heldTargetReads = 0;
  const targetPath = "/public/v1/activities/does-not-exist";
  await page.route(`**${targetPath}`, async (route) => {
    heldTargetReads += 1;
    await gate;
    await route.fulfill({ status: 503, json: { error: "synthetic failure" } });
  });
  const targetHeld = page.waitForRequest(
    (request) =>
      request.method() === "GET" &&
      new URL(request.url()).pathname === targetPath,
  );
  await page.goto(`${PUBLIC_BASE_URL}/?activity=does-not-exist`);
  await targetHeld;
  // List absence alone is not proof: while the detail read for the unknown
  // id is still pending, no not-found notice may be shown.
  await expect.poll(() => heldTargetReads).toBe(1);
  await expect(notFoundNotice(page)).toHaveCount(0);
  await expect(page.getByText("not in the public archive")).toHaveCount(0);
  releaseTarget();
  const targetFailed = page.waitForResponse(
    (response) =>
      response.url().endsWith(targetPath) && response.status() === 503,
  );
  await targetFailed;
  // A 503 for the unknown id is a failure, not a confirmed not-found; the
  // archive fallback stays usable.
  await expect(notFoundNotice(page)).toHaveCount(0);
  await expect(page.getByText("not in the public archive")).toHaveCount(0);
  await archiveVisible(page);
  expect(fixture.requests.filter((path) => path.startsWith("/api/"))).toEqual(
    [],
  );
  expect(fixture.unknown).toEqual([]);
});

test("known activity id keeps the detail view with no false not-found notice", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.goto(`${PUBLIC_BASE_URL}/?activity=synthetic-2026`);
  await expect(
    page.getByRole("heading", { name: "Synthetic run 2026" }),
  ).toBeVisible();
  await expect(notFoundNotice(page)).toHaveCount(0);
  await page.getByRole("link", { name: "Back to archive" }).click();
  await archiveVisible(page);
  await expect(notFoundNotice(page)).toHaveCount(0);
});

test("pending or failing reads are never reported as not found", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  let releaseDetail!: () => void;
  const gate = new Promise<void>((resolve) => {
    releaseDetail = resolve;
  });
  let heldDetailReads = 0;
  await page.route("**/public/v1/activities/synthetic-2026", async (route) => {
    heldDetailReads += 1;
    await gate;
    await route.fulfill({ status: 503, json: { error: "synthetic failure" } });
  });
  await page.goto(`${PUBLIC_BASE_URL}/?activity=synthetic-2026`);
  // While the detail read is still pending, the known record must not be
  // called missing.
  await expect(
    page.getByRole("heading", { name: "Synthetic run 2026" }),
  ).toBeVisible();
  await expect(notFoundNotice(page)).toHaveCount(0);
  await expect(
    page
      .getByRole("status", { name: "Activity not found" })
      .or(page.getByText("not in the public archive")),
  ).toHaveCount(0);
  releaseDetail();
  await expect.poll(() => heldDetailReads).toBe(1);
  // A 503 detail response is a failure, not a confirmed not-found.
  await expect(notFoundNotice(page)).toHaveCount(0);
  // A failing list read surfaces the site-level retry shell instead.
  await page.unrouteAll({ behavior: "wait" });
  await installPilotFixtures(page, "public", "error");
  await page.goto(PUBLIC_BASE_URL);
  await expect(page.getByRole("alert")).toContainText(
    "temporarily unavailable",
  );
  await expect(page.getByText("not in the public archive")).toHaveCount(0);
});
