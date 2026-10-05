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
    const tile = page.getByRole("region", {
      name: "Spending · JPY",
      exact: true,
    });
    await expect(tile).toContainText("¥12,345");
    const before = (await tile.boundingBox())!;
    const plot = page.getByRole("heading", {
      name: "Daily spend · JPY",
      exact: true,
    });
    const plotBefore = (await plot.boundingBox())!;
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(tile.getByText("¥0", { exact: true })).toBeVisible();
    await expect(tile).toContainText("Observed period: 2026-09");
    const after = (await tile.boundingBox())!;
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
  const geography = page.getByRole("region", {
    name: "Route footprint",
    exact: true,
  });
  const records = page.getByRole("region", {
    name: "Recent movement",
    exact: true,
  });
  await expect(geography).toBeVisible();
  await expect(records).toBeVisible();
  const geoBox = (await geography.boundingBox())!;
  const recordBox = (await records.boundingBox())!;
  expect(geoBox.y).toBeLessThanOrEqual(recordBox.y);
  if (geoBox.y === recordBox.y) expect(geoBox.x).toBeLessThan(recordBox.x);
  expect(
    await records.evaluate((element) => {
      const geography = Array.from(
        document.querySelectorAll("section[aria-label]"),
      ).find(
        (section) => section.getAttribute("aria-label") === "Route footprint",
      )!;
      return Boolean(
        geography.compareDocumentPosition(element) &
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
  const distance = page.getByRole("region", {
    name: "Distance · all time",
    exact: true,
  });
  await expect(distance).toContainText("20.50 km");
  await expect(distance).toContainText("All time");
  const sleep = page.getByRole("region", {
    name: "Main sleep · recent",
    exact: true,
  });
  await expect(sleep).toContainText("Recent records (limit 30)");
  await expect(sleep).toContainText("—");
  await expect(
    sleep.getByRole("link", { name: "Explore sleep" }),
  ).toHaveAttribute("href", "/night");
  const library = page.getByRole("region", {
    name: "Library items",
    exact: true,
  });
  await expect(library).toContainText("All time");
  await expect(
    library.getByRole("link", { name: "Explore library" }),
  ).toHaveAttribute("href", "/library");
});

test("Expenses retained summaries and plot units do not adopt a pending currency", async ({
  page,
}) => {
  await installPilotFixtures(page, "expenses");
  await page.goto("/expenses?date=2026-08&currency=JPY");
  const tile = page.getByRole("region", {
    name: "Spending · JPY",
    exact: true,
  });
  await expect(tile).toContainText("¥12,345");
  await expect(tile).toContainText("Observed period: 2026-08");
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
      page
        .getByRole("region", { name: "Expense spending", exact: true })
        .getByRole("status")
        .filter({ hasText: "Updating…" }),
    ).toBeVisible();
    await expect(tile).toContainText("¥12,345");
    await expect(
      page.getByRole("heading", { name: "Daily spend · JPY", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Daily spend", exact: true }),
    ).toContainText("JPY minor");
  } finally {
    release();
  }
  await expect(
    page.getByRole("heading", { name: "Daily spend · USD", exact: true }),
  ).toBeVisible();
});
