import { expect, test } from "@playwright/test";
import { installPilotFixtures } from "./pilot-fixtures";

for (const mode of ["light", "dark"] as const) {
  for (const pilot of ["overview", "expenses"] as const) {
    test(`${pilot} ${mode} labels deferred first load and recovers repeated failures`, async ({
      page,
    }) => {
      await page.setViewportSize({ width: 320, height: 900 });
      await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
      const fixture = await installPilotFixtures(page, pilot);
      const path =
        pilot === "overview"
          ? "**/api/v1/activities/overview?**"
          : "**/api/v1/expenses?**";
      let release!: () => void;
      const gate = new Promise<void>((resolve) => {
        release = resolve;
      });
      let failures = 2;
      await page.route(path, async (route) => {
        await gate;
        if (failures-- > 0)
          await route.fulfill({
            status: 503,
            json: { error: "synthetic failure" },
          });
        else await route.fallback();
      });
      try {
        await page.goto(
          pilot === "overview" ? "/overview" : "/expenses?date=2026-08",
        );
        await expect(
          page.getByRole("status").filter({
            hasText:
              pilot === "overview"
                ? "Loading overview…"
                : "Loading expense records…",
          }),
        ).toBeVisible();
        await expect(
          page.getByRole("button", { name: /Synthetic (run|cafe)/ }),
        ).toHaveCount(0);
      } finally {
        release();
      }
      const retry = page.getByRole("button", {
        name: pilot === "overview" ? "Retry overview" : "Refresh",
        exact: true,
      });
      for (let attempt = 0; attempt < 2; attempt++) {
        await expect(page.getByRole("alert").first()).toBeVisible();
        await retry.focus();
        await page.keyboard.press("Enter");
      }
      await expect(page.getByRole("alert")).toHaveCount(0);
      if (pilot === "overview")
        await expect(
          page.getByText("20.50 km", { exact: true }).first(),
        ).toBeVisible();
      else
        await expect(
          page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
        ).toBeVisible();
      expect(fixture.unknown).toEqual([]);
    });
  }

  test(`Expenses ${mode} keeps loaded records accessible during a deferred scope change`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const fixture = await installPilotFixtures(page, "expenses");
    await page.goto("/expenses?date=2026-08");
    await expect(
      page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
    ).toBeVisible();
    let release!: () => void;
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route("**/api/v1/expenses?**", async (route) => {
      await gate;
      await route.fallback();
    });
    try {
      await page
        .getByRole("combobox", { name: "Filter by month" })
        .selectOption("9");
      await expect(
        page
          .getByRole("region", { name: "Expense records read", exact: true })
          .getByRole("status")
          .filter({ hasText: "Updating…" }),
      ).toBeVisible();
      await expect(
        page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
      ).toBeVisible();
      const selection = page.getByRole("region", {
        name: "Expense period selection",
        exact: true,
      });
      await expect(
        selection.getByText("Observed period: 2026-08", { exact: true }),
      ).toBeVisible();
      await expect(
        selection.getByText("Observed period: 2026-09", { exact: true }),
      ).toHaveCount(0);
      const record = page.getByRole("button", { name: /Synthetic cafe/ });
      await record.focus();
      await expect(record).toBeFocused();
    } finally {
      release();
    }
    await expect(
      page.getByRole("heading", { name: "Synthetic shop", exact: true }),
    ).toBeVisible();
    await expect(
      page
        .getByRole("region", { name: "Expense period selection", exact: true })
        .getByText("Observed period: 2026-09", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: /Synthetic cafe/ }),
    ).toHaveCount(0);
    await expect(page.getByText("Updating…", { exact: true })).toHaveCount(0);
    expect(fixture.unknown).toEqual([]);
  });
}
