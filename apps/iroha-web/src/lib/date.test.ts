import { describe, expect, it } from "vitest";
import { todayInTimezone } from "@iroha/shared/date";

describe("canonical default timezone", () => {
  it("uses Tokyo for the application calendar day", () => {
    expect(todayInTimezone(new Date("2026-08-14T15:00:00Z"))).toBe(
      "2026-08-15",
    );
  });

  it("keeps explicit timezone conversion available to machine callers", () => {
    expect(todayInTimezone(new Date("2026-08-14T15:00:00Z"), "UTC")).toBe(
      "2026-08-14",
    );
  });
});
