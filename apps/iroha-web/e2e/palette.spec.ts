import { expect, test } from "@playwright/test";
import { categoryColor } from "@iroha/shared/domain/category-color";
import { sportColor } from "@iroha/shared/domain/sport";
import { installPilotFixtures } from "./pilot-fixtures";
import { measurePilotPage } from "./pilot-measurements";

const categories = [
  "food",
  "groceries",
  "transport",
  "shopping",
  "housing",
  "utilities",
  "health",
  "entertainment",
  "subscriptions",
  "work",
  "other",
];
const sports = ["run", "walk", "hike", "ride", "swim", "other"];
const publicBase = process.env.E2E_PUBLIC_BASE_URL ?? "http://127.0.0.1:5184";
for (const mode of ["light", "dark"] as const) {
  test(`Shared palette ${mode} has distinct readable identities on both hosts`, async ({
    page,
  }) => {
    await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
    const privateFixture = await installPilotFixtures(page, "expenses");
    await page.goto("/expenses?date=2026-08&currency=JPY");
    await expect(
      page.getByRole("heading", { name: "Synthetic cafe", exact: true }),
    ).toBeVisible();
    const tokens = [
      ...categories.map(categoryColor),
      ...sports.map(sportColor),
    ];
    const readPalette = () =>
      page.evaluate((tokens) => {
        const probe = document.createElement("span");
        document.body.append(probe);
        const colors = tokens.map((color) => {
          probe.style.color = color ?? "";
          return getComputedStyle(probe).color;
        });
        probe.remove();
        return colors;
      }, tokens);
    const checkContrast = async () => {
      for (const surface of ["--bg", "--surface", "--surface-2"]) {
        await page.evaluate(
          ({ tokens, surface }) => {
            const fixture = document.createElement("div");
            fixture.id = "palette-contrast-fixture";
            fixture.style.backgroundColor = `var(${surface})`;
            for (const [index, color] of tokens.entries()) {
              const label = document.createElement("span");
              label.style.color = color ?? "";
              label.style.display = "block";
              label.textContent = `Palette fixture ${index}`;
              fixture.append(label);
            }
            document.body.append(fixture);
          },
          { tokens, surface },
        );
        const measured = await page.evaluate(measurePilotPage);
        const pairs = measured.textPairs.filter((pair) =>
          pair.text.startsWith("Palette fixture "),
        );
        expect(pairs.length).toBe(tokens.length);
        for (const pair of pairs)
          expect(pair.ratio).toBeGreaterThanOrEqual(4.5);
        await page
          .locator("#palette-contrast-fixture")
          .evaluate((el) => el.remove());
      }
    };
    const privateColors = await readPalette();
    await checkContrast();
    const privateText = await page.evaluate(measurePilotPage);
    const deletion = privateText.textPairs.filter(
      (pair) => pair.text === "Delete",
    );
    expect(deletion.length).toBeGreaterThan(0);
    for (const pair of deletion)
      expect(pair.ratio).toBeGreaterThanOrEqual(pair.required);
    await page.unrouteAll({ behavior: "wait" });
    const publicFixture = await installPilotFixtures(page, "public");
    await page.goto(publicBase);
    await expect(
      page.getByRole("heading", { name: "Activities", exact: true }),
    ).toBeVisible();
    const publicColors = await readPalette();
    await checkContrast();
    await page.evaluate((mode) => {
      document.documentElement.dataset.theme = mode;
    }, mode);
    await page.emulateMedia({
      colorScheme: mode === "light" ? "dark" : "light",
    });
    expect(await readPalette()).toEqual(publicColors);
    expect(publicColors).toEqual(privateColors);
    expect(new Set(privateColors.slice(0, categories.length)).size).toBe(
      categories.length,
    );
    expect(new Set(privateColors.slice(categories.length)).size).toBe(
      sports.length,
    );
    const publicText = await page.evaluate(measurePilotPage);
    const links = publicText.textPairs.filter(
      (pair) => pair.text === "2026" || pair.text.includes("iroha"),
    );
    expect(links.length).toBeGreaterThan(0);
    for (const pair of links)
      expect(pair.ratio).toBeGreaterThanOrEqual(pair.required);
    expect(privateFixture.unknown).toEqual([]);
    expect(publicFixture.unknown).toEqual([]);
  });
}
