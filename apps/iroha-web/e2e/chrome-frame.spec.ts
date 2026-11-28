import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { installPilotFixtures } from "./pilot-fixtures";

test("Expenses metric and canonical frames preserve exact data and distinct capabilities", async ({
  page,
}) => {
  const fixture = await installPilotFixtures(page, "expenses");
  await page.goto("/expenses?date=2026-08&currency=JPY");
  await expect(
    page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
  ).toBeVisible();
  const spending = page.getByRole("region", {
    name: "Expense spending",
    exact: true,
  });
  const metric = page.getByRole("region", { name: "Daily spend", exact: true });
  await expect(metric).toHaveCount(1);
  await expect(spending).toContainText("synthetic.v1");
  await spending.getByRole("button", { name: "Table", exact: true }).click();
  await expect(
    metric.getByRole("cell", { name: "¥12,345", exact: true }),
  ).toBeVisible();
  const downloading = page.waitForEvent("download");
  await spending.getByRole("button", { name: "CSV", exact: true }).click();
  const download = await downloading;
  expect(download.suggestedFilename()).toBe(
    "expenses.amount_minor-2026-08.csv",
  );
  const csv = await readFile((await download.path())!, "utf8");
  expect(csv).toContain("12345");
  expect(csv).toContain("JPY minor");
  await spending.getByRole("button", { name: "Chart", exact: true }).click();
  await expect(spending.getByRole("img")).toBeVisible();
  const ledger = page.getByRole("region", {
    name: "Expense records",
    exact: true,
  });
  await expect(ledger).toHaveCount(1);
  await expect(
    ledger.getByRole("button", { name: "Export CSV", exact: true }),
  ).toBeVisible();
  await expect(ledger.getByRole("group", { name: /view$/ })).toHaveCount(0);
  expect(fixture.unknown).toEqual([]);
});
