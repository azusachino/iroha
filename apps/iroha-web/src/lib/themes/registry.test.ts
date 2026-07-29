import { describe, expect, it } from "vitest";
import {
  THEME_DEFINITIONS,
  getThemeDefinition,
  hasThemeRoute,
} from "./registry";

describe("Iroha theme registry", () => {
  it("keeps the six language identities explicit", () => {
    expect(THEME_DEFINITIONS.map((theme) => theme.id)).toEqual([
      "atlas",
      "grapher",
      "field-journal",
      "phenology",
      "sound-map",
      "archive",
    ]);
  });

  it("distinguishes complete and preview renderer sets", () => {
    expect(THEME_DEFINITIONS.every((theme) => theme.components?.shell)).toBe(
      true,
    );
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "today")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "daily")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "activities")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "sleep")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "media")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "dashboard")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "activity-detail")),
    ).toBe(true);
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "preview",
      ).every((theme) => hasThemeRoute(theme, "media-detail")),
    ).toBe(true);
    expect(getThemeDefinition("grapher").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("grapher"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("grapher"), "dashboard")).toBe(
      false,
    );
    expect(
      hasThemeRoute(getThemeDefinition("grapher"), "activity-detail"),
    ).toBe(false);
    expect(hasThemeRoute(getThemeDefinition("grapher"), "media-detail")).toBe(
      false,
    );
    expect(getThemeDefinition("grapher").components).toMatchObject({
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
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
  });
});
