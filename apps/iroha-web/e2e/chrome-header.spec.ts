import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";
import { PUBLIC_BASE_URL } from "../playwright.config";

test("Expenses failed initial load does not claim its ledger period is still loading", async ({
  page,
}) => {
  await installPilotFixtures(page, "expenses", "error");
  await page.goto("/expenses?date=2026-08");
  const header = page.getByRole("region", {
    name: "Expenses header",
    exact: true,
  });
  await expect(
    header.getByText("Ledger period unavailable", { exact: true }),
  ).toBeVisible();
  await expect(
    header.getByText("Loading ledger period…", { exact: true }),
  ).toHaveCount(0);
});

for (const pilot of ["overview", "public"] as const) {
  test(`${pilot} has one compact route header and its actual time controls`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await installPilotFixtures(page, pilot);
    await page.goto(pilot === "public" ? PUBLIC_BASE_URL : `/${pilot}`);
    await expect(page.getByRole("img").first()).toBeVisible();
    const name = pilot === "overview" ? "Overview" : "harus track";
    const header = page.getByRole("region", {
      name: `${name} header`,
      exact: true,
    });
    await expect(header).toBeVisible();
    await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
    const box = await header.boundingBox();
    expect(box!.height).toBeLessThanOrEqual(128);
    if (pilot === "overview") {
      await expect(header).toContainText("Mixed windows");
      await expect(header.getByRole("combobox")).toHaveCount(0);
      await expect(
        page.getByRole("group", { name: "Distance chart range" }),
      ).toBeVisible();
    } else {
      await expect(
        header.getByRole("navigation", { name: "Select year" }),
      ).toHaveCount(1);
      await header.getByRole("button", { name: "2025", exact: true }).click();
      await expect(header).toContainText("Observed year: 2025");
      await expect(
        page.getByText("8.00 km", { exact: true }).first(),
      ).toBeVisible();
    }
  });
}

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
