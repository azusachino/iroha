import { test, expect } from "@playwright/test";
import { installNightFixtures } from "./night-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Night ${mode} category navigation reach announces current route`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");

    // "Domains" group in Primary navigation contains Night with aria-current="page"
    const nav = page.getByRole("navigation", {
      name: "Primary navigation",
      exact: true,
    });
    await expect(nav).toBeVisible();
    const domainsTrigger = nav.getByText("Domains", { exact: true });
    await domainsTrigger.click();

    const nightLink = nav.getByRole("link", { name: /^Night/ });
    await expect(nightLink).toBeVisible();
    await expect(nightLink).toHaveAttribute("aria-current", "page");
    await expect(nightLink).toHaveAttribute("href", "/night");
    await nightLink.focus();
    await expect(nightLink).toBeFocused();

    await nightLink.click();
    await expect(page).toHaveURL(/\/night/);
    await expect(nightLink).toHaveAttribute("aria-current", "page");

    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} exact records table keyboard scroll reaches end column`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");

    const focusButton = page.getByRole("button", {
      name: "Focus sleep records to scroll all columns",
      exact: true,
    });
    await expect(focusButton).toBeVisible();
    const bounds = await focusButton.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.height).toBeGreaterThanOrEqual(44);

    await focusButton.focus();
    await page.keyboard.press("Enter");

    const scrollRegion = page.getByRole("region", {
      name: "Night exact sleep records",
      exact: true,
    });
    await expect(scrollRegion).toBeFocused();

    // Verify container overflows at 320px
    const isOverflowing = await scrollRegion.evaluate(
      (node) => node.scrollWidth > node.clientWidth,
    );
    expect(isOverflowing).toBe(true);

    const initialScroll = await scrollRegion.evaluate(
      (node) => node.scrollLeft,
    );
    expect(initialScroll).toBe(0);

    // ArrowRight advances scroll
    await page.keyboard.press("ArrowRight");
    await expect
      .poll(() => scrollRegion.evaluate((node) => node.scrollLeft))
      .toBeGreaterThan(initialScroll);

    const scrolled = await scrollRegion.evaluate((node) => node.scrollLeft);

    // End key scrolls to rightmost end column
    await page.keyboard.press("End");
    const endScroll = await scrollRegion.evaluate((node) => node.scrollLeft);
    expect(endScroll).toBeGreaterThanOrEqual(scrolled);
    expect(endScroll).toBeGreaterThan(initialScroll);

    const table = scrollRegion.getByRole("table", {
      name: "Sleep session records",
      exact: true,
    });
    await expect(table).toBeVisible();
    const headers = table.locator("thead th");
    await expect(headers).toHaveCount(5);
    await expect(headers.nth(4)).toHaveText("Type");

    // ArrowLeft scrolls backward
    await page.keyboard.press("ArrowLeft");
    await expect
      .poll(() => scrollRegion.evaluate((node) => node.scrollLeft))
      .toBeLessThan(endScroll);

    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} table row keyboard traversal, detail reach, and backlink return`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);

    const detailSessionId = "synthetic-2026-08";
    await page.route(`**/api/v1/sleep/${detailSessionId}`, (route) =>
      route.fulfill({
        json: {
          id: detailSessionId,
          wake_date: "2026-08-14",
          started_at: "2026-08-13T22:00:00Z",
          ended_at: "2026-08-14T06:00:00Z",
          asleep_s: 25200,
          time_in_bed_s: 28800,
          efficiency: 0.875,
          deep_s: 3600,
          rem_s: 3600,
          is_main_sleep: true,
          source: "synthetic",
        },
      }),
    );
    await page.route(`**/api/v1/sleep/${detailSessionId}/segments`, (route) =>
      route.fulfill({ json: [] }),
    );

    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");

    const table = page.getByRole("table", {
      name: "Sleep session records",
      exact: true,
    });
    await expect(table).toBeVisible();

    const rows = table.locator("tbody tr[role=link]");
    await expect(rows).toHaveCount(1);

    // Row keyboard focus traversal
    const firstRow = rows.first();
    await firstRow.focus();
    await expect(firstRow).toBeFocused();

    // Press Enter on the focused row to navigate to detail
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(new RegExp(`/night/${detailSessionId}`));

    // Verify detail page elements
    await expect(
      page.getByRole("heading", { name: "2026-08-14", exact: true }),
    ).toBeVisible();
    await expect(
      page
        .getByText("This session has no stage samples.", { exact: true })
        .first(),
    ).toBeVisible();

    // Verify backlink returns to Night
    const backLink = page.getByRole("link", { name: /Back to Night/ });
    await expect(backLink).toBeVisible();
    await backLink.click();
    await expect(page).toHaveURL(/\/night/);
    await expect(table).toBeVisible();
  });

  test(`Night ${mode} paginated load more traversal and detail state reach`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.paginated = true;

    const detailNextId = "synthetic-2026-08-next";
    await page.route(`**/api/v1/sleep/${detailNextId}`, (route) =>
      route.fulfill({
        json: {
          id: detailNextId,
          wake_date: "2026-08-15",
          started_at: "2026-08-14T22:00:00Z",
          ended_at: "2026-08-15T06:00:00Z",
          asleep_s: 25200,
          time_in_bed_s: 28800,
          efficiency: 0.875,
          deep_s: 3600,
          rem_s: 3600,
          is_main_sleep: true,
          source: "synthetic",
        },
      }),
    );
    await page.route(`**/api/v1/sleep/${detailNextId}/segments`, (route) =>
      route.fulfill({ json: [] }),
    );

    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");

    const table = page.getByRole("table", {
      name: "Sleep session records",
      exact: true,
    });
    // First page shows 2026-08-15
    await expect(
      table.getByRole("cell", { name: "2026-08-15", exact: true }),
    ).toBeVisible();
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toHaveCount(0);

    // Sentinel scroll triggers cursor fetch
    await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
    await expect
      .poll(() =>
        fixture.requests.some((request) => request.includes("cursor=")),
      )
      .toBe(true);

    // Second page row appears
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toBeVisible();
    await expect(
      table.getByRole("cell", { name: "2026-08-15", exact: true }),
    ).toBeVisible();

    // Navigate to the newly paged row's detail
    const row2 = table.locator("tbody tr[role=link]").last();
    await row2.click();
    await expect(page).toHaveURL(new RegExp(`/night/`));
  });
}
