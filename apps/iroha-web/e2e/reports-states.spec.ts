import { test, expect, type Page } from "@playwright/test";
import { installReportsFixtures } from "./reports-fixtures";

async function expectRetainedAugust(page: Page) {
  await expect(
    page.getByText("Selected month: 2026-09", { exact: true }),
  ).toBeVisible();
  expect(new URL(page.url()).searchParams.get("date")).toBe("2026-09");
  await expect(
    page.getByText("Observed month: 2026-08", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Monthly comparison / 2026-08", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", {
      name: "Compare the canonical signals.",
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.locator(".generated")).toContainText(
    "2026-08-01 → 2026-09-01",
  );
  await expect(page.locator(".grapher-heading .period")).toHaveText(
    "2026-08-01 → 2026-09-01",
  );
  await expect(page.locator(".report-comparison .period-range")).toHaveText(
    "2025-09 → 2026-08",
  );
  await expect(
    page.getByRole("region", { name: "Exact evidence", exact: true }),
  ).toContainText("8 km");
  await expect(
    page.getByRole("region", { name: "Observed monthly report", exact: true }),
  ).toContainText("8 km");
}

for (const mode of ["light", "dark"] as const) {
  test(`Reports ${mode} pending September never relabels observed August`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Monthly comparison / 2026-08", { exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "2026-09",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByRole("combobox", { name: "Filter by month" }),
      ).toHaveValue("9");
      await expect(
        page.getByText("Monthly comparison / 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Monthly comparison / 2026-09", { exact: true }),
      ).toHaveCount(0);
      await expectRetainedAugust(page);
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });
  test(`Reports ${mode} failed September preserves observed August`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Monthly comparison / 2026-08", { exact: true }),
    ).toBeVisible();
    fixture.failures.add("2026-09");
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.getByText("Monthly comparison / 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Monthly comparison / 2026-09", { exact: true }),
    ).toHaveCount(0);
    await expectRetainedAugust(page);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} failed first read offers keyboard recovery without empty fabrication`, async ({
    page,
  }, info) => {
    const fixture = await installReportsFixtures(page);
    fixture.failures.add("2026-08");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    const retry = page.getByRole("button", {
      name: "Retry monthly report",
      exact: true,
    });
    await expect(retry).toBeVisible();
    await expect(
      page.getByText("Monthly report unavailable.", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", {
        name: "Compare the canonical signals.",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(page.getByText("No records", { exact: true })).toHaveCount(0);
    let reached = false;
    for (let tab = 0; tab < 30; tab++) {
      await page.keyboard.press("Tab");
      if (
        await retry.evaluate((element) => element === document.activeElement)
      ) {
        reached = true;
        break;
      }
    }
    expect(reached, "Retry monthly report reachable by Tab at 320").toBe(true);
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: info.outputPath("reports-unavailable.png"),
      fullPage: true,
    });
    fixture.failures.delete("2026-08");
    await retry.press("Enter");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", {
        name: "Observed monthly report",
        exact: true,
      }),
    ).toBeFocused();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} deferred first report does not publish an empty envelope`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    let release!: () => void;
    fixture.holds.set(
      "2026-08",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/reports?date=2026-08");
      await expect(
        page.getByText("Generating the monthly report…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", {
          name: "Compare the canonical signals.",
          exact: true,
        }),
      ).toHaveCount(0);
      await expect(page.getByText("No records", { exact: true })).toHaveCount(
        0,
      );
      await expect(
        page.getByRole("combobox", { name: "Filter by month" }),
      ).toBeEnabled();
    } finally {
      release();
    }
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} confirmed empty remains ready during quiet refresh`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    fixture.empty = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Observed monthly report" }),
    ).toContainText("No records");
    let release!: () => void;
    fixture.holds.set(
      "2026-09",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByText("Observed month: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Selected month: 2026-09", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Generating the monthly report…", { exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("region", { name: "Observed monthly report" }),
      ).not.toHaveAttribute("inert", "");
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} failed September retries requested month with exact observed data`, async ({
    page,
  }, info) => {
    const fixture = await installReportsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    fixture.failures.add("2026-09");
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(
      page.getByRole("button", { name: "Retry monthly report", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Selected month: 2026-09", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Observed monthly report" }),
    ).toContainText("8 km");
    fixture.failures.delete("2026-09");
    await page
      .getByRole("button", { name: "Retry monthly report", exact: true })
      .press("Enter");
    await expect(
      page.getByText("Observed month: 2026-09", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Observed monthly report" }),
    ).toContainText("9 km");
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(
      fixture.requests.filter((value) =>
        value.includes("date=2026-09&months=12&timezone=Asia%2FTokyo"),
      ),
    ).toHaveLength(2);
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-09");
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: info.outputPath("reports-populated.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} stale September cannot resurrect after newer August`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "2026-09",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect
        .poll(() =>
          fixture.requests.some((value) => value.includes("date=2026-09&")),
        )
        .toBe(true);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("8");
      await expect(
        page.getByRole("button", { name: "Refresh", exact: true }),
      ).toBeEnabled();
      const old = page.waitForResponse((response) =>
        response.url().includes("date=2026-09&"),
      );
      release();
      await old;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(
        page.getByText("Observed month: 2026-09", { exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByText("Monthly comparison / 2026-08", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} bounds failure preserves requested month and independent recovery`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    fixture.boundsFailure = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Selected month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("combobox", { name: "Filter by year" }),
    ).toHaveValue("2026");
    await expect(
      page.getByRole("combobox", { name: "Filter by month" }),
    ).toHaveValue("8");
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(
      page.getByText("Observed month: 2026-09", { exact: true }),
    ).toBeVisible();
    fixture.boundsFailure = false;
    await page
      .getByRole("button", { name: "Retry report date range", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("region", { name: "Report period controls", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((value) =>
        value.startsWith("/api/v1/reports/monthly-series"),
      ),
    ).toHaveLength(2);
    await expect(
      page
        .getByRole("combobox", { name: "Filter by year" })
        .locator('option[value=""]'),
    ).toHaveCount(0);
    await expect(
      page
        .getByRole("combobox", { name: "Filter by month" })
        .locator('option[value=""]'),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} report retry never retries unrelated failed date range`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    fixture.boundsFailure = true;
    fixture.failures.add("2026-08");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByRole("button", { name: "Retry monthly report", exact: true }),
    ).toBeVisible();
    fixture.failures.delete("2026-08");
    await page
      .getByRole("button", { name: "Retry monthly report", exact: true })
      .press("Enter");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", {
        name: "Retry report date range",
        exact: true,
      }),
    ).toBeVisible();
    expect(
      fixture.requests.filter((value) =>
        value.startsWith("/api/v1/daily/bounds"),
      ),
    ).toHaveLength(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} retry does not steal newer focus`, async ({ page }) => {
    const fixture = await installReportsFixtures(page);
    fixture.failures.add("2026-08");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByRole("button", { name: "Retry monthly report", exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "2026-08",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    fixture.failures.delete("2026-08");
    try {
      await page
        .getByRole("button", { name: "Retry monthly report", exact: true })
        .press("Enter");
      const year = page.getByRole("combobox", { name: "Filter by year" });
      await year.focus();
      release();
      await expect(
        page.getByText("Observed month: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(year).toBeFocused();
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} superseded September error cannot overwrite newer August`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    await expect(
      page.getByText("Observed month: 2026-08", { exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "2026-09",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    fixture.failures.add("2026-09");
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect
        .poll(() =>
          fixture.requests.some((value) => value.includes("date=2026-09&")),
        )
        .toBe(true);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("8");
      await expect(
        page.getByRole("button", { name: "Refresh", exact: true }),
      ).toBeEnabled();
      const old = page.waitForResponse((response) =>
        response.url().includes("date=2026-09&"),
      );
      release();
      const response = await old;
      expect(response.status()).toBe(503);
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(page.getByRole("alert")).toHaveCount(0);
      await expect(
        page.getByText("Selected month: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Monthly comparison / 2026-08", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  for (const scope of ["date=2025", "scope=lifetime"]) {
    test(`Reports ${mode} unsupported ${scope} normalizes to a truthful monthly URL`, async ({
      page,
    }) => {
      const fixture = await installReportsFixtures(page);
      await page.clock.install({ time: new Date("2026-08-15T12:00:00Z") });
      await page.emulateMedia({ colorScheme: mode });
      await page.goto(`/reports?${scope}&keep=synthetic`);
      await expect(
        page.getByText("Selected month: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed month: 2026-08", { exact: true }),
      ).toBeVisible();
      expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
      expect(new URL(page.url()).searchParams.has("scope")).toBe(false);
      expect(new URL(page.url()).searchParams.get("keep")).toBe("synthetic");
      const reports = fixture.requests.filter((value) =>
        value.startsWith("/api/v1/reports/monthly-series"),
      );
      expect(reports).toHaveLength(1);
      expect(reports[0]).toContain(
        "date=2026-08&months=12&timezone=Asia%2FTokyo",
      );
      expect(fixture.unknown).toEqual([]);
    });
  }

  test(`Reports ${mode} repeated failures restore named retries reachable at 320`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    fixture.boundsFailure = true;
    fixture.failures.add("2026-08");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2026-08");
    for (const name of ["Retry report date range", "Retry monthly report"]) {
      const retry = page.getByRole("button", { name, exact: true });
      await expect(retry).toBeVisible();
      let reached = false;
      for (let tab = 0; tab < 30; tab++) {
        await page.keyboard.press("Tab");
        if (
          await retry.evaluate((element) => element === document.activeElement)
        ) {
          reached = true;
          break;
        }
      }
      expect(reached, `${name} reachable by Tab at 320`).toBe(true);
      const before = fixture.requests.length;
      await retry.press("Enter");
      await expect.poll(() => fixture.requests.length).toBeGreaterThan(before);
      await expect(retry).toBeFocused();
    }
    await expect(
      page.getByRole("combobox", { name: "Filter by month" }),
    ).toHaveValue("8");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Reports ${mode} successful bounds retry clamps with a new truthful report read`, async ({
    page,
  }) => {
    const fixture = await installReportsFixtures(page);
    fixture.boundsFailure = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/reports?date=2025-01");
    await expect(
      page.getByText("Observed month: 2025-01", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("combobox", { name: "Filter by month" }),
    ).toHaveValue("1");
    fixture.boundsFailure = false;
    await page
      .getByRole("button", { name: "Retry report date range", exact: true })
      .press("Enter");
    await expect(
      page.getByText("Selected month: 2025-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed month: 2025-08", { exact: true }),
    ).toBeVisible();
    expect(new URL(page.url()).searchParams.get("date")).toBe("2025-08");
    const reports = fixture.requests.filter((value) =>
      value.startsWith("/api/v1/reports/monthly-series"),
    );
    expect(reports).toHaveLength(2);
    expect(reports[0]).toContain("date=2025-01&");
    expect(reports[1]).toContain("date=2025-08&");
    expect(fixture.unknown).toEqual([]);
  });
}
