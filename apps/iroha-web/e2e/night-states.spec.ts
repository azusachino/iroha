import { expect, test } from "@playwright/test";
import { installNightFixtures } from "./night-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Night ${mode} failed reads are unavailable rather than zero history`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    const fixture = await installNightFixtures(page);
    for (const read of ["records", "year", "month", "lifetime"] as const)
      fixture.failures.add(read);
    await page.goto("/night?date=2026-08");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.locator(".summary-row").getByText("0%", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByText("No sleep sessions in this observed scope.", {
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Retry sleep sessions", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", {
        name: "Retry monthly sleep totals",
        exact: true,
      }),
    ).toBeVisible();
    await page.screenshot({
      path: info.outputPath("night-unavailable.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} monthly failure preserves available year totals and records`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode });
    const fixture = await installNightFixtures(page);
    fixture.failures.add("month");
    await page.goto("/night?date=2026-08");
    await expect(
      page.getByRole("region", { name: "Sleep totals for 2026", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("table", { name: "Sleep session records", exact: true }),
    ).toContainText("2026-08-14");
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
    fixture.failures.delete("month");
    await page
      .getByRole("button", { name: "Retry monthly sleep totals", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("region", {
        name: "Sleep totals for 2026-08",
        exact: true,
      }),
    ).toContainText(/Total sessions\s+1/);
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} deferred initial reads never announce a zero`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("records", held);
    fixture.holds.set("month", held);
    try {
      await page.goto("/night?date=2026-08");
      await expect(
        page.getByText("Loading sleep sessions…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Loading sleep totals…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("region", { name: "Sleep summary", exact: true }),
      ).toContainText("—");
      await expect(
        page.getByRole("combobox", { name: "Filter by month" }),
      ).toBeEnabled();
      await expect(
        page.getByText("No sleep sessions in this observed scope.", {
          exact: true,
        }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("region", {
          name: "Sleep totals for 2026",
          exact: true,
        }),
      ).toBeVisible();
    } finally {
      release();
    }
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("2026-08-14");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} failed records preserve exact canonical totals and recover focus`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.failures.add("records");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    await expect(
      page.getByRole("region", { name: "Sleep summary", exact: true }),
    ).toContainText("7:00:00");
    await expect(
      page.getByText("No sleep sessions in this observed scope.", {
        exact: true,
      }),
    ).toHaveCount(0);
    fixture.failures.delete("records");
    await page
      .getByRole("button", { name: "Retry sleep sessions", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("2026-08-14");
    await expect(
      page.getByRole("region", {
        name: "Sleep session read state",
        exact: true,
      }),
    ).toBeFocused();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} confirmed empty scopes stay empty and ready`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-07");
    await expect(
      page.getByText("No sleep sessions in this observed scope.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Sleep summary", exact: true }),
    ).toContainText("Recorded sessions0");
    await expect(
      page
        .getByRole("region", { name: "Sleep summary" })
        .getByText("—", { exact: true }),
    ).toHaveCount(2);
    let release!: () => void;
    fixture.holds.set(
      "records",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-09");
      await expect(
        page.getByText("Updating sleep sessions…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("combobox", { name: "Filter by month" }),
      ).toBeEnabled();
    } finally {
      release();
      fixture.holds.delete("records");
    }
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("2026-09-14");
    fixture.empty = true;
    await page.goto("/night?scope=lifetime");
    await expect(
      page.getByText("No sleep sessions in this observed scope.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Sleep summary", exact: true }),
    ).toContainText("Recorded sessions0");
    await expect(
      page
        .getByRole("region", { name: "Sleep summary" })
        .getByText("—", { exact: true }),
    ).toHaveCount(2);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} failed bounds preserve requested date and recover the year control`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.failures.add("bounds");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("2026-08-14");
    await expect(
      page.getByRole("combobox", { name: "Filter by month" }),
    ).toHaveValue("2026-08");
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
    fixture.failures.delete("bounds");
    await page
      .getByRole("button", { name: "Retry sleep date range", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("combobox", { name: "Filter by year" }),
    ).toBeFocused();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} failed period change retains observed records with independent cached totals`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.paginated = true;
    let releaseAppend!: () => void;
    let releaseRecords!: () => void;
    fixture.pageHold = new Promise<void>((resolve) => {
      releaseAppend = resolve;
    });
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/night?date=2026-08");
      const table = page.getByRole("table", { name: "Sleep session records" });
      await expect(table).toContainText("2026-08-15");
      await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
      await expect
        .poll(() =>
          fixture.requests.some((value) =>
            value.includes("cursor=synthetic-cursor"),
          ),
        )
        .toBe(true);
      fixture.holds.set(
        "records",
        new Promise<void>((resolve) => {
          releaseRecords = resolve;
        }),
      );
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-09");
      await expect(
        page.getByRole("region", { name: "Sleep session read state" }),
      ).toContainText("Observed sessions: 2026-08");
      await expect(
        page.getByRole("region", { name: "Sleep summary" }),
      ).toContainText("6:00:00");
      fixture.failures.add("records");
      releaseRecords();
      fixture.holds.delete("records");
      await expect(
        page.getByRole("button", { name: "Retry sleep sessions", exact: true }),
      ).toBeVisible();
      const old = page.waitForResponse(
        (response) =>
          response.url().includes("cursor=synthetic-cursor") &&
          new URL(response.url()).searchParams.get("date") === "2026-08",
      );
      releaseAppend();
      await old;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(table).toContainText("2026-08-15");
      await expect(page.locator(".theme-load-more")).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Load more nights", exact: true }),
      ).toHaveCount(0);
      fixture.failures.delete("records");
      fixture.pageHold = null;
      await page
        .getByRole("button", { name: "Retry sleep sessions", exact: true })
        .press("Enter");
      await expect(table).toContainText("2026-09-15");
      await expect(table).not.toContainText("2026-08-15");
    } finally {
      releaseAppend();
      releaseRecords?.();
      fixture.pageHold = null;
      fixture.holds.delete("records");
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} year and lifetime failures recover independently`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.failures.add("year");
    fixture.failures.add("lifetime");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    await expect(
      page.getByRole("region", { name: "Sleep summary" }),
    ).toContainText("7:00:00");
    fixture.failures.delete("year");
    await page
      .getByRole("button", { name: "Retry yearly sleep totals", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("button", {
        name: "Retry lifetime sleep totals",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", {
        name: "Retry yearly sleep totals",
        exact: true,
      }),
    ).toHaveCount(0);
    fixture.failures.delete("lifetime");
    await page
      .getByRole("button", { name: "Retry lifetime sleep totals", exact: true })
      .press("Enter");
    await expect(page.getByRole("alert")).toHaveCount(0);
    const requests = fixture.requests.filter((request) =>
      request.includes("/aggregates"),
    );
    expect(
      requests.filter((request) => request.includes("granularity=month")),
    ).toHaveLength(1);
    expect(
      requests.filter((request) => request.includes("granularity=year")),
    ).toHaveLength(2);
    expect(
      requests.filter((request) => request.includes("granularity=lifetime")),
    ).toHaveLength(2);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} preserves canonical year lifetime and nap semantics`, async ({
    page,
  }, info) => {
    const fixture = await installNightFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026");
    await expect(
      page.getByRole("region", { name: "Sleep summary" }),
    ).toContainText("6:30:00");
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("2026-09-14");
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: info.outputPath("night-populated.png"),
      fullPage: true,
    });
    const initial = fixture.requests.map(
      (value) => new URL(value, "http://synthetic.local"),
    );
    const records = initial.find((url) => url.pathname === "/api/v1/sleep")!;
    expect(Object.fromEntries(records.searchParams)).toEqual({
      date: "2026",
      limit: "31",
      timezone: "Asia/Tokyo",
    });
    const aggregates = initial.filter((url) =>
      url.pathname.endsWith("aggregates"),
    );
    expect(aggregates).toHaveLength(3);
    for (const url of aggregates)
      expect(Object.fromEntries(url.searchParams)).toEqual({
        granularity: url.searchParams.get("granularity"),
        timezone: "Asia/Tokyo",
      });
    await page
      .getByRole("combobox", { name: "Filter by year" })
      .selectOption("");
    await expect(
      page.getByRole("region", { name: "Sleep session read state" }),
    ).toContainText("Observed sessions: Lifetime");
    expect(
      fixture.requests.some((value) => value === "/api/v1/sleep?limit=31"),
    ).toBe(true);
    fixture.napsOnly = true;
    await page.goto("/night?date=2026-08");
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("Nap");
    await expect(
      page.getByRole("table", { name: "Sleep session records" }),
    ).toContainText("50:00");
    await expect(
      page.getByRole("region", { name: "Sleep summary" }),
    ).toContainText("—");
    await expect(
      page
        .getByRole("region", { name: "Sleep summary" })
        .getByText("—", { exact: true }),
    ).toHaveCount(2);
    await expect(
      page.getByRole("region", { name: "Sleep totals for 2026-08" }),
    ).toContainText(/Naps\s+1/);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} stale append cannot join a newer month`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.paginated = true;
    let release!: () => void;
    fixture.pageHold = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/night?date=2026-08");
      const table = page.getByRole("table", { name: "Sleep session records" });
      await expect(table).toContainText("2026-08-15");
      await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
      await expect
        .poll(() =>
          fixture.requests.some((value) =>
            value.includes("cursor=synthetic-cursor"),
          ),
        )
        .toBe(true);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-09");
      await expect(table).toContainText("2026-09-15");
      await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
      await expect
        .poll(() =>
          fixture.requests.some(
            (value) =>
              value.includes("cursor=synthetic-cursor") &&
              value.includes("date=2026-09"),
          ),
        )
        .toBe(true);
      const oldResponse = page.waitForResponse(
        (response) =>
          response.url().includes("cursor=synthetic-cursor") &&
          new URL(response.url()).searchParams.get("date") === "2026-08",
      );
      release();
      await oldResponse;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(table).not.toContainText("2026-08-14");
      await expect(table).not.toContainText("2026-08-15");
      await expect(
        page.getByRole("region", { name: "Sleep session read state" }),
      ).toContainText("Observed sessions: 2026-09");
    } finally {
      release();
      fixture.pageHold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} superseded session read cannot resurrect an old selection`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    const table = page.getByRole("table", { name: "Sleep session records" });
    await expect(table).toContainText("2026-08-14");
    let release!: () => void;
    fixture.holds.set(
      "records",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-09");
      await expect
        .poll(() =>
          fixture.requests.some(
            (value) =>
              value.startsWith("/api/v1/sleep?") &&
              value.includes("date=2026-09"),
          ),
        )
        .toBe(true);
      fixture.holds.delete("records");
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-08");
      await expect(
        page.getByText("Updating sleep sessions…", { exact: true }),
      ).toHaveCount(0);
      const old = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname === "/api/v1/sleep" &&
          new URL(response.url()).searchParams.get("date") === "2026-09",
      );
      release();
      await old;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(table).not.toContainText("2026-09-14");
      await expect(table).toContainText("2026-08-14");
    } finally {
      release();
      fixture.holds.delete("records");
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} summary retry does not take a newer focus target`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.failures.add("month");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    const retry = page.getByRole("button", {
      name: "Retry monthly sleep totals",
      exact: true,
    });
    await expect(retry).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "month",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    fixture.failures.delete("month");
    try {
      await retry.press("Enter");
      const year = page.getByRole("combobox", { name: "Filter by year" });
      await year.focus();
      release();
      await expect(
        page.getByRole("region", { name: "Sleep totals for 2026-08" }),
      ).toContainText(/Total sessions\s+1/);
      await expect(year).toBeFocused();
    } finally {
      release();
      fixture.holds.delete("month");
    }
    expect(fixture.unknown).toEqual([]);
  });
}
