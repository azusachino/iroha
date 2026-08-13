import { describe, expect, it } from "vitest";
import { canonicalMonth, monthBounds, shiftMonth } from "@iroha/shared/month";

describe("shared month helpers", () => {
  it("accepts only canonical YYYY-MM values", () => {
    expect(canonicalMonth("2026-02", "2026-01")).toBe("2026-02");
    expect(canonicalMonth("2026-2", "2026-01")).toBe("2026-01");
    expect(canonicalMonth("2026-13", "2026-01")).toBe("2026-01");
  });
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
