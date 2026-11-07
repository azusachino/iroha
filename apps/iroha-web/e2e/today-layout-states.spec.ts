import { expect, test, type Page } from "@playwright/test";
import { fakeSession } from "./session";

const status = {
  availability: "supported",
  collection: "covered",
  operation: "idle",
  freshness: "within_cadence",
};

const populatedSections = [
  {
    key: "daily",
    schema: "daily-day.v1",
    state: "ready",
    status,
    data: {
      items: [
        {
          id: "layout-daily",
          day: "2020-08-15",
          steps: 9000,
          distance_km: 6.2,
          resting_hr: 55,
          hrv_sdnn: 48,
          spo2_avg: 97.5,
          respiratory_rate: 14.5,
          body_mass_kg: 70.2,
          ring: {
            move_kcal: 420,
            move_goal_kcal: 500,
            exercise_min: 31,
            exercise_goal_min: 30,
            stand_hours: 10,
            stand_goal_hours: 12,
          },
          source: "synthetic",
          first_raw_file_id: "layout-raw",
          created_at: "2020-08-15T12:00:00Z",
          updated_at: "2020-08-15T12:00:00Z",
        },
      ],
      has_more: false,
    },
  },
  {
    key: "sleep",
    schema: "sleep-day.v1",
    state: "ready",
    status,
    data: {
      items: [
        {
          id: "layout-sleep",
          wake_date: "2020-08-15",
          started_at: "2020-08-14T22:00:00Z",
          ended_at: "2020-08-15T06:00:00Z",
          time_in_bed_s: 28800,
          asleep_s: 25200,
          efficiency: 0.875,
          is_main_sleep: true,
          core_s: 14400,
          deep_s: 3600,
          rem_s: 7200,
          awake_s: 3600,
          unspecified_s: 0,
          source: "synthetic",
          first_raw_file_id: "layout-raw",
          created_at: "2020-08-15T12:00:00Z",
          updated_at: "2020-08-15T12:00:00Z",
        },
      ],
      has_more: false,
    },
  },
  {
    key: "activities",
    schema: "activities-day.v1",
    state: "ready",
    status,
    data: {
      items: [
        {
          id: "layout-run",
          title: "Synthetic morning run",
          sport_type: "run",
          started_at: "2020-08-15T07:00:00Z",
          timezone: "UTC",
          distance_m: 8000,
          duration_s: 3000,
          source_kind: "synthetic",
          first_raw_file_id: "layout-raw",
          created_at: "2020-08-15T12:00:00Z",
          updated_at: "2020-08-15T12:00:00Z",
        },
      ],
      has_more: false,
    },
  },
];

const dateStates = [
  "confirmed-empty(sections ready-empty)",
  "initial-503",
  "named-retry-success",
  "partial-briefing-503",
  "partial-calendar-503",
  "populated",
  "retained-failed-refetch(previous-day)",
  "section-unavailable(partial-dependency)",
] as const;
type DateState = (typeof dateStates)[number];

const emptySections = ["daily", "sleep", "activities"].map((key) => ({
  key,
  schema: `${key}-day.v1`,
  state: "empty",
  status: { ...status, collection: "covered_empty" },
  data: { items: [], has_more: false },
}));

async function dateControlsFixture(page: Page, state: DateState) {
  const failInitially: Record<DateState, string[]> = {
    "confirmed-empty(sections ready-empty)": [],
    "initial-503": ["briefing", "calendar"],
    "named-retry-success": ["calendar"],
    "partial-briefing-503": ["briefing"],
    "partial-calendar-503": ["calendar"],
    populated: [],
    "retained-failed-refetch(previous-day)": [],
    "section-unavailable(partial-dependency)": [],
  };
  const failures = new Set(failInitially[state]);
  const unknown: string[] = [];
  const requests: { method: string; path: string; status: number }[] = [];
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const kind =
      url.pathname === "/api/v1/briefing"
        ? "briefing"
        : url.pathname === "/api/v1/daily/dates"
          ? "calendar"
          : null;
    const fulfill = async (statusCode: number, json: unknown) => {
      requests.push({
        method: request.method(),
        path: `${url.pathname}${url.search}`,
        status: statusCode,
      });
      return route.fulfill({ status: statusCode, json });
    };
    if (!kind || request.method() !== "GET") {
      unknown.push(`${request.method()} ${url.pathname}`);
      return fulfill(404, { error: "Unknown date-layout fixture request" });
    }
    if (failures.has(kind))
      return fulfill(503, {
        error: "Synthetic date control source unavailable",
      });
    let sections =
      state === "confirmed-empty(sections ready-empty)"
        ? emptySections
        : populatedSections;
    if (
      state === "section-unavailable(partial-dependency)" &&
      kind === "briefing"
    )
      sections = populatedSections.map((section) =>
        section.key === "sleep"
          ? {
              ...section,
              state: "unavailable",
              data: { items: [], has_more: false },
              status: { ...status, operation: "failed" },
            }
          : section,
      );
    return fulfill(
      200,
      kind === "briefing"
        ? {
            date: url.searchParams.get("date"),
            previous_date: "2020-08-14",
            next_date: "2020-08-16",
            sections,
          }
        : ["2020-08-14", "2020-08-15"],
    );
  });
  await fakeSession(page, { setup_required: false, authenticated: true });
  return { failures, requests, unknown };
}

async function dateControlRectangles(page: Page) {
  const controls = [
    page.getByRole("button", { name: "Previous day", exact: true }),
    page.getByRole("button", { name: "Next day", exact: true }),
    page.getByRole("button", { name: /^2020-08-15/ }),
  ];
  return Promise.all(
    controls.map(async (control) => {
      const rect = await control.evaluate((node) => {
        const box = node.getBoundingClientRect();
        return {
          x: box.x,
          y: box.y,
          width: box.width,
          height: box.height,
          right: box.right,
          bottom: box.bottom,
        };
      });
      return {
        name:
          (await control.getAttribute("aria-label")) ??
          (await control.innerText()),
        ...rect,
      };
    }),
  );
}

async function expectDateControlUsable(page: Page, width: number) {
  const names = ["Previous day", "Next day"] as const;
  const controls = [
    page.getByRole("button", { name: names[0], exact: true }),
    page.getByRole("button", { name: /^2020-08-15/ }),
    page.getByRole("button", { name: names[1], exact: true }),
  ];
  const measurements = await dateControlRectangles(page);
  await test.info().attach("date-control-rectangles.json", {
    body: Buffer.from(JSON.stringify({ width, measurements }, null, 2)),
    contentType: "application/json",
  });
  for (const [index, control] of controls.entries()) {
    const size = async () =>
      control.evaluate((node) => {
        const rect = node.getBoundingClientRect();
        return { width: rect.width, height: rect.height };
      });
    let rect = await size();
    expect(
      rect.width,
      `${measurements[index].name} width at ${width}px before focus: ${JSON.stringify(rect)}`,
    ).toBeGreaterThanOrEqual(24);
    expect(
      rect.height,
      `${measurements[index].name} height at ${width}px before focus: ${JSON.stringify(rect)}`,
    ).toBeGreaterThanOrEqual(24);
    await control.focus();
    await expect(control).toBeFocused();
    rect = await size();
    expect(
      rect.width,
      `${measurements[index].name} width at ${width}px focused: ${JSON.stringify(rect)}`,
    ).toBeGreaterThanOrEqual(24);
    expect(
      rect.height,
      `${measurements[index].name} height at ${width}px focused: ${JSON.stringify(rect)}`,
    ).toBeGreaterThanOrEqual(24);
    expect(rect.width).toBeLessThanOrEqual(width);
  }
  const selectedDate = controls[1].locator(".day-main");
  await expect(selectedDate).toHaveText("2020-08-15");
  await expect
    .poll(() =>
      selectedDate.evaluate((node) => getComputedStyle(node).whiteSpace),
    )
    .toBe("nowrap");
  const dateText = await selectedDate.evaluate((node) => {
    const dateBox = node.getBoundingClientRect();
    const buttonBox = node.closest("button")!.getBoundingClientRect();
    return {
      x: dateBox.x,
      right: dateBox.right,
      buttonX: buttonBox.x,
      buttonRight: buttonBox.right,
      textWidth: node.scrollWidth,
      clientWidth: node.clientWidth,
    };
  });
  expect(dateText.x).toBeGreaterThanOrEqual(dateText.buttonX);
  expect(dateText.right).toBeLessThanOrEqual(dateText.buttonRight);
  expect(dateText.textWidth).toBeLessThanOrEqual(dateText.clientWidth);
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth))
    .toBeLessThanOrEqual(width);
  await expect
    .poll(() => page.evaluate(() => document.body.scrollWidth))
    .toBeLessThanOrEqual(width);
}

async function expectCalendarErrorContained(page: Page, width: number) {
  const panel = page.getByRole("region", { name: "Day calendar", exact: true });
  const notice = panel
    .getByRole("alert")
    .filter({ hasText: "Day calendar unavailable:" });
  const retry = panel.getByRole("button", {
    name: "Retry day calendar",
    exact: true,
  });
  await expect(notice).toBeVisible();
  await expect(retry).toBeVisible();
  const panelBox = await panel.boundingBox();
  const noticeBox = await notice.boundingBox();
  const retryBox = await retry.boundingBox();
  expect(panelBox).not.toBeNull();
  expect(noticeBox).not.toBeNull();
  expect(retryBox).not.toBeNull();
  expect(noticeBox!.x).toBeGreaterThanOrEqual(panelBox!.x);
  expect(noticeBox!.x + noticeBox!.width).toBeLessThanOrEqual(
    panelBox!.x + panelBox!.width,
  );
  expect(noticeBox!.x + noticeBox!.width).toBeLessThanOrEqual(width);
  expect(retryBox!.x).toBeGreaterThanOrEqual(noticeBox!.x);
  expect(retryBox!.x + retryBox!.width).toBeLessThanOrEqual(
    noticeBox!.x + noticeBox!.width,
  );
  expect(retryBox!.y).toBeGreaterThanOrEqual(noticeBox!.y);
  expect(retryBox!.y + retryBox!.height).toBeLessThanOrEqual(
    noticeBox!.y + noticeBox!.height,
  );
  const retrySize = async () =>
    retry.evaluate((node) => {
      const rect = node.getBoundingClientRect();
      return { width: rect.width, height: rect.height };
    });
  let retryRect = await retrySize();
  expect(retryRect.width).toBeGreaterThanOrEqual(24);
  expect(retryRect.height).toBeGreaterThanOrEqual(24);
  await retry.focus();
  await expect(retry).toBeFocused();
  retryRect = await retrySize();
  expect(retryRect.width).toBeGreaterThanOrEqual(24);
  expect(retryRect.height).toBeGreaterThanOrEqual(24);
}

for (const mode of ["light", "dark"] as const) {
  for (const state of dateStates) {
    test(`Today date controls ${mode} ${state} at 768px`, async ({ page }) => {
      const fixture = await dateControlsFixture(page, state);
      await page.setViewportSize({ width: 768, height: 900 });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      await page.goto("/?date=2020-08-15");
      await expectDateControlUsable(page, 768);
      if (state === "initial-503" || state === "partial-calendar-503") {
        await expectCalendarErrorContained(page, 768);
        if (state === "partial-calendar-503") {
          fixture.failures.delete("calendar");
          await page
            .getByRole("button", { name: "Retry day calendar" })
            .click();
          await expect(
            page.getByRole("button", { name: "Retry day calendar" }),
          ).toHaveCount(0);
        }
      }
      if (state === "partial-briefing-503")
        await expect(
          page.getByRole("button", { name: "Retry daily briefing" }),
        ).toBeVisible();
      if (state === "confirmed-empty(sections ready-empty)")
        await expect(
          page.getByRole("heading", { name: "No records for 2020-08-15." }),
        ).toBeVisible();
      if (state === "populated")
        await expect(
          page.getByRole("button", { name: /Synthetic morning run/ }),
        ).toBeVisible();
      if (state === "section-unavailable(partial-dependency)")
        await expect(
          page.getByText(/Unavailable briefing sections: sleep/),
        ).toBeVisible();
      if (state === "named-retry-success") {
        const retry = page.getByRole("button", { name: "Retry day calendar" });
        await expect(retry).toBeVisible();
        fixture.failures.delete("calendar");
        await retry.click();
        await expect(
          page.getByRole("button", { name: "Retry day calendar" }),
        ).toHaveCount(0);
      }
      if (state === "retained-failed-refetch(previous-day)") {
        fixture.failures.add("briefing");
        await page.getByRole("button", { name: "Previous day" }).click();
        await expect(page).toHaveURL(/date=2020-08-14/);
        await expect(
          page.getByText(/Could not update 2020-08-14; showing 2020-08-15/),
        ).toBeVisible();
        await expect(
          page.getByRole("button", { name: /Synthetic morning run/ }),
        ).toBeVisible();
      }
      expect(fixture.unknown).toEqual([]);
    });
  }

  for (const state of ["initial-503", "partial-calendar-503"] as const) {
    test(`Today date controls ${mode} ${state} at 1280px`, async ({ page }) => {
      const fixture = await dateControlsFixture(page, state);
      await page.setViewportSize({ width: 1280, height: 900 });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      await page.goto("/?date=2020-08-15");
      await expectDateControlUsable(page, 1280);
      await expectCalendarErrorContained(page, 1280);
      expect(fixture.unknown).toEqual([]);
    });
  }

  for (const width of [320, 375, 390, 414]) {
    test(`Today date controls ${mode} populated state at ${width}px`, async ({
      page,
    }) => {
      const fixture = await dateControlsFixture(page, "populated");
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      await page.goto("/?date=2020-08-15");
      await expectDateControlUsable(page, width);
      expect(fixture.unknown).toEqual([]);
    });
  }

  test(`Today date picker ${mode} keeps keyboard day navigation at 768px`, async ({
    page,
  }) => {
    await dateControlsFixture(page, "populated");
    await page.setViewportSize({ width: 768, height: 900 });
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    await page.goto("/?date=2020-08-15");
    await expectDateControlUsable(page, 768);
    const previous = page.getByRole("button", {
      name: "Previous day",
      exact: true,
    });
    await previous.focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/date=2020-08-14/);
    const center = page.getByRole("button", { name: /^2020-08-14/ });
    await center.click();
    const dialog = page.getByRole("dialog", { name: "Pick a day" });
    await expect(dialog).toBeVisible();
    const nextDate = dialog.locator('button[data-day="2020-08-15"]');
    await nextDate.focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/date=2020-08-15/);
    await expect(dialog).toHaveCount(0);
  });
}
