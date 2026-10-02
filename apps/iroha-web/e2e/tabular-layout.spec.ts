import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  for (const width of [320, 1280]) {
    test(`public ${mode} ${width} numeric scope changes retain surrounding tile layout`, async ({
      page,
    }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      await installPilotFixtures(page, "public");
      await page.goto(PUBLIC_BASE_URL);
      await expect(
        page.getByText("12.50 km", { exact: true }).first(),
      ).toBeVisible();
      const readLayout = () =>
        page.evaluate(() =>
          Array.from(document.querySelectorAll(".stat-grid .stat-tile")).map(
            (el) => {
              const rect = el.getBoundingClientRect();
              return {
                x: rect.x + scrollX,
                y: rect.y + scrollY,
                width: rect.width,
                height: rect.height,
              };
            },
          ),
        );
      const before = await readLayout();
      await page.getByRole("button", { name: "2025", exact: true }).click();
      await expect(
        page.getByText("8.00 km", { exact: true }).first(),
      ).toBeVisible();
      expect(await readLayout()).toEqual(before);
      const figures = await page.evaluate(() => {
        const value = document.querySelector<HTMLElement>(".stat-value")!;
        const probe = value.cloneNode(false) as HTMLElement;
        // Probe the rendered font's digit advances without mutating application data.
        probe.style.position = "absolute";
        probe.style.width = "max-content";
        value.parentElement!.append(probe);
        probe.textContent = "11111";
        const narrow = probe.getBoundingClientRect().width;
        probe.textContent = "88888";
        const wide = probe.getBoundingClientRect().width;
        const variant = getComputedStyle(probe).fontVariantNumeric;
        probe.remove();
        return { narrow, wide, variant };
      });
      expect(figures.variant).toContain("tabular-nums");
      expect(figures.narrow).toBe(figures.wide);
    });
  }
}
