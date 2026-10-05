import { test, expect } from "@playwright/test";
import { fakeSession } from "./session";
for (const mode of ["light", "dark"] as const) {
  for (const width of [320, 375]) {
    test(`Admin ${mode} ${width} selected tabs and upload input remain reachable`, async ({
      page,
    }) => {
      const pageErrors: string[] = [];
      const writes: string[] = [];
      page.on("request", (request) => {
        if (!["GET", "HEAD", "OPTIONS"].includes(request.method()))
          writes.push(request.method() + " " + request.url());
      });
      page.on("pageerror", (error) => pageErrors.push(error.message));
      await page.route("**/api/v1/**", (route) =>
        route.fulfill({
          status: 503,
          json: { error: "Synthetic unavailable Admin read" },
        }),
      );
      await fakeSession(page, { setup_required: false, authenticated: true });
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: mode });
      for (const [id, label] of [
        ["system", "System"],
        ["sources", "Sources"],
        ["jobs", "Jobs"],
        ["imports", "Imports"],
        ["intake", "Intake tokens"],
      ]) {
        await page.goto(`/admin?tab=${id}`);
        const tab = page.getByRole("tab", { name: label, exact: true });
        await expect(tab).toHaveAttribute("aria-selected", "true");
        const bounds = await tab.boundingBox();
        expect(bounds).not.toBeNull();
        expect(bounds!.x).toBeGreaterThanOrEqual(0);
        expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width);
        expect(bounds!.height).toBeGreaterThanOrEqual(24);
        await tab.focus();
        await expect(tab).toBeFocused();
        await page.keyboard.press("ArrowRight");
        const selected = page.getByRole("tab", { selected: true });
        await expect(selected).toBeFocused();
        const nextBounds = await selected.boundingBox();
        expect(nextBounds).not.toBeNull();
        expect(nextBounds!.x).toBeGreaterThanOrEqual(0);
        expect(nextBounds!.x + nextBounds!.width).toBeLessThanOrEqual(width);
      }
      await page.goto("/admin?tab=imports");
      const file = page.getByLabel("File", { exact: true });
      await expect(file).toBeVisible();
      const inputBounds = await file.boundingBox();
      const formBounds = await page.locator("form.upload").boundingBox();
      expect(inputBounds).not.toBeNull();
      expect(formBounds).not.toBeNull();
      expect(inputBounds!.x).toBeGreaterThanOrEqual(formBounds!.x);
      expect(inputBounds!.x + inputBounds!.width).toBeLessThanOrEqual(
        formBounds!.x + formBounds!.width + 1,
      );
      expect(inputBounds!.x + inputBounds!.width).toBeLessThanOrEqual(width);
      await file.setInputFiles({
        name: `synthetic-${"long-evidence-name-".repeat(8)}.gpx`,
        mimeType: "application/gpx+xml",
        buffer: Buffer.from("<gpx version='1.1'></gpx>"),
      });
      expect(
        await file.evaluate(
          (element: HTMLInputElement) => element.files?.length,
        ),
      ).toBe(1);
      const selectedInputBounds = await file.boundingBox();
      const selectedFormBounds = await page
        .locator("form.upload")
        .boundingBox();
      expect(selectedInputBounds).not.toBeNull();
      expect(selectedFormBounds).not.toBeNull();
      expect(
        selectedInputBounds!.x + selectedInputBounds!.width,
      ).toBeLessThanOrEqual(width);
      expect(
        selectedInputBounds!.x + selectedInputBounds!.width,
      ).toBeLessThanOrEqual(
        selectedFormBounds!.x + selectedFormBounds!.width + 1,
      );
      expect(
        selectedFormBounds!.x + selectedFormBounds!.width,
      ).toBeLessThanOrEqual(width);
      await expect(
        page.getByRole("button", { name: "Upload and import", exact: true }),
      ).toBeEnabled();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      for (const selector of [
        ".admin-page",
        ".admin-card[aria-labelledby='upload-title']",
      ]) {
        const bounds = await page.locator(selector).boundingBox();
        expect(bounds).not.toBeNull();
        expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width);
      }
      expect(writes).toEqual([]);
      expect(pageErrors).toEqual([]);
    });
  }
}
