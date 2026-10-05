import { test, expect } from "@playwright/test";
import { installExpensesFixtures } from "./expenses-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Expenses ${mode} first failure does not fabricate empty records`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.failures.add("ledger");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.getByText("No canonical expenses match these filters.", {
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("heading", { name: "Expense records", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Retry expense records", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} spending failure leaves exact ledger usable`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.failures.add("spending");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Expense records", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Daily spend · JPY", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Retry expense spending", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} deferred first reads show no invented ledger or totals`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    let release!: () => void;
    const hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("ledger", hold);
    fixture.holds.set("spending", hold);
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/expenses?date=2026-08");
      await expect(
        page.getByText("Loading expense records…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: "Expense records", exact: true }),
      ).toHaveCount(0);
      await expect(page.locator(".stat-tile")).toHaveCount(0);
      await expect(
        page.getByText("No canonical expenses match these filters.", {
          exact: true,
        }),
      ).toHaveCount(0);
    } finally {
      release();
    }
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} confirmed empty remains ready and null is not zero`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.empty = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    const empty = page.getByText("No canonical expenses match these filters.", {
      exact: true,
    });
    await expect(empty).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Spending · JPY", exact: true })
        .locator(".stat-value"),
    ).toHaveText("—");
    await expect(
      page.getByRole("region", { name: "Spending · JPY", exact: true }),
    ).toContainText("Record count unavailable");
    let release!: () => void;
    const hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("ledger", hold);
    fixture.holds.set("spending", hold);
    try {
      await page.getByRole("button", { name: "Refresh", exact: true }).click();
      await expect(
        page.getByText("Updating…", { exact: true }).first(),
      ).toBeVisible();
      await expect(empty).toBeVisible();
      expect(
        await empty.evaluate((node) => node.closest("[inert]")),
      ).toBeNull();
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} independent range failure keeps calendar choices and recovers`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.failures.add("bounds");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08&keep=receipt");
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByText(
        "Period choices are calendar choices, not an observed data inventory.",
        { exact: true },
      ),
    ).toBeVisible();
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(
      page.getByText("Synthetic september", { exact: true }).first(),
    ).toBeVisible();
    expect(new URL(page.url()).searchParams.get("keep")).toBe("receipt");
    fixture.failures.delete("bounds");
    const count = fixture.requests.filter((request) =>
      request.startsWith("GET /api/v1/expenses?"),
    ).length;
    await page
      .getByRole("button", { name: "Retry expense range", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Retry expense range", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("region", {
        name: "Expense period selection",
        exact: true,
      }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((request) =>
        request.startsWith("GET /api/v1/expenses?"),
      ).length,
    ).toBe(count);
    await expect(
      page
        .getByRole("combobox", { name: "Filter by year" })
        .locator("option[value='2025']"),
    ).toHaveCount(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} keyboard retry recovers only failed records`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.failures.add("ledger");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    const retry = page.getByRole("button", {
      name: "Retry expense records",
      exact: true,
    });
    await expect(retry).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Spending · JPY", exact: true })
        .locator(".stat-value"),
    ).toHaveText("¥12,345");
    const count = fixture.requests.filter((request) =>
      request.includes("/series?"),
    ).length;
    let reached = false;
    for (let tab = 0; tab < 40; tab++) {
      await page.keyboard.press("Tab");
      if (await retry.evaluate((node) => node === document.activeElement)) {
        reached = true;
        break;
      }
    }
    expect(reached).toBe(true);
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("button", { name: "Retry expense records", exact: true }),
    ).toBeFocused();
    fixture.failures.delete("ledger");
    await page.keyboard.press("Enter");
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Expense records read", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((request) => request.includes("/series?")).length,
    ).toBe(count);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} failed spending refresh retains its own period and currency`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    await expect(
      page
        .getByRole("region", { name: "Spending · JPY", exact: true })
        .locator(".stat-value"),
    ).toHaveText("¥12,345");
    fixture.failures.add("spending:2026-09");
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("9");
    await expect(
      page.getByRole("button", { name: "Retry expense spending", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Synthetic september", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByText(
        "Observed spending period: 2026-08 · All currencies · All categories",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(
      page.getByText(
        "Observed ledger period: 2026-09 · All currencies · All categories",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Daily spend · JPY", exact: true }),
    ).toBeVisible();
    const records = fixture.requests.filter((request) =>
      request.startsWith("GET /api/v1/expenses?"),
    ).length;
    fixture.failures.delete("spending:2026-09");
    await page
      .getByRole("button", { name: "Retry expense spending", exact: true })
      .click();
    await expect(
      page
        .getByRole("region", { name: "Spending · USD", exact: true })
        .locator(".stat-value"),
    ).toHaveText("$1,234.56");
    await expect(
      page.getByRole("region", { name: "Expense spending", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((request) =>
        request.startsWith("GET /api/v1/expenses?"),
      ).length,
    ).toBe(records);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} stale September cannot overwrite a newer August read`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    let release!: () => void;
    const hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("ledger:2026-09", hold);
    fixture.holds.set("spending:2026-09", hold);
    const oldResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        url.pathname === "/api/v1/expenses" &&
        url.searchParams.get("date") === "2026-09"
      );
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByText(
          "Selected period: 2026-09 · All currencies · All categories",
          { exact: true },
        ),
      ).toBeVisible();
      await expect(
        page.getByText(
          "Observed ledger period: 2026-08 · All currencies · All categories",
          { exact: true },
        ),
      ).toBeVisible();
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("8");
      await expect(
        page.getByRole("button", { name: "Refresh", exact: true }),
      ).toBeEnabled();
    } finally {
      release();
    }
    await oldResponse;
    await expect(
      page.getByText("Synthetic august", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByText("Synthetic september", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page
        .getByRole("region", { name: "Spending · JPY", exact: true })
        .locator(".stat-value"),
    ).toHaveText("¥12,345");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} year totals cover both months and exact money units`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026&keep=receipt");
    await expect(
      page
        .getByRole("region", { name: "Spending · JPY", exact: true })
        .locator(".stat-value"),
    ).toHaveText("¥12,345");
    await expect(
      page
        .getByRole("region", { name: "Spending · USD", exact: true })
        .locator(".stat-value"),
    ).toHaveText("$1,234.56");
    await expect(
      page.getByText("123456 minor USD", { exact: true }),
    ).toHaveCount(0);
    await page.getByRole("button", { name: /Synthetic september/ }).click();
    await expect(
      page.getByText("123456 minor USD", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("manual · synthetic-september", { exact: true }),
    ).toBeVisible();
    expect(
      fixture.requests
        .filter(
          (request) =>
            request.includes("/series?") && !request.includes("grain=day"),
        )
        .every(
          (request) =>
            request.includes("grain=year") &&
            request.includes("from=2026-01-01") &&
            request.includes("to=2027-01-01"),
        ),
    ).toBe(true);
    expect(new URL(page.url()).searchParams.get("keep")).toBe("receipt");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} observed net zero is not an unavailable money value`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.netZero = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08&currency=JPY&category=food");
    const tile = page.getByRole("region", {
      name: "Spending · JPY",
      exact: true,
    });
    await expect(tile.locator(".stat-value")).toHaveText("¥0");
    await expect(tile).toContainText("2 records");
    await expect(
      page.getByText("Synthetic refund", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      page.getByText("Selected period: 2026-08 · JPY · Food", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Expenses ${mode} mutation failure survives an unrelated successful read`, async ({
    page,
  }) => {
    const fixture = await installExpensesFixtures(page);
    fixture.failures.add("delete");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/expenses?date=2026-08");
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await page
      .getByRole("alertdialog")
      .getByRole("button", { name: "Delete", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText("Delete failed:");
    const count = fixture.requests.filter((request) =>
      request.startsWith("GET /api/v1/expenses?"),
    ).length;
    await page.getByRole("button", { name: "Refresh", exact: true }).click();
    await expect
      .poll(
        () =>
          fixture.requests.filter((request) =>
            request.startsWith("GET /api/v1/expenses?"),
          ).length,
      )
      .toBe(count + 1);
    await expect(
      page.getByRole("button", { name: "Refresh", exact: true }),
    ).toBeEnabled();
    await expect(page.getByRole("alert")).toContainText("Delete failed:");
    expect(fixture.unknown).toEqual([]);
  });
}
