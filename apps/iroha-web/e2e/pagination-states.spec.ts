import { test, expect } from "@playwright/test";
import { installMotionFixtures } from "./motion-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Motion ${mode} cursor failure keeps rows and offers exact retry`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
    fixture.paginated = true;
    fixture.pageFailures.add("2026-08");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/motion?date=2026-08");
    const first = page.getByRole("link", {
      name: "Synthetic run 2026-08",
      exact: true,
    });
    await expect(first).toBeVisible();
    await page.getByTestId("motion-load-sentinel").scrollIntoViewIfNeeded();
    const retry = page.getByRole("button", {
      name: "Retry more activities",
      exact: true,
    });
    await expect(retry).toBeVisible();
    await expect(first).toBeVisible();
    await expect(first).toHaveAttribute("href", "/motion/synthetic-2026-08");
    await expect(
      page.getByRole("link", {
        name: "Synthetic next run 2026-08",
        exact: true,
      }),
    ).toHaveCount(0);
    await retry.click();
    await expect(
      page.getByRole("button", { name: "Retry more activities", exact: true }),
    ).toBeFocused();
    fixture.pageFailures.delete("2026-08");
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("link", {
        name: "Synthetic next run 2026-08",
        exact: true,
      }),
    ).toBeVisible();
    await expect(first).toHaveCount(1);
    await expect(
      page.getByRole("link", {
        name: "Synthetic next run 2026-08",
        exact: true,
      }),
    ).toHaveCount(1);
    await expect(
      page.getByRole("button", { name: "Load more rows", exact: true }),
    ).toHaveCount(0);
    const cursors = fixture.requests.filter((request) =>
      request.includes("cursor="),
    );
    expect(cursors).toHaveLength(3);
    expect(new Set(cursors).size).toBe(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Motion ${mode} old pending cursor cannot block or fail a newer scope`, async ({
    page,
  }) => {
    const fixture = await installMotionFixtures(page);
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
        url.pathname === "/api/v1/activities" &&
        url.searchParams.get("date") === "2026-08" &&
        url.searchParams.has("cursor")
      );
    });
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/motion?date=2026-08");
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-08", exact: true }),
      ).toBeVisible();
      await page.getByTestId("motion-load-sentinel").scrollIntoViewIfNeeded();
      await expect
        .poll(
          () =>
            fixture.requests.filter((request) => request.includes("cursor="))
              .length,
        )
        .toBe(1);
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page.getByRole("link", { name: "Synthetic run 2026-09", exact: true }),
      ).toBeVisible();
      const next = page.getByRole("link", {
        name: "Synthetic next run 2026-09",
        exact: true,
      });
      await page.getByTestId("motion-load-sentinel").scrollIntoViewIfNeeded();
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
      page.getByRole("button", { name: "Retry more activities", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("link", {
        name: "Synthetic next run 2026-08",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("link", {
        name: "Synthetic next run 2026-09",
        exact: true,
      }),
    ).toHaveCount(1);
    expect(fixture.unknown).toEqual([]);
  });
}
