import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`private and public ${mode} preserve equivalent record quantities`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode });
    const privateFixture = await installPilotFixtures(
      page,
      "overview",
      "single-year",
    );
    await page.goto("/overview");
    const privateRecord = page.getByRole("row").filter({
      has: page.getByRole("button", {
        name: "Synthetic run 2026",
        exact: true,
      }),
    });
    await expect(privateRecord).toContainText("12.50 km");
    await expect(privateRecord).toContainText("1:02:00");
    await expect(page.getByText("1 h 02 min", { exact: true })).toBeVisible();
    const publicPage = await page.context().newPage();
    await publicPage.emulateMedia({ colorScheme: mode });
    const publicFixture = await installPilotFixtures(
      publicPage,
      "public",
      "single-year",
    );
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
      const activity = privateFixture.activities[0];
      await page.route("**/api/v1/activities/synthetic-2026", (route) =>
        route.fulfill({
          json: {
            ...activity,
            source_kind: "synthetic",
            first_raw_file_id: "synthetic-raw",
            created_at: activity.started_at,
            updated_at: activity.started_at,
          },
        }),
      );
      for (const suffix of ["route", "samplings", "laps"]) {
        await page.route(
          `**/api/v1/activities/synthetic-2026/${suffix}*`,
          (route) => route.fulfill({ json: [] }),
        );
      }
      await privateRecord
        .getByRole("button", { name: "Synthetic run 2026", exact: true })
        .click();
      await expect(
        page.locator(".grapher-detail").getByText("1:00 /km", { exact: true }),
      ).toBeVisible();
      await expect(
        page
          .locator(".grapher-detail")
          .getByText("1:02:00", { exact: true })
          .first(),
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
