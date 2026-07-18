import { describe, expect, it } from "vitest";
import { THEME_DEFINITIONS, getThemeDefinition } from "./registry";

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

  it("marks only themes with a complete renderer set as curated", () => {
    expect(getThemeDefinition("grapher").implementation).toBe("curated");
    expect(getThemeDefinition("grapher").components).toMatchObject({
      today: expect.anything(),
      daily: expect.anything(),
      activities: expect.anything(),
      sleep: expect.anything(),
      share: expect.anything(),
    });
    expect(
      THEME_DEFINITIONS.filter(
        (theme) => theme.implementation === "palette-only",
      ),
    ).toHaveLength(5);
  });
});
