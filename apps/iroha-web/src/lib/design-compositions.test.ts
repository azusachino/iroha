import { describe, expect, it } from "vitest";
import {
  DESIGN_COMPOSITIONS,
  designComposition,
  isDesignComposition,
} from "@iroha/shared/design-compositions";

describe("design workshop compositions", () => {
  it("keeps every implemented layout addressable", () => {
    expect(DESIGN_COMPOSITIONS.map((composition) => composition.id)).toEqual([
      "editorial",
      "command",
      "chronicle",
      "cover",
      "workspace",
      "journal",
      "quiet",
    ]);
    expect(
      new Set(DESIGN_COMPOSITIONS.map((composition) => composition.label)).size,
    ).toBe(DESIGN_COMPOSITIONS.length);
  });

  it("normalizes unknown URL values to the first implementation", () => {
    expect(designComposition("chronicle")).toBe("chronicle");
    expect(designComposition("not-a-composition")).toBe("editorial");
    expect(isDesignComposition("quiet")).toBe(true);
    expect(isDesignComposition("not-a-composition")).toBe(false);
  });
});
