import { test, expect } from "@playwright/test";
import { installLibraryFixtures } from "./library-fixtures";

for (const mode of ["light", "dark"] as const) {
  test(`Library ${mode} failed initial reads do not invent titles or empty history`, async ({
    page,
  }, info) => {
    const fixture = await installLibraryFixtures(page);
    fixture.failures.add("records");
    fixture.failures.add("aggregates");
    await page.setViewportSize({ width: 320, height: 900 });
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByText("No completion records.", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByText("No titles match this selection.", { exact: true }),
    ).toHaveCount(0);
    await expect(page.locator(".media-count")).toContainText("—");
    await expect(
      page.getByRole("button", { name: "Retry library totals", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Retry library records", exact: true }),
    ).toBeVisible();
    for (const name of ["Retry library totals", "Retry library records"]) {
      const retry = page.getByRole("button", { name, exact: true });
      let reached = false;
      for (let tab = 0; tab < 30; tab++) {
        await page.keyboard.press("Tab");
        if (
          await retry.evaluate((element) => element === document.activeElement)
        ) {
          reached = true;
          break;
        }
      }
      expect(reached, `${name} reachable by Tab at 320`).toBe(true);
    }
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: info.outputPath("library-unavailable.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });
  test(`Library ${mode} records failure preserves independently available totals`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.failures.add("records");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(page.locator(".media-count")).toContainText("3");
    await expect(
      page.getByText("No completion records.", { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Retry library records", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} deferred first reads never announce zero or empty`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("records", held);
    fixture.holds.set("aggregates", held);
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/library");
      await expect(
        page.getByText("Loading library totals…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Loading library records…", { exact: true }),
      ).toBeVisible();
      await expect(
        page
          .getByRole("region", { name: "Library summary" })
          .getByText("—", { exact: true }),
      ).toHaveCount(4);
      await expect(
        page.getByText("No titles match this selection.", { exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Books", exact: true }),
      ).toBeEnabled();
    } finally {
      release();
    }
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toBeVisible();
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} confirmed empty stays ready for quiet filters`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.empty = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByText("No titles match this selection.", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("No completion records.", { exact: true }),
    ).toBeVisible();
    await expect(page.locator(".media-count")).toContainText("0");
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("records", held);
    fixture.holds.set("aggregates", held);
    try {
      await page.getByRole("button", { name: "Books", exact: true }).click();
      await expect(
        page.getByText("Updating library records…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Updating library totals…", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("combobox", { name: "Status", exact: true }),
      ).toBeEnabled();
      await expect(
        page.getByText("Loading library records…", { exact: true }),
      ).toHaveCount(0);
    } finally {
      release();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} failed totals retain records and recover keyboard focus`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.failures.add("aggregates");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toContainText("5/20 chapters");
    await expect(page.locator(".media-count")).toContainText("—");
    fixture.failures.delete("aggregates");
    await page
      .getByRole("button", { name: "Retry library totals", exact: true })
      .press("Enter");
    await expect(page.locator(".media-count")).toContainText("3");
    await expect(
      page.getByRole("region", {
        name: "Library totals read state",
        exact: true,
      }),
    ).toBeFocused();
    expect(
      fixture.requests.filter((value) => value.startsWith("/api/v1/media?")),
    ).toHaveLength(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} records retry leaves failed totals alone`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.failures.add("records");
    fixture.failures.add("aggregates");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByRole("button", { name: "Retry library records", exact: true }),
    ).toBeVisible();
    fixture.failures.delete("records");
    await page
      .getByRole("button", { name: "Retry library records", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("region", {
        name: "Library records read state",
        exact: true,
      }),
    ).toBeFocused();
    await expect(
      page.getByRole("button", { name: "Retry library totals", exact: true }),
    ).toBeVisible();
    expect(
      fixture.requests.filter((value) =>
        value.startsWith("/api/v1/media/aggregates"),
      ),
    ).toHaveLength(1);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} failed family change retains independent observed scopes`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.paginated = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "records",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page.getByRole("button", { name: "Books", exact: true }).click();
      await expect(
        page.getByRole("region", { name: "Library totals read state" }),
      ).toContainText("Observed totals: book · All statuses · Lifetime");
      await expect(
        page.getByRole("region", { name: "Library records read state" }),
      ).toContainText("Observed records: All kinds · All statuses · Lifetime");
      await expect(page.locator(".media-count")).toContainText("1");
      fixture.failures.add("records");
    } finally {
      release();
      fixture.holds.delete("records");
    }
    await expect(
      page.getByRole("button", { name: "Retry library records", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Load more", exact: true }),
    ).toHaveCount(0);
    fixture.failures.delete("records");
    await page
      .getByRole("button", { name: "Retry library records", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} stale filter reads cannot resurrect old rows or totals`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByRole("link", { name: "Synthetic game", exact: true }),
    ).toBeVisible();
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    fixture.holds.set("records", held);
    fixture.holds.set("aggregates", held);
    try {
      await page.getByRole("button", { name: "Anime", exact: true }).click();
      await expect
        .poll(
          () =>
            fixture.requests.filter((value) => value.includes("family=anime"))
              .length,
        )
        .toBe(2);
      fixture.holds.clear();
      await page.getByRole("button", { name: "Books", exact: true }).click();
      await expect(
        page.getByRole("region", { name: "Library records read state" }),
      ).toContainText("Observed records: book");
      const old = page.waitForResponse(
        (response) =>
          response.url().includes("family=anime") &&
          new URL(response.url()).pathname === "/api/v1/media/aggregates",
      );
      release();
      await old;
      await page.evaluate(() => new Promise(requestAnimationFrame));
      await expect(
        page.getByRole("link", { name: "Synthetic anime", exact: true }),
      ).toHaveCount(0);
      await expect(
        page.getByRole("region", { name: "Library totals read state" }),
      ).toContainText("Observed totals: book");
    } finally {
      release();
      fixture.holds.clear();
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} status year lifetime queries preserve facet count semantics`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(page.locator(".media-count")).toContainText("3");
    await page
      .getByRole("combobox", { name: "Completed year", exact: true })
      .selectOption("2025");
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toHaveCount(0);
    await page
      .getByRole("combobox", { name: "Status", exact: true })
      .selectOption("completed");
    await expect(
      page.getByText("No titles match this selection.", { exact: true }),
    ).toBeVisible();
    await expect(
      page
        .locator(".stats > div")
        .filter({ has: page.getByText("In progress", { exact: true }) }),
    ).toContainText("1");
    await expect(
      page
        .locator(".stats > div")
        .filter({ has: page.getByText("In progress", { exact: true }) }),
    ).toContainText(
      "Counts across statuses · Observed facet: All kinds · All statuses · 2025",
    );
    expect(fixture.requests).toContain(
      "/api/v1/media?status=completed&completed_year=2025&limit=100",
    );
    expect(fixture.requests).toContain(
      "/api/v1/media/aggregates?status=completed&completed_year=2025",
    );
    await page
      .getByRole("combobox", { name: "Completed year", exact: true })
      .selectOption("");
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toBeVisible();
    expect(fixture.requests).toContain(
      "/api/v1/media?status=completed&limit=100",
    );
    expect(
      fixture.requests.every(
        (value) => !value.includes("timezone=") && !value.includes("date="),
      ),
    ).toBe(true);
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} genuine rated zero stays zero and links remain exact`, async ({
    page,
  }, info) => {
    const fixture = await installLibraryFixtures(page);
    fixture.zeroRating = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page
        .locator(".stats > div")
        .filter({ has: page.getByText("Average score", { exact: true }) }),
    ).toContainText("0.0");
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toHaveAttribute("href", "/library/00000000-0000-4000-8000-000000000003");
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toContainText("5/20 chapters");
    await page.screenshot({
      path: info.outputPath("library-populated.png"),
      fullPage: true,
    });
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} pending old append cannot corrupt or block a new page`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.paginated = true;
    let release!: () => void;
    fixture.pageHold = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.emulateMedia({ colorScheme: mode });
    try {
      await page.goto("/library");
      await expect(
        page.getByRole("link", { name: "Synthetic anime", exact: true }),
      ).toBeVisible();
      await page
        .getByRole("button", { name: "Load more", exact: true })
        .click();
      await expect
        .poll(
          () =>
            fixture.requests.filter((value) =>
              value.includes("cursor=synthetic-next"),
            ).length,
        )
        .toBe(1);
      await page.getByRole("button", { name: "Books", exact: true }).click();
      await expect(
        page.getByRole("link", { name: "Synthetic book", exact: true }),
      ).toBeVisible();
      await page.getByRole("button", { name: "All", exact: true }).click();
      await expect(
        page.getByRole("region", { name: "Library records read state" }),
      ).toContainText("Observed records: All kinds");
      await page
        .getByRole("button", { name: "Load more", exact: true })
        .click();
      await expect
        .poll(
          () =>
            fixture.requests.filter((value) =>
              value.includes("cursor=synthetic-next"),
            ).length,
        )
        .toBe(2);
      release();
      await expect(
        page.getByRole("link", { name: "Synthetic game", exact: true }),
      ).toHaveCount(1);
      await expect(
        page.getByRole("link", { name: "Synthetic book", exact: true }),
      ).toHaveCount(1);
      await expect(
        page.getByRole("link", { name: "Synthetic anime", exact: true }),
      ).toHaveCount(1);
    } finally {
      release();
      fixture.pageHold = null;
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} retry respects a newer focus target`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.failures.add("aggregates");
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(
      page.getByRole("button", { name: "Retry library totals", exact: true }),
    ).toBeVisible();
    let release!: () => void;
    fixture.holds.set(
      "aggregates",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    fixture.failures.delete("aggregates");
    try {
      await page
        .getByRole("button", { name: "Retry library totals", exact: true })
        .press("Enter");
      const status = page.getByRole("combobox", {
        name: "Status",
        exact: true,
      });
      await status.focus();
      release();
      await expect(page.locator(".media-count")).toContainText("3");
      await expect(status).toBeFocused();
    } finally {
      release();
      fixture.holds.delete("aggregates");
    }
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} failed totals refresh keeps its observed scope while records recover`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(page.locator(".media-count")).toContainText("3");
    let release!: () => void;
    fixture.holds.set(
      "aggregates",
      new Promise<void>((resolve) => {
        release = resolve;
      }),
    );
    try {
      await page.getByRole("button", { name: "Books", exact: true }).click();
      await expect(
        page.getByRole("region", { name: "Library records read state" }),
      ).toContainText("Observed records: book");
      await expect(
        page.getByRole("region", { name: "Library totals read state" }),
      ).toContainText("Observed totals: All kinds");
      await expect(page.locator(".media-count")).toContainText("3");
      fixture.failures.add("aggregates");
    } finally {
      release();
      fixture.holds.delete("aggregates");
    }
    await expect(
      page.getByRole("button", { name: "Retry library totals", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toBeVisible();
    await expect(page.locator(".media-count")).toContainText("3");
    fixture.failures.delete("aggregates");
    await page
      .getByRole("button", { name: "Retry library totals", exact: true })
      .press("Enter");
    await expect(
      page.getByRole("region", { name: "Library totals read state" }),
    ).toContainText("Observed totals: book");
    await expect(page.locator(".media-count")).toContainText("1");
    expect(fixture.unknown).toEqual([]);
  });

  test(`Library ${mode} movie manga and paused facets preserve historical completion`, async ({
    page,
  }) => {
    const fixture = await installLibraryFixtures(page);
    fixture.animeType = "movie";
    fixture.bookType = "manga";
    fixture.paused = true;
    await page.emulateMedia({ colorScheme: mode });
    await page.goto("/library");
    await expect(page.locator(".media-count")).toContainText("3");
    await expect(
      page
        .locator(".stats > div")
        .filter({ has: page.getByText("In progress", { exact: true }) })
        .locator("strong"),
    ).toHaveText("0");
    await page.getByRole("button", { name: "Anime", exact: true }).click();
    await expect(
      page.getByRole("link", { name: "Synthetic anime", exact: true }),
    ).toContainText("Movie");
    await expect(page.locator(".media-count")).toContainText("1");
    await page
      .getByRole("button", { name: "Manga & light novels", exact: true })
      .click();
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toContainText("Manga");
    await page
      .getByRole("combobox", { name: "Completed year", exact: true })
      .selectOption("2025");
    await expect(
      page.getByRole("link", { name: "Synthetic book", exact: true }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("combobox", { name: "Completed year", exact: true })
        .locator("option[value='2024']"),
    ).toHaveCount(0);
    expect(fixture.requests).toContain(
      "/api/v1/media?family=manga_book&completed_year=2025&limit=100",
    );
    expect(fixture.unknown).toEqual([]);
  });
}
