import { describe, expect, it } from "vitest";
import { formatCanonicalMonth } from "@iroha/shared/format/format";
import {
  canonicalMonth,
  currentYear,
  MONTH_OPTIONS,
  monthBounds,
  monthOptionsInRange,
  shiftMonth,
  shiftMonthWithin,
  yearOptions,
} from "@iroha/shared/format/month";

describe("shared month helpers", () => {
  it("accepts only canonical YYYY-MM values", () => {
    expect(canonicalMonth("2026-02", "2026-01")).toBe("2026-02");
    expect(canonicalMonth("2026-2", "2026-01")).toBe("2026-01");
    expect(canonicalMonth("2026-13", "2026-01")).toBe("2026-01");
  });

  it("keeps month labels canonical for shared controls", () => {
    expect(formatCanonicalMonth("2026-2")).toBe("2026-02");
    expect(formatCanonicalMonth("2026-13")).toBe("2026-13");
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

  it("does not move period navigation past the current month", () => {
    expect(shiftMonthWithin("2026-07", 1, "2026-08")).toBe("2026-08");
    expect(shiftMonthWithin("2026-08", 1, "2026-08")).toBe("2026-08");
    expect(shiftMonthWithin("2026-08", -1, "2026-08")).toBe("2026-07");
  });

  it("provides one shared month and descending year vocabulary", () => {
    expect(MONTH_OPTIONS).toHaveLength(12);
    expect(MONTH_OPTIONS[0]).toEqual({ value: "1", label: "January" });
    expect(yearOptions(2024, 2026)).toEqual(["2026", "2025", "2024"]);
  });

  it("derives the canonical current year without local formatting", () => {
    expect(currentYear(new Date("2026-08-14T12:00:00+09:00"))).toBe("2026");
  });

  it("lists months newest first, matching yearOptionsInRange's convention", () => {
    const bounds = { min: "2024-06-01", max: "2026-08-31" };
    expect(
      monthOptionsInRange("2026", bounds).map((option) => option.value),
    ).toEqual(["8", "7", "6", "5", "4", "3", "2", "1"]);
    // A boundary year clips to the real min/max month, still newest first.
    expect(
      monthOptionsInRange("2024", bounds).map((option) => option.value),
    ).toEqual(["12", "11", "10", "9", "8", "7", "6"]);
  });
});
