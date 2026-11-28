import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Expenses ${mode} summary geometry stays stable when a loaded amount becomes zero`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    await installPilotFixtures(page, "expenses");
    await page.goto("/expenses?date=2026-08&currency=JPY");
    const band = page.getByRole("region", {
      name: "Spending summary",
      exact: true,
    });
    const spending = page.getByRole("region", {
      name: "Expense spending",
      exact: true,
    });
    await expect(band).toContainText("¥12,345");
    const before = (await band.boundingBox())!;
    const plot = page.getByRole("heading", {
      name: "Daily spend · JPY",
      exact: true,
    });
    const plotBefore = (await plot.boundingBox())!;
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(band.getByText("¥0", { exact: true })).toBeVisible();
    await expect(spending).toContainText("Observed period: 2026-09");
    const after = (await band.boundingBox())!;
    const plotAfter = (await plot.boundingBox())!;
    expect(after.width).toBeCloseTo(before.width, 2);
    expect(after.height).toBeCloseTo(before.height, 2);
    expect(plotAfter.y).toBeCloseTo(plotBefore.y, 2);
  });
}

test("Overview geography plot precedes exact records in DOM and visual order", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await installPilotFixtures(page, "overview");
  await page.goto("/overview");
  const chart = page.getByRole("region", {
    name: "Monthly distance",
    exact: true,
  });
  const records = page
    .getByRole("region", {
      name: "Recent movement",
      exact: true,
    })
    .first();
  await expect(chart).toBeVisible();
  await expect(records).toBeVisible();
  await expect(
    records.getByRole("table", { name: "Recent movement", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Route footprint", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("region", { name: "Last 12 months", exact: true }),
  ).toContainText("No routes recorded yet");
  const chartBox = (await chart.boundingBox())!;
  const recordBox = (await records.boundingBox())!;
  expect(chartBox.y).toBeLessThanOrEqual(recordBox.y);
  if (chartBox.y === recordBox.y) expect(chartBox.x).toBeLessThan(recordBox.x);
  expect(
    await records.evaluate((element) => {
      const chart = Array.from(document.querySelectorAll(".panel-frame")).find(
        (section) => section.textContent?.includes("Monthly distance"),
      )!;
      return Boolean(
        chart.compareDocumentPosition(element) &
        Node.DOCUMENT_POSITION_FOLLOWING,
      );
    }),
  ).toBe(true);
});

test("Overview shared summaries name mixed windows and keep existing links and independent missing states", async ({
  page,
}) => {
  await installPilotFixtures(page, "overview");
  await page.goto("/overview");
  const band = page.getByRole("region", {
    name: "Last 12 months",
    exact: true,
  });
  await expect(band).toContainText("12.50 km");
  await expect(
    band.getByRole("link", { name: "Explore movement →" }),
  ).toHaveAttribute("href", "/motion");
  await expect(band).toContainText(
    "Sleep has no records · Library has no records · No routes recorded yet",
  );
  const archive = page.getByRole("region", {
    name: "Across the archive",
    exact: true,
  });
  await expect(
    archive.getByRole("link", { name: /Activity sessions/ }),
  ).toHaveAttribute("href", "/motion");
  await expect(archive).toContainText(
    "No records yet for night sessions or media items.",
  );
});

test("Expenses retained summaries and plot units do not adopt a pending currency", async ({
  page,
}) => {
  await installPilotFixtures(page, "expenses");
  await page.goto("/expenses?date=2026-08&currency=JPY");
  const band = page.getByRole("region", {
    name: "Spending summary",
    exact: true,
  });
  const spending = page.getByRole("region", {
    name: "Expense spending",
    exact: true,
  });
  await expect(band).toContainText("¥12,345");
  await expect(spending).toContainText("Observed period: 2026-08");
  let release!: () => void;
  const deferred = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/api/v1/expenses?**", async (route) => {
    await deferred;
    await route.fallback();
  });
  await page.route("**/api/v1/metrics/*/series?**", async (route) => {
    await deferred;
    await route.fallback();
  });
  try {
    await page
      .getByRole("combobox", { name: "Currency", exact: true })
      .selectOption("USD");
    await expect(
      spending.getByRole("status").filter({ hasText: "Updating…" }),
    ).toBeVisible();
    await expect(band).toContainText("¥12,345");
    await expect(
      page.getByRole("heading", { name: "Daily spend · JPY", exact: true }),
    ).toBeVisible();
    await expect(spending).toContainText("JPY minor");
  } finally {
    release();
  }
  await expect(
    page.getByRole("heading", { name: "Daily spend · USD", exact: true }),
  ).toBeVisible();
});
