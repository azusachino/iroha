import { test, expect } from "@playwright/test";
import { installPatternsFixtures } from "./patterns-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Patterns ${mode} first series failure is not zero periods`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.failures.add("monthly");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(page.getByText("0 periods", { exact: true })).toHaveCount(0);
    await expect(
      page.getByRole("heading", { name: "Steps over time", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("heading", {
        name: "Move, exercise, stand.",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry monthly patterns", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} failed new year does not relabel retained periods`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    fixture.failures.add("monthly:2025");
    await page
      .getByRole("combobox", { name: "Filter by year" })
      .selectOption("2025");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: 2026 · all months", {
        exact: true,
      }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} latest failure does not discard successful exact periods`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.failures.add("latest");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "9,000", exact: true }),
    ).toBeVisible();
    const count = fixture.requests.filter((request) =>
      request.startsWith("monthly:"),
    ).length;
    fixture.failures.delete("latest");
    await page
      .getByRole("button", { name: "Retry latest reading", exact: true })
      .click();
    await expect(
      page.getByRole("heading", {
        name: "Move, exercise, stand.",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Latest pattern reading", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((request) => request.startsWith("monthly:"))
        .length,
    ).toBe(count);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} confirmed empty is distinct from unavailable periods`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.empty = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await expect(page.getByText("0 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Steps over time", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} first pending periods keep independent latest reading and controls usable`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    let release!: () => void;
    fixture.holds.set(
      "monthly",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/patterns?date=2026&gran=month");
      await expect(
        page.getByText("Loading time-series data…", { exact: true }),
      ).toBeVisible();
      await expect(page.getByText("0 periods", { exact: true })).toHaveCount(0);
      await expect(
        page.getByRole("heading", { name: "Steps over time", exact: true }),
      ).toHaveCount(0);
      const latest = page.getByRole("heading", {
        name: "Move, exercise, stand.",
        exact: true,
      });
      await expect(latest).toBeVisible();
      expect(
        await latest.evaluate((node) => node.closest("[inert]")),
      ).toBeNull();
      await expect(
        page.getByRole("combobox", { name: "Filter by year" }),
      ).toBeEnabled();
    } finally {
      release();
    }
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} named series retry repeats failure then clears only that error`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.failures.add("monthly");
    fixture.failures.add("latest");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await page
      .getByRole("button", { name: "Retry monthly patterns", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Retry monthly patterns", exact: true }),
    ).toBeFocused();
    const count = fixture.requests.filter((request) =>
      request.startsWith("latest:"),
    ).length;
    fixture.failures.delete("monthly");
    await page.keyboard.press("Enter");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Pattern periods", exact: true }),
    ).toBeFocused();
    await expect(
      page.getByRole("button", { name: "Retry latest reading", exact: true }),
    ).toBeVisible();
    expect(
      fixture.requests.filter((request) => request.startsWith("latest:"))
        .length,
    ).toBe(count);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} range failure leaves truthful fallback selection and its own retry`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.failures.add("bounds");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month&keep=source");
    await expect(
      page.getByText("Calendar choices are not an observed data inventory.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("2026-08");
    await expect(page.getByText("1 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    fixture.failures.delete("bounds");
    await page
      .getByRole("button", { name: "Retry pattern range", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Retry pattern range", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("region", {
        name: "Pattern scope selection",
        exact: true,
      }),
    ).toBeFocused();
    expect(new URL(page.url()).searchParams.get("keep")).toBe("source");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} failed daily read does not destroy cached monthly periods`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    fixture.failures.add("days");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "day", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Retry daily patterns", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("0 periods", { exact: true })).toHaveCount(0);
    await expect(
      page.getByRole("heading", { name: "Steps over time", exact: true }),
    ).toHaveCount(0);
    await page.getByRole("button", { name: "month", exact: true }).click();
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    fixture.failures.delete("days");
    await page.getByRole("button", { name: "day", exact: true }).click();
    await expect(page.getByText("1 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: 2026-09", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} failed next daily scope retains exact prior day and observed month`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026-08&gran=day");
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    fixture.failures.add("days:2026-09");
    await page
      .getByRole("combobox", { name: "Filter by month" })
      .selectOption("2026-09");
    await expect(
      page.getByRole("button", { name: "Retry daily patterns", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: 2026-08", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("1 periods", { exact: true })).toBeVisible();
    fixture.failures.delete("days:2026-09");
    await page
      .getByRole("button", { name: "Retry daily patterns", exact: true })
      .click();
    await expect(
      page.getByRole("cell", { name: "9,000", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: 2026-09", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} lifetime year series keeps both years and averages`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?scope=lifetime&gran=year&keep=source");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "8,500", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "2,000", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: Lifetime", { exact: true }),
    ).toBeVisible();
    expect(new URL(page.url()).searchParams.get("scope")).toBe("lifetime");
    expect(new URL(page.url()).searchParams.get("keep")).toBe("source");
    expect(fixture.requests).toContain("yearly:lifetime");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} old pending day cannot block a newer month`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    let release!: () => void;
    fixture.holds.set(
      "days:2026-09",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    const oldResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        url.pathname === "/api/v1/daily" &&
        url.searchParams.get("date") === "2026-09"
      );
    });
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/patterns?date=2026-09&gran=day");
      await expect(
        page.getByText("Loading time-series data…", { exact: true }),
      ).toBeVisible();
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-08");
      await expect(
        page.getByRole("cell", { name: "8,000", exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed pattern scope: 2026-08", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
    }
    await oldResponse;
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("cell", { name: "9,000", exact: true }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Patterns ${mode} drilling retained month restores its actual year and URL`, async ({
    page,
  }) => {
    const fixture = await installPatternsFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/patterns?date=2026&gran=month&keep=source");
    await expect(page.getByText("2 periods", { exact: true })).toBeVisible();
    fixture.failures.add("monthly:2025");
    await page
      .getByRole("combobox", { name: "Filter by year" })
      .selectOption("2025");
    await expect(
      page.getByRole("button", { name: "Retry monthly patterns", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "2026-08, 8,000 steps", exact: true })
      .click();
    await expect(
      page.getByRole("cell", { name: "8,000", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed pattern scope: 2026-08", { exact: true }),
    ).toBeVisible();
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
    expect(new URL(page.url()).searchParams.get("gran")).toBe("day");
    expect(new URL(page.url()).searchParams.get("keep")).toBe("source");
    expect(fixture.unknown).toEqual([]);
  });
}
