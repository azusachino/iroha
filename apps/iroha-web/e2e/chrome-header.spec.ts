import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";

test("Expenses has one compact header with truthful loaded period and reachable controls", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await installPilotFixtures(page, "expenses");
  await page.goto("/expenses?date=2026-08");
  await expect(
    page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
  ).toBeVisible();
  const header = page.getByRole("region", {
    name: "Expenses header",
    exact: true,
  });
  await expect(header).toBeVisible();
  await expect(header.getByRole("heading", { level: 1 })).toHaveText(
    "Expenses",
  );
  await expect(
    header.getByText("Observed period: 2026-08", { exact: true }),
  ).toBeVisible();
  await expect(
    header.getByRole("combobox", { name: "Filter by year" }),
  ).toHaveCount(1);
  await expect(
    header.getByRole("combobox", { name: "Filter by month" }),
  ).toHaveCount(1);
  await expect(
    header.getByRole("button", { name: "Refresh", exact: true }),
  ).toBeVisible();
  const box = await header.boundingBox();
  expect(box!.height).toBeLessThanOrEqual(128);
  let release!: () => void;
  const deferred = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/api/v1/expenses?**", async (route) => {
    await deferred;
    await route.fallback();
  });
  try {
    await header
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("2026-09");
    await expect(
      header.getByText("Observed period: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("status").filter({ hasText: "Updating…" }),
    ).toBeVisible();
  } finally {
    release();
  }
  await expect(
    header.getByText("Observed period: 2026-09", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Synthetic shop", exact: true }),
  ).toBeVisible();
});
