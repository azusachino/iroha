import { test, expect } from "@playwright/test";
import { installTogoFixtures } from "./togo-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`To-go ${mode} task failure is not successful empty and jobs remain visible`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("tasks");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.getByText("Nothing pressing. Add one thing worth carrying.", {
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("AniList sync");
    await expect(
      page.getByRole("button", { name: "Retry tasks", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} jobs failure is not empty history and tasks remain usable`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("jobs");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(page.getByRole("alert").first()).toBeVisible();
    await expect(
      page.getByText("No jobs have run yet.", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByText("Synthetic intention", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry jobs", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} confirmed empty tasks and jobs show legitimate empty states`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.tasks = [];
    fixture.jobs = [];
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByText("Nothing pressing. Add one thing worth carrying.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      page.getByText("No jobs have run yet.", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("0 open in loaded tasks", { exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} pending tasks keep form and independently loaded jobs usable`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    let release!: () => void;
    fixture.holds.set(
      "tasks",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/to-go");
      await expect(
        page.getByText("Loading tasks…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Nothing pressing. Add one thing worth carrying.", {
          exact: true,
        }),
      ).toHaveCount(0);
      await expect(
        page.getByText("0 open in loaded tasks", { exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("region", { name: "Recent syncs", exact: true }),
      ).toContainText("AniList sync");
      await page
        .getByRole("textbox", { name: "New task", exact: true })
        .fill("Keep typed intention");
      expect(
        await page
          .getByRole("textbox", { name: "New task", exact: true })
          .evaluate((node) => node.closest("[inert]")),
      ).toBeNull();
    } finally {
      release();
    }
    await expect(
      page.getByText("Synthetic intention", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("textbox", { name: "New task", exact: true }),
    ).toHaveValue("Keep typed intention");
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} repeated keyboard task retry recovers only tasks`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("tasks");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    const retry = page.getByRole("button", {
      name: "Retry tasks",
      exact: true,
    });
    await expect(retry).toBeVisible();
    const jobs = fixture.requests.filter((request) =>
      request.startsWith("GET /api/v1/jobs?"),
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
      page.getByRole("button", { name: "Retry tasks", exact: true }),
    ).toBeFocused();
    fixture.failures.delete("tasks");
    await page.keyboard.press("Enter");
    await expect(
      page.getByText("Synthetic intention", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "To-go list", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((request) =>
        request.startsWith("GET /api/v1/jobs?"),
      ).length,
    ).toBe(jobs);
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} failed add retains typed fields and survives unrelated read and finish`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("add");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByText("Synthetic intention", { exact: true }),
    ).toBeVisible();
    await page
      .getByRole("textbox", { name: "New task", exact: true })
      .fill("Typed next step");
    await page
      .getByRole("textbox", { name: "Task notes", exact: true })
      .fill("Retain context");
    await page
      .getByRole("spinbutton", { name: "Task priority", exact: true })
      .fill("7");
    await page.getByRole("button", { name: "Add", exact: true }).click();
    await expect(page.getByRole("alert")).toContainText("Could not add task:");
    await page
      .getByRole("button", { name: "Refresh jobs", exact: true })
      .click();
    await page
      .getByRole("button", {
        name: "Complete Synthetic intention",
        exact: true,
      })
      .click();
    await expect(
      page.getByText("0 open in loaded tasks", { exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toContainText("Could not add task:");
    await expect(
      page.getByRole("textbox", { name: "New task", exact: true }),
    ).toHaveValue("Typed next step");
    await expect(
      page.getByRole("textbox", { name: "Task notes", exact: true }),
    ).toHaveValue("Retain context");
    await expect(
      page.getByRole("spinbutton", { name: "Task priority", exact: true }),
    ).toHaveValue("7");
    fixture.failures.delete("add");
    await page.getByRole("button", { name: "Add", exact: true }).click();
    await expect(
      page.getByText("Typed next step", { exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(
      page.getByRole("textbox", { name: "New task", exact: true }),
    ).toHaveValue("");
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} completion failure does not clear on jobs refresh`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("finish");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await page
      .getByRole("button", {
        name: "Complete Synthetic intention",
        exact: true,
      })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "Could not complete Synthetic intention:",
    );
    await page
      .getByRole("button", { name: "Refresh jobs", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "Could not complete Synthetic intention:",
    );
    await expect(
      page.getByText("Synthetic intention", { exact: true }),
    ).toBeVisible();
    fixture.failures.delete("finish");
    await page
      .getByRole("button", {
        name: "Complete Synthetic intention",
        exact: true,
      })
      .click();
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(
      page.getByText("Completed today", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} completed today uses completion date not due date`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.tasks = [
      {
        ...fixture.tasks[0],
        id: "synthetic-future",
        title: "Future due completed today",
        status: "completed",
        due_date: "2026-10-06",
        completed_at: "2026-10-03T16:00:00Z",
      },
      {
        ...fixture.tasks[0],
        id: "synthetic-old",
        title: "Due today completed yesterday",
        status: "completed",
        due_date: "2026-10-04",
        completed_at: "2026-10-02T16:00:00Z",
      },
    ];
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByText("Future due completed today · p2", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Due today completed yesterday · p2", { exact: true }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} action error survives successful jobs refresh`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("action");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    const action = page.getByRole("button", {
      name: "AniList Refresh anime and manga",
      exact: true,
    });
    await expect(action).toBeEnabled();
    await action.click();
    await expect(page.getByRole("alert")).toContainText(
      "Could not start action:",
    );
    const count = fixture.requests.filter((request) =>
      request.startsWith("GET /api/v1/jobs?"),
    ).length;
    await page
      .getByRole("button", { name: "Refresh jobs", exact: true })
      .click();
    await expect
      .poll(
        () =>
          fixture.requests.filter((request) =>
            request.startsWith("GET /api/v1/jobs?"),
          ).length,
      )
      .toBe(count + 1);
    await expect(page.getByRole("alert")).toContainText(
      "Could not start action:",
    );
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} initial and refreshed jobs exclude importer work`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.jobs = [
      ...fixture.jobs,
      {
        ...fixture.jobs[0],
        id: "synthetic-import",
        kind: "import_fit",
        status: "running",
      },
    ];
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("AniList sync");
    await page
      .getByRole("button", { name: "Refresh jobs", exact: true })
      .click();
    await expect
      .poll(
        () =>
          fixture.requests.filter((request) =>
            request.startsWith("GET /api/v1/jobs?"),
          ).length,
      )
      .toBe(2);
    await expect(page.getByText("import fit", { exact: true })).toHaveCount(0);
    expect(
      fixture.requests
        .filter((request) => request.startsWith("GET /api/v1/jobs?"))
        .every(
          (request) =>
            new URL(request.slice(4), "http://fixture").searchParams.get(
              "kind",
            ) === "media_sync_anilist,media_sync_bangumi,media_bridge_refresh",
        ),
    ).toBe(true);
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} poll failure retains jobs and retry keeps its own error identity`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.jobs = [{ ...fixture.jobs[0], status: "running" }];
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("running · 1/3 attempts");
    fixture.failures.add("jobs");
    await page.clock.runFor(4000);
    await expect(
      page.getByRole("button", { name: "Retry job polling", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("running · 1/3 attempts");
    await page
      .getByRole("button", { name: "Retry job polling", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Retry job polling", exact: true }),
    ).toBeFocused();
    fixture.failures.delete("jobs");
    fixture.jobs = [{ ...fixture.jobs[0], status: "completed" }];
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("button", { name: "Retry job polling", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("completed · 1/3 attempts");
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toBeFocused();
    expect(
      fixture.requests
        .filter((request) => request.startsWith("GET /api/v1/jobs?"))
        .every((request) =>
          new URL(request.slice(4), "http://fixture").searchParams.has("kind"),
        ),
    ).toBe(true);
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} pending action is single-shot and old jobs cannot overwrite its receipt`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    const action = page.getByRole("button", {
      name: "AniList Refresh anime and manga",
      exact: true,
    });
    await expect(action).toBeEnabled();
    let releaseAction!: () => void;
    let releaseRead!: () => void;
    fixture.holds.set(
      "action",
      new Promise<void>((resolve) => {
        releaseAction = resolve;
      }),
    );
    fixture.holds.set(
      "jobs:2",
      new Promise<void>((resolve) => {
        releaseRead = resolve;
      }),
    );
    const oldResponse = page.waitForResponse(
      (response) => new URL(response.url()).pathname === "/api/v1/jobs",
    );
    try {
      await page
        .getByRole("button", { name: "Refresh jobs", exact: true })
        .click();
      await action.click();
      await expect(
        page.getByRole("button", {
          name: "AniList Refresh anime and manga",
          exact: true,
        }),
      ).toBeDisabled();
      expect(
        fixture.requests.filter((request) =>
          request.startsWith("POST /api/v1/actions/"),
        ).length,
      ).toBe(1);
      releaseAction();
      await expect(
        page.getByRole("region", { name: "Recent syncs", exact: true }),
      ).toContainText("queued · 0/3 attempts");
    } finally {
      releaseAction();
      releaseRead();
    }
    await oldResponse;
    await expect(
      page.getByRole("region", { name: "Recent syncs", exact: true }),
    ).toContainText("queued · 0/3 attempts");
    await expect(
      page.getByRole("button", {
        name: "AniList Sync already running",
        exact: true,
      }),
    ).toBeDisabled();
    expect(fixture.unknown).toEqual([]);
  });

  test(`To-go ${mode} successful add cannot certify a failing full task inventory`, async ({
    page,
  }) => {
    const fixture = await installTogoFixtures(page);
    fixture.failures.add("tasks");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/to-go");
    await expect(
      page.getByRole("button", { name: "Retry tasks", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("textbox", { name: "New task", exact: true })
      .fill("Known created task");
    await page.getByRole("button", { name: "Add", exact: true }).click();
    await expect(
      page.getByText("Known created task added.", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry tasks", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Task count unavailable", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Nothing pressing. Add one thing worth carrying.", {
        exact: true,
      }),
    ).toHaveCount(0);
    fixture.failures.delete("tasks");
    await page
      .getByRole("button", { name: "Retry tasks", exact: true })
      .click();
    await expect(
      page.getByText("Known created task", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("2 open in loaded tasks", { exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });
}
