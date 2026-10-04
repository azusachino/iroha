import { expect, test } from "@playwright/test";
import { installMotionFixtures } from "./motion-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Motion ${mode} empty month does not display the nonempty year total`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-07");
    await expect(
      page.getByText("Observed summary: 2026-07", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("0 m");
    await expect(
      page.getByText("No activity sessions in this observed scope.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByText("20.50 km", { exact: true })).toHaveCount(0);
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} unavailable date range preserves the requested period`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.failures.add("bounds");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("button", {
        name: "Retry activity date range",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("combobox", { name: "Filter by year" }),
    ).toHaveValue("2026");
    await expect(
      page.getByRole("combobox", { name: "Filter by month" }),
    ).toHaveValue("8");
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
    await expect(
      page.getByText("Observed summary: 2026-08", { exact: true }),
    ).toBeVisible();
    fixture.failures.clear();
    await page
      .getByRole("button", { name: "Retry activity date range", exact: true })
      .press("Enter");
    await expect(
      page
        .getByRole("combobox", { name: "Filter by year" })
        .getByRole("option", { name: "2025", exact: true }),
    ).toHaveCount(1);
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(new URL(page.url()).searchParams.get("date")).toBe("2026-08");
    await expect(
      page.getByRole("combobox", { name: "Filter by year" }),
    ).toBeFocused();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} preserves calendar sport and canonical series queries`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08&sport=run");
    const table = page.getByRole("table", { name: "Exact movement series" });
    await expect(table).toContainText("12.50 km");
    await expect(table).toContainText("1h 2m");
    await expect(
      page.getByText("Observed summary: 2026-08 · Run", { exact: true }),
    ).toBeVisible();
    const requests = fixture.requests.map(
      (request) => new URL(request, "http://synthetic.local"),
    );
    const records = requests.find(
      (request) => request.pathname === "/api/v1/activities",
    )!;
    expect(Object.fromEntries(records.searchParams)).toEqual({
      date: "2026-08",
      sport_type: "run",
      limit: "24",
      timezone: "Asia/Tokyo",
    });
    const summary = requests.find((request) =>
      request.pathname.endsWith("/summary"),
    )!;
    expect(Object.fromEntries(summary.searchParams)).toEqual({
      date: "2026-08",
      sport: "run",
      timezone: "Asia/Tokyo",
    });
    const series = requests.filter((request) =>
      request.pathname.endsWith("/series"),
    );
    expect(series).toHaveLength(2);
    for (const request of series)
      expect(Object.fromEntries(request.searchParams)).toEqual({
        from: "2026-08-01",
        to: "2026-09-01",
        grain: "day",
        timezone: "Asia/Tokyo",
        dimension: "sport:run",
      });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} missing distance remains unknown in records summary and exact series`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.missingDistance = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByText("1 session has no observed distance.", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("—");
    const table = page.getByRole("table", { name: "Exact movement series" });
    await expect(table).toContainText("2026-08-14");
    await expect(table).toContainText("—");
    await expect(table).toContainText("1h 2m");
    await expect(
      page.getByRole("region", { name: "Activity records", exact: true }),
    ).toContainText("—");
    await expect(page.getByText("0 m", { exact: true })).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} stale append cannot join a newer period`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.paginated = true;
    await page.emulateMedia({ colorScheme: mode });
    let release!: () => void;
    // The production sentinel can request a page before a manual click.
    // Hold it BEFORE navigation, then exercise the existing scroll trigger.
    fixture.pageHold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page.goto("/motion?date=2026-08");
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed summary: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("table", { name: "Exact movement series" }),
      ).toBeVisible();
      await page.getByTestId("motion-load-sentinel").scrollIntoViewIfNeeded();
      await expect
        .poll(() =>
          fixture.requests.some((request) =>
            request.includes("cursor=synthetic-cursor"),
          ),
        )
        .toBe(true);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-09", exact: true }),
      ).toBeVisible();
      const oldPage = page.waitForResponse((response) =>
        response.url().includes("cursor=synthetic-cursor"),
      );
      release();
      await oldPage;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(
        page.getByRole("link", {
          name: "Synthetic next run 2026-08",
          exact: true,
        }),
      ).toHaveCount(0);
      await expect(
        page.getByText("Observed records: 2026-09", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
      fixture.pageHold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} retry does not steal focus from a new user target`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.failures.add("records");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    const retry = page.getByRole("button", {
      name: "Retry activities",
      exact: true,
    });
    await expect(retry).toBeVisible();
    fixture.failures.clear();
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await retry.press("Enter");
      const command = page.getByRole("button", {
        name: "Open command palette",
      });
      await command.focus();
      release();
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
      ).toBeVisible();
      await expect(command).toBeFocused();
    } finally {
      release();
      fixture.hold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} deferred first load does not announce zero history`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page.goto("/motion?date=2026-08");
      await expect(
        page.getByRole("status").filter({ hasText: "Loading activities…" }),
      ).toBeVisible();
      await expect(page.getByText("0 m", { exact: true })).toHaveCount(0);
      await expect(page.getByText("0:00", { exact: true })).toHaveCount(0);
    } finally {
      release();
      fixture.hold = null;
    }
    await expect(
      page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed summary: 2026-08", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} failed reads are unavailable, not zero history`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installMotionFixtures(page);
    for (const dependency of ["summary", "records", "series"] as const)
      fixture.failures.add(dependency);
    await page.goto("/motion?date=2026-08");
    await expect(page.getByText(/Could not load activity data/)).toBeVisible();
    await expect(page.getByText("0 m", { exact: true })).toHaveCount(0);
    await expect(page.getByText("0:00", { exact: true })).toHaveCount(0);
    await expect(
      page.getByText("No movement series is available for this scope.", {
        exact: true,
      }),
    ).toHaveCount(0);
    const retryNames = [
      "Retry activities",
      "Retry activity summary",
      "Retry movement series",
    ];
    for (const name of retryNames)
      await expect(
        page.getByRole("button", { name, exact: true }),
      ).toBeVisible();
    const reached = new Set<string>();
    for (let step = 0; step < 20 && reached.size < retryNames.length; step++) {
      await page.keyboard.press("Tab");
      const name = await page.evaluate(
        () => document.activeElement?.textContent?.trim() ?? "",
      );
      if (retryNames.includes(name)) reached.add(name);
    }
    expect([...reached].sort()).toEqual([...retryNames].sort());
    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement)
        document.activeElement.blur();
      window.scrollTo(0, 0);
    });
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("—");
    await page.screenshot({
      path: info.outputPath("motion-unavailable.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} summary failure preserves independent records and chart`, async ({
    page,
  }, info) => {
    const fixture = await installMotionFixtures(page);
    fixture.failures.add("summary");
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("—");
    await expect(
      page.getByRole("button", { name: "Retry activity summary", exact: true }),
    ).toBeVisible();
    fixture.failures.clear();
    await page
      .getByRole("button", { name: "Retry activity summary", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("12.50 km");
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(
      page.getByRole("region", {
        name: "Activity summary read state",
        exact: true,
      }),
    ).toBeFocused();
    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement)
        document.activeElement.blur();
      window.scrollTo(0, 0);
    });
    await page.screenshot({
      path: info.outputPath("motion-populated.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} list failure retains independent facts and recovers`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.failures.add("records");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("12.50 km");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    await expect(
      page.getByText("No activity sessions in this observed scope.", {
        exact: true,
      }),
    ).toHaveCount(0);
    fixture.failures.clear();
    await page
      .getByRole("button", { name: "Retry activities", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(
      page.getByRole("region", { name: "Activity records", exact: true }),
    ).toBeFocused();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} series failure is not an empty plot`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.failures.add("series");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("No movement series is available for this scope.", {
        exact: true,
      }),
    ).toHaveCount(0);
    fixture.failures.clear();
    await page
      .getByRole("button", { name: "Retry movement series", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Movement over time", exact: true }),
    ).toBeFocused();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} successful empty lifetime is ready for quiet refetch`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?scope=lifetime&sport=walk");
    await expect(
      page.getByText("No activity sessions in this observed scope.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("0 m");
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by year" })
        .selectOption("2026");
      await expect(page.getByText("Updating…", { exact: true })).toBeVisible();
      await expect(
        page.getByText("Loading activities…", { exact: true }),
      ).toHaveCount(0);
      await expect(page.getByRole("combobox", { name: "Sport" })).toBeEnabled();
      await expect(
        page.getByText("Observed records: Lifetime · Walk", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
      fixture.hold = null;
    }
    await expect(
      page.getByText("Observed records: 2026 · Walk", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} failed lifetime scope cannot revive a superseded chart`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toContainText("2026-08-14");
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect
        .poll(
          () =>
            fixture.requests.filter(
              (request) =>
                request.includes("date=2026-09") ||
                request.includes("from=2026-09-01"),
            ).length,
        )
        .toBeGreaterThanOrEqual(4);
      fixture.hold = null;
      fixture.failures.add("summary");
      await page
        .getByRole("combobox", { name: "Filter by year" })
        .selectOption("");
      await expect(
        page.getByRole("button", {
          name: "Retry activity summary",
          exact: true,
        }),
      ).toBeVisible();
      const oldResponses = [
        "movement.distance_m/series?",
        "movement.duration_s/series?",
      ].map((path) =>
        page.waitForResponse(
          (response) =>
            response.url().includes(path) &&
            response.url().includes("from=2026-09-01"),
        ),
      );
      release();
      await Promise.all(oldResponses);
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(
        page.getByRole("table", { name: "Exact movement series" }),
      ).toContainText("2026-08-14");
      await expect(
        page.getByText("Observed summary: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed records: Lifetime", { exact: true }),
      ).toBeVisible();
      fixture.failures.clear();
      await page
        .getByRole("button", { name: "Retry movement series", exact: true })
        .click();
      await expect(
        page.getByText("Observed summary: Lifetime", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("region", { name: "Movement over time" }),
      ).toContainText("· Lifetime.");
    } finally {
      release();
      fixture.hold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} superseded month responses cannot resurrect old data`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect
        .poll(
          () =>
            fixture.requests.filter(
              (request) =>
                request.includes("date=2026-09") ||
                request.includes("from=2026-09-01"),
            ).length,
        )
        .toBeGreaterThanOrEqual(4);
      fixture.hold = null;
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("8");
      await expect(page.getByText("Updating…", { exact: true })).toHaveCount(0);
      const oldResponses = [
        "/activities?",
        "/activities/summary?",
        "movement.distance_m/series?",
        "movement.duration_s/series?",
      ].map((path) =>
        page.waitForResponse(
          (response) =>
            response.url().includes(path) &&
            (response.url().includes("date=2026-09") ||
              response.url().includes("from=2026-09-01")),
        ),
      );
      release();
      await Promise.all(oldResponses);
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(
        page.getByText("Observed summary: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed records: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-09", exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("region", { name: "Filtered activity summary" }),
      ).toContainText("12.50 km");
    } finally {
      release();
      fixture.hold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} deferred and failed month change retains observed scope`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    const record = page.getByRole("link", {
      name: "Synthetic run 2026-08",
      exact: true,
    });
    await expect(record).toBeVisible();
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    let release!: () => void;
    fixture.hold = new Promise<void>((resolve) => {
      release = resolve;
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(page.getByText("Updating…", { exact: true })).toBeVisible();
      await expect(record).toBeVisible();
      await record.focus();
      await expect(record).toBeFocused();
      await expect(
        page.getByText("Observed records: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed summary: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("region", { name: "Filtered activity summary" }),
      ).toContainText("12.50 km");
      await expect(
        page.getByRole("table", { name: "Exact movement series" }),
      ).toBeVisible();
      for (const dependency of ["summary", "records", "series"] as const)
        fixture.failures.add(dependency);
    } finally {
      release();
      fixture.hold = null;
    }
    await expect(
      page.getByRole("button", { name: "Retry activities", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry activity summary", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry movement series", exact: true }),
    ).toBeVisible();
    await expect(record).toBeVisible();
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed summary: 2026-08", { exact: true }),
    ).toBeVisible();
    fixture.failures.clear();
    await page
      .getByRole("button", { name: "Retry activities", exact: true })
      .click();
    await expect(
      page.getByRole("link", { name: "Synthetic run 2026-09", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed records: 2026-09", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Observed summary: 2026-08", { exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Retry activity summary", exact: true })
      .click();
    await expect(
      page.getByText("Observed summary: 2026-09", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Filtered activity summary" }),
    ).toContainText("8.00 km");
    await expect(
      page.getByRole("button", { name: "Retry movement series", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Retry movement series", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toContainText("2026-09-14");
    await expect(
      page.getByRole("table", { name: "Exact movement series" }),
    ).toContainText("8.00 km");
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });
}
