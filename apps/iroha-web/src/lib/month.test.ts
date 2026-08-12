import { describe, expect, it } from "vitest";
import { monthBounds, shiftMonth } from "@iroha/shared/month";

describe("shared month helpers", () => {
  it("returns an exclusive upper bound for API month filters", () => {
    expect(monthBounds("2026-02")).toEqual({
      from: "2026-02-01",
      to: "2026-03-01",
    });
  });

  it("moves across year boundaries", () => {
    expect(shiftMonth("2026-01", -1)).toBe("2025-12");
    expect(shiftMonth("2025-12", 1)).toBe("2026-01");
  });
});
