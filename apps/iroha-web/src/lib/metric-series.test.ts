import { describe, expect, it } from "vitest";
import {
  metricSeriesCsv,
  pointHasValue,
  pointValue,
} from "@iroha/shared/metric-series";

describe("shared metric series helpers", () => {
  it("keeps null periods distinct from zero", () => {
    expect(
      pointValue({ period: "2026-01", value: null, observed_days: 0 }),
    ).toBeNull();
    expect(pointValue({ period: "2026-02", value: 0, observed_days: 1 })).toBe(
      0,
    );
    expect(
      pointHasValue({ period: "2026-01", value: null, observed_days: 0 }),
    ).toBe(false);
  });

  it("exports dimensions and values as lossless CSV cells", () => {
    const csv = metricSeriesCsv("expenses.amount_minor", "minor", [
      {
        dimensions: { category: "food", currency: "JPY" },
        points: [{ period: "2026-01", value_minor: 800, observed_days: 1 }],
      },
    ]);
    expect(csv).toContain(
      'expenses.amount_minor,minor,"category=food,currency=JPY",2026-01,800,1',
    );
  });
});
