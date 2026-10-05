import { test, expect } from "@playwright/test";
import { installNightFixtures } from "./night-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Night ${mode} cursor failure retains rows and cursor for named retry`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.paginated = true;
    fixture.pageFailures.add("2026-08");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/night?date=2026-08");
    const table = page.getByRole("table", {
      name: "Sleep session records",
      exact: true,
    });
    await expect(
      table.getByRole("cell", { name: "2026-08-15", exact: true }),
    ).toBeVisible();
    await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
    const retry = page.getByRole("button", {
      name: "Retry more nights",
      exact: true,
    });
    await expect(retry).toBeVisible();
    await expect(
      table.getByRole("cell", { name: "2026-08-15", exact: true }),
    ).toHaveCount(1);
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toHaveCount(0);
    await retry.click();
    await expect(
      page.getByRole("button", { name: "Retry more nights", exact: true }),
    ).toBeFocused();
    fixture.pageFailures.delete("2026-08");
    await page.keyboard.press("Enter");
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toBeVisible();
    await expect(
      table.getByRole("cell", { name: "2026-08-15", exact: true }),
    ).toHaveCount(1);
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toHaveCount(1);
    await expect(
      page.getByRole("button", { name: "Load more nights", exact: true }),
    ).toHaveCount(0);
    const cursors = fixture.requests.filter((request) =>
      request.includes("cursor="),
    );
    expect(cursors).toHaveLength(3);
    expect(new Set(cursors).size).toBe(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Night ${mode} superseded cursor cannot block or error a new month`, async ({
    page,
  }) => {
    const fixture = await installNightFixtures(page);
    fixture.paginated = true;
    let releaseNew!: () => void;
    fixture.pageHolds.set(
      "2026-09",
      new Promise<void>((resolve) => {
        releaseNew = resolve;
      }),
    );
    let release!: () => void;
    fixture.pageHolds.set(
      "2026-08",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    const oldResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        url.pathname === "/api/v1/sleep" &&
        url.searchParams.get("date") === "2026-08" &&
        url.searchParams.has("cursor")
      );
    });
    await page.emulateMedia({ colorScheme: mode });
    const table = page.getByRole("table", {
      name: "Sleep session records",
      exact: true,
    });
    try {
      await page.goto("/night?date=2026-08");
      await expect(
        table.getByRole("cell", { name: "2026-08-15", exact: true }),
      ).toBeVisible();
      await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
      await expect
        .poll(
          () =>
            fixture.requests.filter((request) => request.includes("cursor="))
              .length,
        )
        .toBe(1);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("2026-09");
      await expect(
        table.getByRole("cell", { name: "2026-09-15", exact: true }),
      ).toBeVisible();
      const next = table.getByRole("cell", { name: "2026-09-14", exact: true });
      await page.locator(".theme-load-more").scrollIntoViewIfNeeded();
      await expect
        .poll(
          () =>
            fixture.requests.filter(
              (request) =>
                request.includes("cursor=") && request.includes("date=2026-09"),
            ).length,
        )
        .toBe(1);
      releaseNew();
      await expect(next).toBeVisible();
      fixture.pageFailures.add("2026-08");
    } finally {
      releaseNew();
      release();
    }
    await oldResponse;
    await expect(
      page.getByRole("button", { name: "Retry more nights", exact: true }),
    ).toHaveCount(0);
    await expect(
      table.getByRole("cell", { name: "2026-08-14", exact: true }),
    ).toHaveCount(0);
    await expect(
      table.getByRole("cell", { name: "2026-09-14", exact: true }),
    ).toHaveCount(1);
    expect(fixture.unknown).toEqual([]);
  });
}
