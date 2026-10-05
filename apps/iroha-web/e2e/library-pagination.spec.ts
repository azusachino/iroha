import { test, expect } from "@playwright/test";
import { installLibraryFixtures } from "./library-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Library ${mode} failed cursor preserves rows and provides named recovery`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.paginated = true;
    fixture.pageFailures.add("all");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    const records = page.getByRole("region", {
      name: "Library records read state",
      exact: true,
    });
    await expect(records).toContainText("Synthetic anime");
    await page.getByRole("button", { name: "Load more", exact: true }).click();
    const retry = page.getByRole("button", {
      name: "Retry more titles",
      exact: true,
    });
    await expect(retry).toBeVisible();
    await expect(records).toContainText("Synthetic anime");
    await expect(
      records.getByText("Synthetic book", { exact: true }),
    ).toHaveCount(0);
    await retry.click();
    await expect(
      page.getByRole("button", { name: "Retry more titles", exact: true }),
    ).toBeFocused();
    fixture.pageFailures.delete("all");
    await page.keyboard.press("Enter");
    await expect(
      records.getByText("Synthetic book", { exact: true }),
    ).toBeVisible();
    await expect(
      records.getByText("Synthetic game", { exact: true }),
    ).toBeVisible();
    await expect(
      records.getByText("Synthetic anime", { exact: true }),
    ).toHaveCount(1);
    await expect(
      records.getByText("Synthetic book", { exact: true }),
    ).toHaveCount(1);
    await expect(
      records.getByText("Synthetic game", { exact: true }),
    ).toHaveCount(1);
    await expect(
      page.getByRole("button", { name: "Load more", exact: true }),
    ).toHaveCount(0);
    await expect(records).toBeFocused();
    const cursors = fixture.requests.filter((request) =>
      request.includes("cursor="),
    );
    expect(cursors).toHaveLength(3);
    expect(new Set(cursors).size).toBe(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} old cursor cannot revive titles or errors under a newer family`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.paginated = true;
    let release!: () => void;
    fixture.pageHolds.set(
      "all",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    const oldResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        url.pathname === "/api/v1/media" &&
        url.searchParams.has("cursor") &&
        !url.searchParams.has("family")
      );
    });
    await page.emulateMedia({ colorScheme: mode });
    const records = page.getByRole("region", {
      name: "Library records read state",
      exact: true,
    });
    try {
      await page.goto("/library");
      await expect(records).toContainText("Synthetic anime");
      await page
        .getByRole("button", { name: "Load more", exact: true })
        .click();
      await expect
        .poll(
          () =>
            fixture.requests.filter((request) => request.includes("cursor="))
              .length,
        )
        .toBe(1);
      await page.getByRole("button", { name: "Games", exact: true }).click();
      await expect(
        records.getByText("Synthetic game", { exact: true }),
      ).toBeVisible();
      fixture.pageFailures.add("all");
    } finally {
      release();
    }
    await oldResponse;
    await expect(
      page.getByRole("button", { name: "Retry more titles", exact: true }),
    ).toHaveCount(0);
    await expect(
      records.getByText("Synthetic anime", { exact: true }),
    ).toHaveCount(0);
    await expect(
      records.getByText("Synthetic book", { exact: true }),
    ).toHaveCount(0);
    await expect(
      records.getByText("Synthetic game", { exact: true }),
    ).toHaveCount(1);
    expect(fixture.unknown).toEqual([]);
  });
}
