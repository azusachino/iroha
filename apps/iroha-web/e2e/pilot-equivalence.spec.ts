import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`private and public ${mode} preserve equivalent record quantities`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode });
    const privateFixture = await installPilotFixtures(page, "overview");
    await page.goto("/overview");
    const privateRecord = page.getByRole("row").filter({
      has: page.getByRole("button", {
        name: "Synthetic run 2026",
        exact: true,
      }),
    });
    await expect(privateRecord).toContainText("12.50 km");
    await expect(privateRecord).toContainText("1:02:00");
    const publicPage = await page.context().newPage();
    await publicPage.emulateMedia({ colorScheme: mode });
    const publicFixture = await installPilotFixtures(publicPage, "public");
    try {
      await publicPage.goto(PUBLIC_BASE_URL);
      const publicRecord = publicPage
        .getByRole("row")
        .filter({ hasText: "Synthetic run 2026" });
      await expect(publicRecord).toContainText("12.50 km");
      // Distance-sport rows show pace; duration is the scoped summary quantity.
      await expect(publicRecord).toContainText("1:00 /km");
      await expect(
        publicPage.getByText("1 h 02 min", { exact: true }),
      ).toBeVisible();
      await expect(
        publicPage.getByText("Synthetic owner", { exact: true }),
      ).toHaveCount(0);
      expect(
        publicFixture.requests.filter((request) => request.startsWith("/api/")),
      ).toEqual([]);
      expect(publicFixture.unknown).toEqual([]);
      expect(privateFixture.unknown).toEqual([]);
    } finally {
      await publicPage.close();
    }
  });
}
