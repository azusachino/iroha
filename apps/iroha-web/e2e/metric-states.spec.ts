import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Metrics ${mode} labels deferred first load without invented values`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "metrics");
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/api/v1/metrics", async (route) => {
      await gate;
      await route.fallback();
    });
    try {
      await page.goto("/metrics?date=2026-08");
      await expect(
        page.getByRole("status").filter({ hasText: "Loading metrics…" }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: "Steps", exact: true }),
      ).toHaveCount(0);
    } finally {
      release();
    }
    await expect(
      page.getByRole("heading", { name: "Steps", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Metrics ${mode} retries failed series without discarding prior observations`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "metrics");
    await page.goto("/metrics?date=2026-08");
    const chart = page.getByRole("group", { name: "Steps view", exact: true });
    await expect(chart).toBeVisible();
    let failures = 2;
    await page.route("**/api/v1/metrics/*/series?**", async (route) => {
      if (failures-- > 0)
        await route.fulfill({
          status: 503,
          json: { error: "synthetic failure" },
        });
      else await route.fallback();
    });
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    for (let attempt = 0; attempt < 2; attempt++) {
      await expect(page.getByRole("alert")).toBeVisible();
      await expect(chart).toBeVisible();
      const retry = page.getByRole("button", {
        name: "Try again",
        exact: true,
      });
      await retry.focus();
      await page.keyboard.press("Enter");
    }
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(chart).toBeVisible();
    await expect(page).toHaveURL(/date=2026-09/);
    expect(fixture.unknown).toEqual([]);
  });
  test(`Metrics ${mode} retains data during a deferred refetch`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "metrics");
    await page.goto("/metrics?date=2026-08");
    const chart = page.getByRole("group", { name: "Steps view", exact: true });
    await expect(chart).toBeVisible();
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/api/v1/metrics/*/series?**", async (route) => {
      await gate;
      await route.fallback();
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByRole("status").filter({ hasText: "Updating…" }),
      ).toBeVisible();
      await expect(chart).toBeVisible();
      await expect(page.getByText(/Observed window:/)).toContainText(
        "2025-09-01–2026-09-01",
      );
      await chart.getByRole("button", { name: "Table", exact: true }).focus();
      await expect(
        chart.getByRole("button", { name: "Table", exact: true }),
      ).toBeFocused();
    } finally {
      release();
    }
    await expect(page.getByText("Updating…", { exact: true })).toHaveCount(0);
    await expect(page.getByText(/Observed window:/)).toContainText(
      "2025-10-01–2026-10-01",
    );
    await expect(page).toHaveURL(/date=2026-09/);
    await expect(chart).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Metrics ${mode} recovers from repeated catalog failures by keyboard`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "metrics");
    let failures = 2;
    await page.route("**/api/v1/metrics", async (route) => {
      if (failures-- > 0)
        await route.fulfill({
          status: 503,
          json: { error: "synthetic failure" },
        });
      else await route.fallback();
    });
    await page.goto("/metrics?date=2026-08");
    const retry = page.getByRole("button", { name: "Try again", exact: true });
    for (let attempt = 0; attempt < 2; attempt++) {
      await expect(page.getByRole("alert")).toBeVisible();
      await retry.focus();
      await page.keyboard.press("Enter");
    }
    await expect(
      page.getByRole("heading", { name: "Steps", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });
}
