import { describe, expect, it } from "vitest";
import { THEME_ROUTES } from "@iroha/shared/themes";
import {
  THEME_DEFINITIONS,
  getThemeDefinition,
  hasThemeRoute,
} from "./registry";

describe("Iroha theme registry", () => {
  it("keeps the six language identities explicit", () => {
    expect(THEME_DEFINITIONS.map((theme) => theme.identity.id)).toEqual([
      "atlas",
      "grapher",
      "field-journal",
      "phenology",
      "sound-map",
      "archive",
    ]);
  });

  it("requires every theme to own its period control appearance", () => {
    expect(
      THEME_DEFINITIONS.map(
        (theme) => theme.primitives.periodControl.appearance,
      ),
    ).toEqual(THEME_DEFINITIONS.map((theme) => theme.identity.id));
  });

  it("keeps the curated page lenses and visual marks in the shared manifest", () => {
    for (const theme of THEME_DEFINITIONS) {
      expect(theme.identity.mark).toBeTruthy();
      expect(theme.identity.swatch).toMatch(/^#[0-9a-f]{6}$/i);
      for (const lens of Object.values(theme.identity.lenses)) {
        expect(lens.question).toBeTruthy();
        expect(lens.lead).toBeTruthy();
        expect(lens.time).toBeTruthy();
        expect(lens.interaction).toBeTruthy();
        expect(lens.detail).toBeTruthy();
        expect(lens.avoid).toBeTruthy();
      }
    }
    expect(
      new Set(
        THEME_DEFINITIONS.map((theme) => theme.identity.lenses.expenses.lead),
      ).size,
    ).toBe(THEME_DEFINITIONS.length);
    expect(
      new Set(
        THEME_DEFINITIONS.map((theme) => theme.identity.lenses.reports.lead),
      ).size,
    ).toBe(THEME_DEFINITIONS.length);
  });

  it("requires every curated theme to own every page renderer", () => {
    expect(THEME_DEFINITIONS.every((theme) => theme.components?.shell)).toBe(
      true,
    );
    const routes = [
      "today",
      "daily",
      "activities",
      "sleep",
      "media",
      "dashboard",
      "activity-detail",
      "media-detail",
    ] as const;
    for (const theme of THEME_DEFINITIONS) {
      for (const route of routes) {
        expect(
          hasThemeRoute(theme, route),
          `${theme.identity.id}/${route}`,
        ).toBe(true);
      }
    }
    expect(getThemeDefinition("grapher").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("grapher"), "today")).toBe(true);
    expect(getThemeDefinition("grapher").components).toMatchObject({
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(getThemeDefinition("atlas").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("atlas"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("atlas"), "daily")).toBe(true);
    expect(getThemeDefinition("atlas").components).toMatchObject({
      shell: expect.anything(),
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(getThemeDefinition("field-journal").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("field-journal"), "today")).toBe(
      true,
    );
    expect(hasThemeRoute(getThemeDefinition("field-journal"), "daily")).toBe(
      true,
    );
    expect(getThemeDefinition("field-journal").components).toMatchObject({
      shell: expect.anything(),
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(getThemeDefinition("phenology").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("phenology"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("phenology"), "daily")).toBe(true);
    expect(getThemeDefinition("phenology").components).toMatchObject({
      shell: expect.anything(),
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(getThemeDefinition("sound-map").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("sound-map"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("sound-map"), "daily")).toBe(true);
    expect(getThemeDefinition("sound-map").components).toMatchObject({
      shell: expect.anything(),
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(getThemeDefinition("archive").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("archive"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("archive"), "daily")).toBe(true);
    expect(getThemeDefinition("archive").components).toMatchObject({
      shell: expect.anything(),
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      media: expect.anything(),
      dashboard: expect.anything(),
      "activity-detail": expect.anything(),
      "media-detail": expect.anything(),
    });
    expect(
      THEME_DEFINITIONS.filter((theme) => theme.implementation === "preview"),
    ).toHaveLength(0);
    expect(
      THEME_DEFINITIONS.every((theme) => hasThemeRoute(theme, "expenses")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.every((theme) => hasThemeRoute(theme, "reports")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.every((theme) => hasThemeRoute(theme, "metrics")),
    ).toBe(true);
  });

  it("keeps report composition ownership distinct across languages", () => {
    const reportComponents = THEME_DEFINITIONS.map(
      (theme) => theme.components.reports,
    );
    expect(reportComponents.every(Boolean)).toBe(true);
    expect(new Set(reportComponents).size).toBe(THEME_DEFINITIONS.length);
  });

  it("does not permit a partial production route tree", () => {
    for (const theme of THEME_DEFINITIONS) {
      expect(theme.implementation).toBe("curated");
      for (const route of THEME_ROUTES) {
        expect(theme.components[route]).toEqual(expect.anything());
      }
    }
  });

  it("keeps expense composition ownership distinct across languages", () => {
    const expenseComponents = THEME_DEFINITIONS.map(
      (theme) => theme.components.expenses,
    );
    expect(expenseComponents.every(Boolean)).toBe(true);
    expect(new Set(expenseComponents).size).toBe(THEME_DEFINITIONS.length);
  });
});
