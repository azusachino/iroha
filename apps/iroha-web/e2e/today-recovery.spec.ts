import { test, expect, type Page } from "@playwright/test";
import { fakeSession } from "./session";
async function fixtures(page: Page) {
  const failures = new Set<string>();
  const requests: string[] = [];
  const unknown: string[] = [];
  const state = { populated: false, unavailable: false };
  const task = {
    id: "today-a",
    title: "Synthetic daily task",
    status: "open",
    due_at: "2020-08-15T12:00:00Z",
    created_at: "2020-08-15T00:00:00Z",
    updated_at: "2020-08-15T00:00:00Z",
    completed_at: null,
  };
  await page.route("**/api/v1/**", (route) => {
    const url = new URL(route.request().url());
    requests.push(url.pathname + url.search);
    const kind =
      url.pathname === "/api/v1/briefing"
        ? "briefing"
        : url.pathname === "/api/v1/daily/dates"
          ? "calendar"
          : null;
    if (!kind) {
      unknown.push(url.pathname);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown Today fixture" },
      });
    }
    if (failures.has(kind))
      return route.fulfill({
        status: 503,
        json: { error: "Synthetic Today unavailable" },
      });
    return route.fulfill({
      json:
        kind === "briefing"
          ? {
              date: url.searchParams.get("date"),
              previous_date: "2020-08-14",
              next_date: "2020-08-16",
              sections: state.unavailable
                ? [
                    {
                      key: "daily",
                      state: "ready",
                      data: {
                        date: url.searchParams.get("date"),
                        items: [{ steps: 1234, ring: null }],
                      },
                    },
                    ...["activities", "sleep", "media"].map((key) => ({
                      key,
                      state: "unavailable",
                      data: null,
                    })),
                  ]
                : [],
            }
          : ["2020-08-15"],
    });
  });
  await fakeSession(page, { setup_required: false, authenticated: true });
  return { failures, requests, unknown, state };
}
for (const mode of ["light", "dark"] as const) {
  for (const [read, label] of [
    ["briefing", "Retry daily briefing"],
    ["calendar", "Retry day calendar"],
  ]) {
    test(`Today ${mode} ${read} failure has independent named recovery`, async ({
      page,
    }) => {
      const fixture = await fixtures(page);
      fixture.failures.add(read);
      await page.emulateMedia({ colorScheme: mode });
      await page.goto("/?date=2020-08-15");
      const retry = page.getByRole("button", { name: label, exact: true });
      await expect(retry).toBeVisible();
      await retry.click();
      await expect(
        page.getByRole("button", { name: label, exact: true }),
      ).toBeFocused();
      fixture.failures.delete(read);
      await page.keyboard.press("Enter");
      await expect(
        page.getByRole("button", { name: label, exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("heading", {
          name: "No records for 2020-08-15.",
          exact: true,
        }),
      ).toBeVisible();
      for (const [kind, path] of [
        ["briefing", "/api/v1/briefing"],
        ["calendar", "/api/v1/daily/dates"],
      ])
        expect(
          fixture.requests.filter((request) => request.startsWith(path)),
        ).toHaveLength(kind === read ? 3 : 1);
      expect(fixture.unknown).toEqual([]);
    });
  }
  test(`Today ${mode} modifiers and focused widget arrows do not scrub days`, async ({
    page,
  }) => {
    await fixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/?date=2020-08-15&keep=yes");
    await expect(
      page.getByRole("heading", {
        name: "No records for 2020-08-15.",
        exact: true,
      }),
    ).toBeVisible();
    await page.locator("body").click({ position: { x: 2, y: 2 } });
    await page.keyboard.press("Control+ArrowLeft");
    await expect(page).toHaveURL(/date=2020-08-15/);
    await page
      .getByRole("button", { name: "Previous day", exact: true })
      .focus();
    await page.keyboard.press("ArrowLeft");
    await expect(page).toHaveURL(/date=2020-08-15/);
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/date=2020-08-14/);
    await expect(page).toHaveURL(/keep=yes/);
    await page.locator("main button[aria-haspopup='dialog']").click();
    const dialog = page.getByRole("dialog", {
      name: "Pick a day",
      exact: true,
    });
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button").first().focus();
    await page.keyboard.press("ArrowLeft");
    await expect(page).toHaveURL(/date=2020-08-14/);
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
  });
  test(`Today ${mode} partial unavailable sections never claim confirmed empty`, async ({
    page,
  }) => {
    const fixture = await fixtures(page);
    fixture.state.unavailable = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/?date=2020-08-15");
    await expect(
      page.getByText(/Unavailable briefing sections: activities, sleep, media/),
    ).toBeVisible();
    for (const text of [
      "No activity sessions recorded for this day.",
      "No sleep session recorded for this day.",
      "No exact media sessions recorded for this day.",
    ])
      await expect(page.getByText(text, { exact: true })).toHaveCount(0);
    for (const text of [
      "Activity sessions unavailable for this day.",
      "Sleep session unavailable for this day.",
      "Media sessions unavailable for this day.",
    ])
      await expect(page.getByText(text, { exact: true })).toBeVisible();
    await expect(
      page.getByRole("heading", {
        name: "No records for 2020-08-15.",
        exact: true,
      }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });
}
