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
    expect(getThemeDefinition("grapher").implementation).toBe("curated");
    expect(hasThemeRoute(getThemeDefinition("grapher"), "today")).toBe(true);
    expect(hasThemeRoute(getThemeDefinition("grapher"), "dashboard")).toBe(
      false,
    );
    expect(getThemeDefinition("grapher").components).toMatchObject({
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      share: expect.anything(),
    });
    expect(getThemeDefinition("field-journal").implementation).toBe("preview");
    expect(hasThemeRoute(getThemeDefinition("field-journal"), "today")).toBe(
      true,
    );
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "palette-only",
      ),
    ).toHaveLength(4);
  });
});
