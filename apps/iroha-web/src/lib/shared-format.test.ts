import { describe, expect, it } from "vitest";
import * as shared from "@iroha/shared/format/format";
import * as privateFormat from "./format";

const DASH = "—";

describe("shared Grapher quantity formatting", () => {
  it("groups deterministically, preserves zero and supports an explicit locale", () => {
    expect(shared.formatMetricValue(12345, "count")).toBe("12,345");
    expect(shared.formatMetricValue(0, "count")).toBe("0");
    expect(shared.formatMetricValue(12345.6, "km", "de-DE")).toBe("12.345,6");
    expect(shared.formatDistance(12345000)).toBe("12,345.00 km");
  });

  it.each([undefined, null, NaN, Infinity, -Infinity])(
    "does not fabricate quantities for %s",
    (value) => {
      expect(shared.formatMetricValue(value, "count")).toBe(DASH);
      expect(shared.formatDistance(value)).toBe(DASH);
      expect(shared.formatDuration(value)).toBe(DASH);
      expect(shared.formatHumanDuration(value)).toBe(DASH);
      expect(shared.formatPace(value)).toBe(DASH);
      expect(shared.formatElevation(value)).toBe(DASH);
      expect(shared.formatHr(value)).toBe(DASH);
    },
  );

  it("distinguishes human summaries from exact clock values", () => {
    expect(shared.formatHumanDuration(3720)).toBe("1 h 02 min");
    expect(shared.formatHumanDuration(2700)).toBe("45 min");
    expect(shared.formatHumanDuration(0)).toBe("0 s");
    expect(shared.formatHumanDuration(59.6)).toBe("1 min");
    expect(shared.formatHumanDuration(3599.6)).toBe("1 h 00 min");
    expect(shared.formatDuration(3720)).toBe("1:02:00");
  });

  it("rounds pace before splitting minutes and seconds", () => {
    expect(shared.formatPace(59.6)).toBe("1:00 /km");
    expect(shared.formatPace(359.6)).toBe("6:00 /km");
    expect(shared.formatPace(0)).toBe(DASH);
    expect(shared.formatPace(-1)).toBe(DASH);
    expect(shared.formatHumanDuration(-1)).toBe(DASH);
    expect(shared.formatDuration(-1)).toBe(DASH);
  });

  it("preserves explicit timezone behavior in both hosts", () => {
    const instant = "2026-08-14T15:00:00Z";
    expect(shared.formatDate(instant, "UTC")).toBe("2026-08-14 15:00:00");
    expect(shared.formatDate(instant, "Asia/Tokyo")).toBe(
      "2026-08-15 00:00:00",
    );
    expect(shared.formatDateOnly(instant, "Asia/Tokyo")).toBe("2026-08-15");
    expect(shared.formatDateOnly("2026-08-14", "Asia/Tokyo")).toBe(
      "2026-08-14",
    );
    expect(privateFormat.formatDate(instant, "UTC")).toBe(
      shared.formatDate(instant, "UTC"),
    );
    expect(privateFormat.formatDateOnly(instant, "Asia/Tokyo")).toBe(
      shared.formatDateOnly(instant, "Asia/Tokyo"),
    );
    expect(privateFormat.formatDateShort(instant, "UTC")).toBe("Aug 14");
    expect(privateFormat.formatDateShort(instant, "Asia/Tokyo")).toBe("Aug 15");
    expect(privateFormat.formatDateShort(undefined)).toBe(DASH);
    expect(privateFormat.formatDateShort("not-a-date")).toBe("not-a-date");
    expect(shared.formatDate(undefined)).toBe(DASH);
    expect(shared.formatDateOnly(undefined)).toBe(DASH);
    expect(shared.formatDate("not-a-date")).toBe("not-a-date");
    expect(shared.formatDateOnly("not-a-date")).toBe("not-a-date");
  });

  it("keeps the private host on the same non-timezone helpers", () => {
    expect(privateFormat.formatMetricValue).toBe(shared.formatMetricValue);
    expect(privateFormat.formatDistance).toBe(shared.formatDistance);
    expect(privateFormat.formatDuration).toBe(shared.formatDuration);
    expect(privateFormat.formatPace).toBe(shared.formatPace);
    expect(privateFormat.formatElevation).toBe(shared.formatElevation);
    expect(privateFormat.formatHr).toBe(shared.formatHr);
  });
});
