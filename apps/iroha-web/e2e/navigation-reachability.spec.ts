import { test, expect } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";
import { PUBLIC_BASE_URL } from "../playwright.config";
for (const mode of ["light", "dark"] as const) {
  test(`Navigation ${mode} Reports belongs to Analyze and announces current route`, async ({
    page,
  }) => {
    await installPilotFixtures(page, "overview");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    const nav = page.getByRole("navigation", {
      name: "Primary navigation",
      exact: true,
    });
    await nav.getByText("Analyze", { exact: true }).click();
    const reports = nav.getByRole("link", { name: /Reports/ });
    await expect(reports).toBeVisible();
    await expect(reports).toHaveAttribute("aria-current", "page");
    await expect(reports).toHaveAttribute("href", "/reports");
    await reports.focus();
    await expect(reports).toBeFocused();
  });
  test(`Public ${mode} compact exact columns and sort controls are keyboard reachable`, async ({
    page,
  }) => {
    const fixture = await installPilotFixtures(page, "public");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto(PUBLIC_BASE_URL);
    const table = page.getByRole("table", {
      name: "Public activity records",
      exact: true,
    });
    await expect(table).toBeVisible();
    const region = page.getByRole("region", {
      name: "Public activity records — horizontally scrollable",
      exact: true,
    });
    const focusScroll = page.getByRole("button", {
      name: "Focus public records to scroll all columns",
      exact: true,
    });
    await focusScroll.focus();
    await page.keyboard.press("Enter");
    await expect(region).toBeFocused();
    const beforeScroll = await region.evaluate((element) => element.scrollLeft);
    await page.keyboard.press("ArrowRight");
    await expect
      .poll(() => region.evaluate((element) => element.scrollLeft))
      .toBeGreaterThan(beforeScroll);
    const sorts = table.locator("thead button");
    await expect(sorts).toHaveCount(4);
    for (const button of await sorts.all()) {
      await button.focus();
      await expect(button).toBeFocused();
      const rect = await button.evaluate((element) => {
        const r = element.getBoundingClientRect();
        return { x: r.x, right: r.right, height: r.height };
      });
      expect(rect.x).toBeGreaterThanOrEqual(0);
      expect(rect.right).toBeLessThanOrEqual(321);
      expect(rect.height).toBeGreaterThanOrEqual(44);
      await page.keyboard.press("Enter");
    }
    await expect(table.locator("tbody tr")).toHaveCount(1);
    await expect(table).toContainText("Synthetic run 2026");
    await page
      .getByRole("navigation", { name: "Select year", exact: true })
      .getByRole("button", { name: "2025", exact: true })
      .click();
    await expect(table.locator("tbody tr")).toHaveCount(1);
    await expect(table).toContainText("Synthetic run 2025");
    expect(fixture.unknown).toEqual([]);
    expect(
      fixture.requests.some((request) => request.startsWith("/api/v1/")),
    ).toBe(false);
  });
}
