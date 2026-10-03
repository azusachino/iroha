import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";

test("public chart startup completes without partial ECharts models", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await installPilotFixtures(page, "public");
  await page.goto(PUBLIC_BASE_URL);
  await expect(
    page.getByRole("heading", { name: "Activities", exact: true }),
  ).toBeVisible();
  expect(errors).toEqual([]);
  await expect(
    page.getByText("Loading public activity data…", { exact: true }),
  ).toHaveCount(0);
  const charts = await page.evaluate(inspectPilotCharts);
  expect(charts.verified).toBe(true);
  if (charts.verified) {
    expect(charts.charts).toHaveLength(2);
    expect(
      charts.charts.every((chart) =>
        chart.series.every((series) => typeof series.color === "string"),
      ),
    ).toBe(true);
  }
});
