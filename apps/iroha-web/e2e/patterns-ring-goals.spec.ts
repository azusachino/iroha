import { expect, test } from "@playwright/test";
import { installPatternsFixtures } from "./patterns-fixtures";

const row = (day: string, goals: boolean) => ({
  id: `synthetic-${day}`,
  day,
  ring: {
    move_kcal: 100,
    move_goal_kcal: goals ? 300 : 0,
    exercise_min: 10,
    exercise_goal_min: goals ? 30 : 0,
    stand_hours: 4,
    stand_goal_hours: goals ? 10 : 0,
  },
  source: "synthetic",
});

test("Patterns uses dated last-observed goals when the latest reading has none", async ({
  page,
}) => {
  await installPatternsFixtures(page);
  await page.route("**/api/v1/daily?**", async (route) => {
    const limit = new URL(route.request().url()).searchParams.get("limit");
    if (limit !== "1" && limit !== "100") return route.fallback();
    const latest = row("2026-09-14", false);
    return route.fulfill({
      json: {
        items: limit === "1" ? [latest] : [latest, row("2026-09-10", true)],
        has_more: false,
        next_cursor: null,
      },
    });
  });
  await page.goto("/patterns");
  const gauge = page.getByRole("img", { name: "Activity rings" });
  await expect(gauge.locator("circle.arc")).toHaveCount(3);
  await expect(
    page.getByText("Missing goals last recorded 2026-09-10"),
  ).toBeVisible();
  const arcs = await gauge
    .locator("circle.arc")
    .evaluateAll((nodes) =>
      nodes.map((node) => parseFloat(getComputedStyle(node).strokeDasharray)),
    );
  expect(arcs.every((value) => value > 0)).toBe(true);
});

test("Patterns identifies missing goals when none are available", async ({
  page,
}) => {
  await installPatternsFixtures(page);
  await page.route("**/api/v1/daily?**", async (route) => {
    const limit = new URL(route.request().url()).searchParams.get("limit");
    if (limit !== "1" && limit !== "100") return route.fallback();
    return route.fulfill({
      json: {
        items: [row("2026-09-14", false)],
        has_more: false,
        next_cursor: null,
      },
    });
  });
  await page.goto("/patterns");
  const gauge = page.getByRole("img", { name: "Activity rings" });
  await expect(gauge).toBeVisible();
  await expect(gauge.locator("circle.arc")).toHaveCount(0);
  await expect(page.getByText("goal unavailable")).toHaveCount(3);
  await expect(page.getByText(/Missing goals last recorded/)).toHaveCount(0);
});
