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
        : url.pathname === "/api/v1/tasks"
          ? "tasks"
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
          : kind === "calendar"
            ? ["2020-08-15"]
            : state.populated
              ? [task]
              : [],
    });
  });
  await fakeSession(page, { setup_required: false, authenticated: true });
  return { failures, requests, unknown, state, task };
}
for (const mode of ["light", "dark"] as const) {
  for (const [read, label] of [
    ["briefing", "Retry daily briefing"],
    ["tasks", "Retry daily tasks"],
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
      if (read === "tasks")
        await expect(
          page.getByText("No open tasks for this day.", { exact: true }),
        ).toHaveCount(0);
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
        ["tasks", "/api/v1/tasks"],
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
  test(`Today ${mode} completion failure preserves observed task and recovers`, async ({
    page,
  }) => {
    const fixture = await fixtures(page);
    fixture.state.populated = true;
    let writes = 0;
    await page.route("**/api/v1/tasks/today-a", (route) => {
      expect(route.request().method()).toBe("PATCH");
      return route.fulfill(
        ++writes === 1
          ? { status: 503, json: { error: "Synthetic completion failure" } }
          : { json: { ...fixture.task, status: "completed" } },
      );
    });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/?date=2020-08-15");
    await expect(
      page.getByText("Observed tasks: 2020-08-15 · up to 5 open tasks", {
        exact: true,
      }),
    ).toBeVisible();
    const complete = page.getByRole("button", {
      name: "Complete Synthetic daily task",
      exact: true,
    });
    await complete.click();
    await expect(
      page.getByRole("alert").filter({ hasText: "Task completion failed" }),
    ).toBeVisible();
    await expect(complete).toBeVisible();
    await complete.focus();
    await page.keyboard.press("Enter");
    await expect(complete).toHaveCount(0);
    await expect(
      page.getByRole("alert").filter({ hasText: "Task completion failed" }),
    ).toHaveCount(0);
    expect(writes).toBe(2);
    expect(fixture.unknown).toEqual([]);
  });
  test(`Today ${mode} completion cannot drop a pending next-day inventory`, async ({
    page,
  }) => {
    const fixture = await fixtures(page);
    fixture.state.populated = true;
    let releaseWrite!: () => void;
    const writeHold = new Promise<void>((resolve) => {
      releaseWrite = resolve;
    });
    let releaseRead!: () => void;
    const readHold = new Promise<void>((resolve) => {
      releaseRead = resolve;
    });
    let writes = 0;
    let nextReads = 0;
    await page.route("**/api/v1/tasks/today-a", async (route) => {
      writes++;
      await writeHold;
      await route.fulfill({ json: { ...fixture.task, status: "completed" } });
    });
    await page.route("**/api/v1/tasks?**", async (route) => {
      if (
        new URL(route.request().url()).searchParams.get("due") !== "2020-08-16"
      )
        return route.fallback();
      if (++nextReads === 1) {
        await readHold;
        await route.fulfill({ status: 503, json: { error: "Stale day read" } });
      } else
        await route.fulfill({
          json: [
            { ...fixture.task, id: "day-two", title: "Next-day inventory" },
          ],
        });
    });
    try {
      await page.emulateMedia({ colorScheme: mode });
      await page.goto("/?date=2020-08-15");
      await page
        .getByRole("button", {
          name: "Complete Synthetic daily task",
          exact: true,
        })
        .click();
      await expect.poll(() => writes).toBe(1);
      await expect(
        page.getByRole("button", {
          name: "Complete Synthetic daily task",
          exact: true,
        }),
      ).toBeDisabled();
      await expect(
        page.getByRole("button", {
          name: "Complete Synthetic daily task",
          exact: true,
        }),
      ).toHaveAttribute("aria-busy", "true");
      await page.getByRole("button", { name: "Next day", exact: true }).click();
      await expect.poll(() => nextReads).toBe(1);
      releaseWrite();
      await expect(
        page.getByRole("button", {
          name: "Complete Next-day inventory",
          exact: true,
        }),
      ).toBeVisible();
      await expect(
        page.getByText("Observed tasks: 2020-08-16 · up to 5 open tasks", {
          exact: true,
        }),
      ).toBeVisible();
      const oldResponse = page.waitForResponse(
        (response) =>
          response.url().includes("due=2020-08-16") &&
          response.status() === 503,
      );
      releaseRead();
      await oldResponse;
      await expect(
        page.getByRole("button", {
          name: "Complete Next-day inventory",
          exact: true,
        }),
      ).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Retry daily tasks", exact: true }),
      ).toHaveCount(0);
      expect(nextReads).toBe(2);
      expect(writes).toBe(1);
      expect(fixture.unknown).toEqual([]);
    } finally {
      releaseWrite();
      releaseRead();
    }
  });
}
