import { describe, expect, it } from "vitest";
import { buildMonthlyDistanceSeries } from "@iroha/shared/theme-ui/grapher/monthly-distance";

describe("buildMonthlyDistanceSeries", () => {
  it("uses a calendar-month window and fills unreported months with zero", () => {
    const buckets = [
      { key: "2026-08", distance_m: 150_000 },
      { key: "2025-12", distance_m: 125_000 },
      { key: "2025-10", distance_m: 100_000 },
    ];
    const series = buildMonthlyDistanceSeries(buckets, "2026-09", 12);

    expect(series).toHaveLength(12);
    expect(series[0].key).toBe("2025-10");
    expect(series.at(-1)?.key).toBe("2026-09");
    expect(series.find(({ key }) => key === "2025-11")?.distance_m).toBe(0);
    expect(series.find(({ key }) => key === "2025-12")?.distance_m).toBe(
      125_000,
    );
    expect(series.find(({ key }) => key === "2026-09")?.distance_m).toBe(0);
  });

  it("supports each chart period as that many calendar months", () => {
    for (const [months, firstMonth] of [
      [6, "2026-04"],
      [12, "2025-10"],
      [24, "2024-10"],
    ] as const) {
      const series = buildMonthlyDistanceSeries(
        [{ key: "2026-08", distance_m: 150_000 }],
        "2026-09",
        months,
      );

      expect(series).toHaveLength(months);
      expect(series[0].key).toBe(firstMonth);
      expect(series.at(-1)?.key).toBe("2026-09");
    }
  });

  it("returns no points when the archive has no monthly evidence", () => {
    expect(buildMonthlyDistanceSeries([], "2026-09", 12)).toEqual([]);
  });
});
