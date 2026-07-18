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

  it("does not present palette-only entries as complete themes", () => {
    expect(
      THEME_DEFINITIONS.every(
        (theme) => theme.implementation === "palette-only",
      ),
    ).toBe(true);
    expect(getThemeDefinition("grapher").description).toContain("evidence");
  });
});
