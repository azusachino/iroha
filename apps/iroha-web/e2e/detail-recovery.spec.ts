import { test, expect, type Page } from "@playwright/test";
import { fakeSession } from "./session";

const id = "00000000-0000-4000-8000-000000000003";
async function fixtures(page: Page) {
  const failures = new Set<string>();
  const requests: string[] = [];
  const responses: string[] = [];
  page.on("response", (response) => {
    const path = new URL(response.url()).pathname;
    if (path.startsWith("/api/v1/")) responses.push(path);
  });
  const unknown: string[] = [];
  const payloads: Record<string, unknown> = {
    [`/api/v1/media/${id}`]: {
      item: {
        id,
        title: "Synthetic detail book",
        media_type: "book",
        status: "planned",
        item_role: "standalone",
        last_update_at: "2026-08-15T12:00:00Z",
      },
      work: {
        id,
        work_kind: "book",
        primary_title: "Synthetic detail book",
        original_title: "",
        original_language: "",
        description: "",
      },
      creators: [],
      relations: [],
      events: [],
      updates: [],
    },
    [`/api/v1/sleep/${id}`]: {
      id,
      wake_date: "2026-08-15",
      started_at: "2026-08-14T22:00:00Z",
      ended_at: "2026-08-15T06:00:00Z",
      asleep_s: 25200,
      time_in_bed_s: 28800,
      efficiency: 0.875,
      deep_s: 3600,
      rem_s: 7200,
      is_main_sleep: true,
      source: "synthetic",
    },
    [`/api/v1/sleep/${id}/segments`]: [],
    [`/api/v1/activities/${id}`]: {
      id,
      title: "Synthetic detail run",
      sport_type: "run",
      started_at: "2026-08-15T12:00:00Z",
      ended_at: "2026-08-15T13:00:00Z",
      timezone: "Asia/Tokyo",
      duration_s: 3600,
      distance_m: 8000,
      source_kind: "synthetic",
    },
    [`/api/v1/activities/${id}/route`]: [],
    [`/api/v1/activities/${id}/samplings`]: [],
    [`/api/v1/activities/${id}/laps`]: [],
  };
  await page.route("**/api/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    requests.push(path);
    if (!(path in payloads)) {
      unknown.push(path);
      return route.fulfill({
        status: 404,
        json: { error: "Unknown fixture read" },
      });
    }
    return failures.has(path)
      ? route.fulfill({
          status: 503,
          json: { error: "Synthetic detail unavailable" },
        })
      : route.fulfill({ json: payloads[path] });
  });
  await fakeSession(page, {
    setup_required: false,
    authenticated: true,
    display_name: "Synthetic owner",
  });
  return { failures, requests, responses, unknown };
}
for (const mode of ["light", "dark"] as const) {
  for (const [route, api, retry, heading] of [
    ["library", "media", "Retry library item", "Synthetic detail book"],
    ["night", "sleep", "Retry sleep session", "2026-08-15"],
    ["motion", "activities", "Retry activity", "Synthetic detail run"],
  ]) {
    test(`${route} ${mode} initial detail failure has keyboard recovery and backlink`, async ({
      page,
    }) => {
      const fixture = await fixtures(page);
      fixture.failures.add(`/api/v1/${api}/${id}`);
      await page.emulateMedia({ colorScheme: mode });
      await page.goto(`/${route}/${id}`);
      const button = page.getByRole("button", { name: retry, exact: true });
      await expect(button).toBeVisible();
      await expect(
        page.getByRole("link", {
          name: new RegExp(
            `Back to ${route === "library" ? "Library" : route === "night" ? "Night" : "Motion"}`,
          ),
        }),
      ).toBeVisible();
      await button.click();
      await expect(
        page.getByRole("button", { name: retry, exact: true }),
      ).toBeFocused();
      fixture.failures.delete(`/api/v1/${api}/${id}`);
      await page.keyboard.press("Enter");
      await expect(
        page.getByRole("heading", { name: heading, exact: true }),
      ).toBeVisible();
      expect(
        fixture.responses.filter((path) => path === `/api/v1/${api}/${id}`),
      ).toHaveLength(3);
      expect(fixture.unknown).toEqual([]);
    });
  }
  test(`Night ${mode} segment failure retains session and is not no-stage evidence`, async ({
    page,
  }) => {
    const fixture = await fixtures(page);
    fixture.failures.add(`/api/v1/sleep/${id}/segments`);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto(`/night/${id}`);
    await expect(
      page.getByRole("heading", { name: "2026-08-15", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("This session has no stage samples.", { exact: true }),
    ).toHaveCount(0);
    const retry = page.getByRole("button", {
      name: "Retry sleep stages",
      exact: true,
    });
    await expect(retry).toBeVisible();
    fixture.failures.clear();
    await retry.click();
    await expect(
      page
        .getByText("This session has no stage samples.", { exact: true })
        .first(),
    ).toBeVisible();
    expect(
      fixture.responses.filter((path) => path === `/api/v1/sleep/${id}`),
    ).toHaveLength(1);
    expect(fixture.unknown).toEqual([]);
  });
  test(`Motion ${mode} independent stream failures retain record and recover separately`, async ({
    page,
  }) => {
    const fixture = await fixtures(page);
    for (const stream of ["route", "samplings", "laps"])
      fixture.failures.add(`/api/v1/activities/${id}/${stream}`);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto(`/motion/${id}`);
    await expect(
      page.getByRole("heading", { name: "Synthetic detail run", exact: true }),
    ).toBeVisible();
    for (const name of [
      "Retry activity route",
      "Retry heart-rate samples",
      "Retry activity laps",
    ])
      await expect(
        page.getByRole("button", { name, exact: true }),
      ).toBeVisible();
    fixture.failures.delete(`/api/v1/activities/${id}/route`);
    await page
      .getByRole("button", { name: "Retry activity route", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Retry activity route", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", {
        name: "Retry heart-rate samples",
        exact: true,
      }),
    ).toBeVisible();
    expect(
      fixture.responses.filter((path) => path === `/api/v1/activities/${id}`),
    ).toHaveLength(1);
    expect(
      fixture.responses.filter((path) => path.endsWith("/samplings")),
    ).toHaveLength(1);
    expect(fixture.unknown).toEqual([]);
  });
}
