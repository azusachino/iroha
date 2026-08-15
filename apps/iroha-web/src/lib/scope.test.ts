import { describe, expect, it } from "vitest";
import {
  currentCalendarScope,
  isFutureScope,
  parseCalendarScope,
  readCalendarScope,
  scopeBounds,
  scopeFromParts,
  scopeParts,
  serializeCalendarScope,
  shiftCalendarScope,
  writeCalendarScope,
} from "@iroha/shared/scope";

const NOW = new Date("2026-08-15T03:00:00Z");

describe("calendar scope", () => {
  it("parses only canonical year, month, and day values", () => {
    expect(parseCalendarScope("2026")).toEqual({ kind: "year", year: 2026 });
    expect(parseCalendarScope("2026-08")).toEqual({
      kind: "month",
      year: 2026,
      month: 8,
    });
    expect(parseCalendarScope("2026-08-15")).toEqual({
      kind: "day",
      year: 2026,
      month: 8,
      day: 15,
    });
    expect(parseCalendarScope("2026-8")).toBeNull();
    expect(parseCalendarScope("0000-08")).toBeNull();
    expect(parseCalendarScope("2026-02-30")).toBeNull();
  });

  it("round-trips URL state and removes legacy aliases", () => {
    const params = new URLSearchParams("month=2026-08&year=2026");
    const scope = readCalendarScope(params, {
      fallback: { kind: "month", year: 2026, month: 7 },
    });
    expect(scope).toEqual({ kind: "month", year: 2026, month: 8 });

    writeCalendarScope(params, { kind: "day", year: 2026, month: 8, day: 15 });
    expect(params.toString()).toBe("date=2026-08-15");
    expect(serializeCalendarScope(scopeFromParts("2026", "8"))).toBe("2026-08");
  });

  it("resolves half-open calendar bounds without local timezone math", () => {
    expect(scopeBounds({ kind: "month", year: 2026, month: 8 })).toEqual({
      from: "2026-08-01",
      to: "2026-09-01",
    });
    expect(scopeBounds({ kind: "day", year: 2026, month: 8, day: 15 })).toEqual(
      {
        from: "2026-08-15",
        to: "2026-08-16",
      },
    );
    expect(scopeBounds({ kind: "lifetime" })).toBeNull();
  });

  it("uses the supplied timezone for current and future scope decisions", () => {
    expect(currentCalendarScope("day", NOW, "America/New_York")).toEqual({
      kind: "day",
      year: 2026,
      month: 8,
      day: 14,
    });
    expect(
      isFutureScope({ kind: "day", year: 2026, month: 8, day: 16 }, NOW, "UTC"),
    ).toBe(true);
    expect(
      isFutureScope(
        { kind: "day", year: 2026, month: 8, day: 14 },
        NOW,
        "America/New_York",
      ),
    ).toBe(false);
  });

  it("shifts scopes and clamps the forward edge to the configured timezone", () => {
    expect(
      shiftCalendarScope(
        { kind: "month", year: 2026, month: 8 },
        1,
        NOW,
        "Asia/Tokyo",
      ),
    ).toEqual({ kind: "month", year: 2026, month: 8 });
    expect(
      shiftCalendarScope(
        { kind: "month", year: 2026, month: 8 },
        -1,
        NOW,
        "Asia/Tokyo",
      ),
    ).toEqual({ kind: "month", year: 2026, month: 7 });
  });

  it("clamps shifts to a supplied data range, tighter than the plain future guard", () => {
    const bounds = { min: "2019-03-02", max: "2026-06-20" };
    // Without bounds this would land on August 2026 (today's month, not
    // future) -- bounds.max is earlier than today, so it wins.
    expect(
      shiftCalendarScope(
        { kind: "month", year: 2026, month: 7 },
        1,
        NOW,
        "UTC",
        bounds,
      ),
    ).toEqual({ kind: "month", year: 2026, month: 6 });
    // Backward shifts, previously unclamped entirely, now stop at bounds.min.
    expect(
      shiftCalendarScope({ kind: "year", year: 2019 }, -1, NOW, "UTC", bounds),
    ).toEqual({ kind: "year", year: 2019 });
    // An ordinary shift fully inside the range is untouched.
    expect(
      shiftCalendarScope({ kind: "year", year: 2021 }, -1, NOW, "UTC", bounds),
    ).toEqual({ kind: "year", year: 2020 });
    // Omitting bounds entirely preserves the pre-existing, unclamped-backward
    // behavior for every current call site.
    expect(
      shiftCalendarScope({ kind: "year", year: 2019 }, -1, NOW, "UTC"),
    ).toEqual({ kind: "year", year: 2018 });
  });

  it("represents lifetime explicitly without treating it as a date", () => {
    const params = new URLSearchParams("scope=lifetime");
    expect(
      readCalendarScope(params, {
        fallback: { kind: "month", year: 2026, month: 8 },
      }),
    ).toEqual({ kind: "lifetime" });
    writeCalendarScope(params, { kind: "lifetime" });
    expect(params.toString()).toBe("scope=lifetime");
    expect(scopeParts({ kind: "lifetime" })).toEqual({ year: "", month: "" });
  });
});
