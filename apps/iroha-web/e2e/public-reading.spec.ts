import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { inspectPilotCharts } from "./pilot-charts";
import { fakeSession } from "./session";

test("public reading section renders the sanitized book list", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.goto(PUBLIC_BASE_URL);

  await expect(
    page.getByRole("heading", { name: "Miles & Margins" }),
  ).toBeVisible();
  const run = page.getByRole("tab", { name: "Run" });
  const books = page.getByRole("tab", { name: "Books" });
  await expect(run).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("heading", { name: "Activities" })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Reading", exact: true }),
  ).toHaveCount(0);
  await run.focus();
  await page.keyboard.press("ArrowRight");
  await expect(books).toBeFocused();
  await expect(books).toHaveAttribute("aria-selected", "true");
  await expect(
    page.getByRole("heading", { name: "Reading", exact: true }),
  ).toBeVisible();

  const section = page.getByRole("region", { name: "Reading list" });
  await expect(section).toBeVisible();

  await expect(section.getByText("A life in books.")).toBeVisible();
  await expect(section.locator(".total strong")).toHaveText("6");
  await expect(section.getByText("A Winter Book")).toBeVisible();
  await expect(section.getByText("A Spring Book")).toBeVisible();
  await expect(section.getByText("如何了解一个人")).toHaveCount(0);
  await expect(section.getByText("2 books finished")).toBeVisible();
  await expect(
    section.getByRole("img", {
      name: /Books finished each month in 2026: January 1, February 0, March 1/,
    }),
  ).toBeVisible();

  await expect(
    section.getByRole("img", { name: /Books finished chart/ }),
  ).toBeVisible();
  await section
    .getByRole("combobox", { name: "Reading year" })
    .selectOption("2024");
  await expect(section.getByText("克拉拉与太阳")).toBeVisible();
  await expect(section.getByText("石黑一雄")).toBeVisible();
  await expect(section.getByText("A Winter Book")).toHaveCount(0);
  await section
    .getByRole("combobox", { name: "Reading year" })
    .selectOption("2025");
  await expect(section.getByText("New Year's Eve")).toBeVisible();
  await expect(
    section.getByRole("img", { name: /November 0, December 1/ }),
  ).toBeVisible();
  await section
    .getByRole("combobox", { name: "Reading year" })
    .selectOption("undated");
  await expect(section.getByText("An Undated Book")).toBeVisible();
  await expect(section.getByText("A Book With an Invalid Date")).toBeVisible();
  await expect(
    section.getByRole("img", { name: /Books finished each month/ }),
  ).toHaveCount(0);
});

test("yearly trend scales beyond a decade and includes empty years", async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await installPilotFixtures(page, "public");
  const years = Array.from({ length: 16 }, (_, index) => 2010 + index).filter(
    (year) => year !== 2018,
  );
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({
      json: years.map((year) => ({
        id: String(year),
        title: `Book ${year}`,
        authors: [],
        status: "completed",
        finished_at: `${year}-06-01`,
      })),
    }),
  );
  await page.goto(PUBLIC_BASE_URL);
  await page.getByRole("tab", { name: "Books" }).click();
  const section = page.getByRole("region", { name: "Reading list" });
  const range = section.getByRole("combobox", { name: "Trend range" });
  await expect(range).toHaveValue("recent");
  const scroller = section.getByRole("region", { name: /Yearly trend chart/ });
  await expect
    .poll(() => scroller.evaluate((element) => element.scrollLeft))
    .toBeGreaterThan(0);
  await section.getByText("View chart data").click();
  const table = section.getByRole("table", {
    name: "Books finished by category",
  });
  await expect(table.getByRole("row", { name: "2016 1" })).toBeVisible();
  await expect(table.getByRole("row", { name: "2018 0" })).toBeVisible();
  await expect(table.getByRole("row", { name: "2010 1" })).toHaveCount(0);
  await section
    .getByRole("combobox", { name: "Reading year" })
    .selectOption("2010");
  await expect(range).toHaveValue("2010");
  await expect
    .poll(() => scroller.evaluate((element) => element.scrollLeft))
    .toBe(0);
  await expect(table.getByRole("row", { name: "2010 1" })).toBeVisible();
  await expect(table.getByRole("row", { name: "2018 0" })).toBeVisible();
  await expect(section.getByText("Book 2010")).toBeVisible();
  await expect
    .poll(async () => {
      const result = await page.evaluate(inspectPilotCharts);
      if (!result.verified) return [];
      return result.charts.find((chart) =>
        chart.label?.startsWith("Books finished chart"),
      )?.xAxis[0]?.data;
    })
    .toEqual([
      "2010",
      "2011",
      "2012",
      "2013",
      "2014",
      "2015",
      "2016",
      "2017",
      "2018",
      "2019",
    ]);
  await range.selectOption("recent");
  await expect(
    section.getByRole("combobox", { name: "Reading year" }),
  ).toHaveValue("2025");
  await expect(table.getByRole("row", { name: "2025 1" })).toBeVisible();
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(320);
});

test("distant dates do not make the trend unbounded", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await installPilotFixtures(page, "public");
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({
      json: ["0000", "2025"].map((year) => ({
        id: year,
        title: `Book ${year}`,
        authors: [],
        status: "completed",
        finished_at: `${year}-06-01`,
      })),
    }),
  );
  await page.goto(PUBLIC_BASE_URL);
  await page.getByRole("tab", { name: "Books" }).click();
  const section = page.getByRole("region", { name: "Reading list" });
  await section
    .getByRole("combobox", { name: "Reading year" })
    .selectOption("0000");
  await expect(
    section.getByRole("combobox", { name: "Trend range" }),
  ).toHaveValue("0");
  await section.getByText("View chart data").click();
  const table = section.getByRole("table", {
    name: "Books finished by category",
  });
  await expect(table.getByRole("row", { name: "0000 1" })).toBeVisible();
  await expect(table.locator("tbody tr")).toHaveCount(10);
  await expect(section.getByText("Book 0000")).toBeVisible();
  const chartWidth = await section
    .getByRole("img", { name: /Books finished chart/ })
    .evaluate((element) => element.getBoundingClientRect().width);
  expect(chartWidth).toBeLessThanOrEqual(550);
});

test("undated books with long titles fit a narrow screen", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await installPilotFixtures(page, "public");
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({
      json: [
        {
          id: "undated",
          title: "LongUnbrokenTitle".repeat(20),
          authors: ["Author"],
          status: "completed",
        },
      ],
    }),
  );
  await page.goto(PUBLIC_BASE_URL);
  await page.getByRole("tab", { name: "Books" }).click();
  const section = page.getByRole("region", { name: "Reading list" });
  await expect(section.locator(".total strong")).toHaveText("1");
  await expect(section.getByText("Undated books")).toBeVisible();
  await expect(section.getByText("LongUnbrokenTitle".repeat(20))).toBeVisible();
  await expect(section.locator(".book-date")).toHaveText("Date unavailable");
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(320);
});

test("public reading section handles records with no finished books", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({
      json: [
        {
          id: "reading-only",
          title: "Still reading",
          authors: [],
          status: "in_progress",
        },
      ],
    }),
  );
  await page.goto(PUBLIC_BASE_URL);
  await page.getByRole("tab", { name: "Books" }).click();
  await expect(page.getByText("No finished books yet.")).toBeVisible();
  await expect(page.getByText("Still reading")).toHaveCount(0);
});

test("public reading section quiets itself when the part is unavailable", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await page.route("**/public/v1/books", (route) =>
    route.fulfill({ status: 503, json: { error: "synthetic failure" } }),
  );
  await page.goto(PUBLIC_BASE_URL);

  await page.getByRole("tab", { name: "Books" }).click();
  await expect(
    page.getByText("Reading list is temporarily unavailable."),
  ).toBeVisible();
  // Switching back preserves access to the activity archive.
  await page.getByRole("tab", { name: "Run" }).click();
  await expect(
    page.getByRole("heading", { name: "Activities", exact: true }),
  ).toBeVisible();
});

test("signed-in owners can open and navigate the public archive", async ({
  page,
}) => {
  await installPilotFixtures(page, "public");
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    username: "synthetic-owner",
    display_name: "Synthetic owner",
  });
  await page.goto("/overview");
  await page.locator("summary").filter({ hasText: "More" }).click();
  await page.getByRole("link", { name: "Public archive" }).click();
  await expect(page).toHaveURL(/\/showcase$/);
  await expect(
    page.getByRole("heading", { name: "Miles & Margins" }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Books" }).click();
  await expect(
    page.getByRole("region", { name: "Reading list" }),
  ).toBeVisible();
  await page.goto("/showcase?activity=synthetic-2026");
  await page.getByRole("link", { name: "Back to archive" }).click();
  await expect(page).toHaveURL(/\/showcase$/);
  await expect(
    page.getByRole("heading", { name: "Miles & Margins" }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Today", exact: true }),
  ).toBeVisible();
});

test("public tabs fit and switch at a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await installPilotFixtures(page, "public");
  await page.goto(PUBLIC_BASE_URL);
  await page.getByRole("tab", { name: "Books" }).click();
  await expect(
    page.getByRole("region", { name: "Reading list" }),
  ).toBeVisible();
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(320);
  await page.getByRole("tab", { name: "Run" }).click();
  await expect(page.getByRole("heading", { name: "Activities" })).toBeVisible();
});

test("public reading section stays empty-safe for signed-out visitors at the showcase address", async ({
  page,
}) => {
  await installPilotFixtures(page, "public", "empty");
  await page.goto("/showcase");

  await page.getByRole("tab", { name: "Books" }).click();
  await expect(page.getByText("No reading records yet.")).toBeVisible();
});
