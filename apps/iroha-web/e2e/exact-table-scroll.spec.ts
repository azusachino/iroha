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
    const focus = page.getByRole("button", {
      name: "Focus sleep records to scroll all columns",
      exact: true,
    });
    await expect(focus).toBeVisible();
    const bounds = await focus.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.height).toBeGreaterThanOrEqual(44);
    await focus.focus();
    await page.keyboard.press("Enter");
    const region = page.getByRole("region", {
      name: "Night exact sleep records",
      exact: true,
    });
    await expect(region).toBeFocused();
    const before = await region.evaluate((node) => node.scrollLeft);
    await page.keyboard.press("ArrowRight");
    await expect
      .poll(() => region.evaluate((node) => node.scrollLeft))
      .toBeGreaterThan(before);
    await expect(page).toHaveURL(/date=2026-08/);
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
    for (const [buttonLabel, regionLabel] of [
      [
        "Focus activity records to scroll all columns",
        "Motion exact activity records",
      ],
      [
        "Focus movement series to scroll all columns",
        "Exact movement series scroll area",
      ],
    ]) {
      const button = page.getByRole("button", {
        name: buttonLabel,
        exact: true,
      });
      await expect(button).toBeVisible();
      const bounds = await button.boundingBox();
      expect(bounds).not.toBeNull();
      expect(bounds!.height).toBeGreaterThanOrEqual(44);
      await button.focus();
      await page.keyboard.press("Enter");
      const region = page.getByRole("region", {
        name: regionLabel,
        exact: true,
      });
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
      await expect(table.locator("thead th")).toHaveCount(
        buttonLabel.includes("activity records") ? 5 : 4,
      );
    }
    expect(fixture.unknown).toEqual([]);
  });
}
