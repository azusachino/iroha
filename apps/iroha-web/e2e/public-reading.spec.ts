import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

test("public reading section renders the sanitized book list", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.goto(PUBLIC_BASE_URL);

  const heading = page.getByRole("heading", { name: "Reading", exact: true });
  await expect(heading).toBeVisible();
  // The reading list loads lazily: bring its section into the viewport.
  await heading.scrollIntoViewIfNeeded();

  const section = page.getByRole("region", { name: "Reading list" });
  await expect(section).toBeVisible();

  // Grouped by year, most recent year first: 2026 before 2024.
  const years = section.locator(".book-year");
  await expect(years).toHaveCount(2);
  await expect(years.nth(0)).toHaveText("2026");
  await expect(years.nth(1)).toHaveText("2024");

  const finished = section.locator(".book-row", {
    hasText: "克拉拉与太阳",
  });
  await expect(finished).toContainText("石黑一雄");
  await expect(finished.locator(".book-status")).toHaveText("Finished");

  const reading = section.locator(".book-row", {
    hasText: "如何了解一个人",
  });
  await expect(reading.locator(".book-status")).toHaveText("Reading");
  await expect(reading.locator(".book-date")).toContainText("2026");
});

test("public reading section quiets itself when the part is unavailable", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({ status: 503, json: { error: "synthetic failure" } }),
  );
  await page.goto(PUBLIC_BASE_URL);

  await expect(
    page.getByRole("heading", { name: "Reading", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("heading", { name: "Reading", exact: true })
    .scrollIntoViewIfNeeded();
  await expect(
    page.getByText("Reading list is temporarily unavailable."),
  ).toBeVisible();
  // The rest of the public site is unaffected.
  await expect(
    page.getByRole("heading", { name: "Activities", exact: true }),
  ).toBeVisible();
});

test("public reading section stays empty-safe", async ({ page }) => {
  await installPilotFixtures(page, "public", "empty");
  await page.goto(PUBLIC_BASE_URL);

  await expect(
    page.getByRole("heading", { name: "Reading", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("heading", { name: "Reading", exact: true })
    .scrollIntoViewIfNeeded();
  await expect(page.getByText("No reading records yet.")).toBeVisible();
});
