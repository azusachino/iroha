import { test, expect } from "@playwright/test";
import { installMotionFixtures } from "./motion-fixtures";
import { installNightFixtures } from "./night-fixtures";
for (const mode of ["light", "dark"] as const) {
  test(`Night ${mode} exact records are named and keyboard scrollable`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    const region = page.getByRole("region", {
      name: "Night exact sleep records",
      exact: true,
    });
    await expect(region).toBeVisible();
    await expect(region).toHaveAttribute("tabindex", "0");
    await region.focus();
    await expect(region).toBeFocused();
    const before = await region.evaluate((node) => node.scrollLeft);
    await page.keyboard.press("ArrowRight");
    await expect
      .poll(() => region.evaluate((node) => node.scrollLeft))
      .toBeGreaterThan(before);
    await expect(page).toHaveURL(/date=2026-08/);
    await page.keyboard.press("End");
    const endScroll = await region.evaluate((node) => node.scrollLeft);
    expect(endScroll).toBeGreaterThan(before);
    await expect(region.getByRole("table")).toBeVisible();
    await expect(region.getByRole("table").locator("thead th")).toHaveCount(5);
    expect(fixture.unknown).toEqual([]);
  });
  test(`Motion ${mode} exact overflow tables are named and keyboard scrollable`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    for (const [regionLabel, columnCount] of [
      ["Motion exact activity records", 5],
      ["Exact movement series scroll area", 4],
    ] as const) {
      const region = page.getByRole("region", {
        name: regionLabel,
        exact: true,
      });
      await expect(region).toBeVisible();
      await expect(region).toHaveAttribute("tabindex", "0");
      await region.focus();
      await expect(region).toBeFocused();
      expect(
        await region.evaluate((node) => node.scrollWidth > node.clientWidth),
      ).toBe(true);
      const before = await region.evaluate((node) => node.scrollLeft);
      await page.keyboard.press("ArrowRight");
      await expect
        .poll(() => region.evaluate((node) => node.scrollLeft))
        .toBeGreaterThan(before);
      await expect(page).toHaveURL(/date=2026-08/);
      await page.keyboard.press("End");
      const table = region.getByRole("table");
      await expect(table).toBeVisible();
      await expect(table.locator("thead th")).toHaveCount(columnCount);
    }
    expect(fixture.unknown).toEqual([]);
  });
}
