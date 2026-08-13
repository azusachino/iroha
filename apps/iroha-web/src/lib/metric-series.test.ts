import { describe, expect, it } from "vitest";
import { panelCsv, seriesPanelRows } from "@iroha/shared/metric-panel";
import { pointHasValue, pointValue } from "@iroha/shared/metric-series";
import { expenseLedgerCsv } from "$lib/expense-view";

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

  it("turns dimensioned series into exact panel rows", () => {
    const rows = seriesPanelRows(
      [
        {
          dimensions: { category: "food", currency: "JPY" },
          points: [
            { period: "2026-01", value_minor: 800, observed_days: 1 },
            { period: "2026-02", value_minor: null, observed_days: 0 },
          ],
        },
      ],
      (value) => (value == null ? "—" : `¥${value}`),
    );
    expect(rows).toEqual([
      {
        label: "2026-01",
        breakdown: "category: food, currency: JPY",
        value: 800,
        display: "¥800",
        observed: 1,
      },
      {
        label: "2026-02",
        breakdown: "category: food, currency: JPY",
        value: null,
        display: "—",
        observed: 0,
      },
    ]);
  });
});

describe("panel CSV export", () => {
  it("exports raw value and displayed value as lossless cells", () => {
    const csv = panelCsv("expenses.amount_minor", "minor", "Period", [
      {
        label: "2026-01",
        breakdown: "category: food, currency: JPY",
        value: 800,
        display: "¥800",
        observed: 1,
      },
    ]);
    expect(csv.split("\n")[0]).toBe(
      '"metric_id","unit","period","breakdown","value","display","observed_days"',
    );
    expect(csv).toContain(
      '"expenses.amount_minor","minor","2026-01","category: food, currency: JPY",800,"¥800",1',
    );
  });

  it("leaves missing values empty rather than inventing zero", () => {
    const csv = panelCsv("health.steps", "count", "Period", [
      { label: "2026-01", value: null, display: "—" },
    ]);
    expect(csv).toContain('"health.steps","count","2026-01","",,"—",""\n');
  });

  it("quotes every ledger text cell and keeps raw money columns numeric", () => {
    const csv = expenseLedgerCsv(
      [
        {
          id: "expense-1",
          occurred_on: "2026-08-14",
          currency: "JPY",
          currency_exponent: 0,
          amount_minor: 800,
          category: "food",
          merchant: "Ramen, + bar",
          note: 'Dinner "late"\nwith friend',
          items: [{ name: "ramen", amount_minor: 800 }],
          source: { kind: "api", ref: "telegram:+1" },
          created_at: "2026-08-14T00:00:00Z",
          updated_at: "2026-08-14T00:00:00Z",
        },
      ],
      (amount, currency) => `${currency} + ${amount}`,
    );
    expect(csv.split("\n")[0]).toContain('"display_amount"');
    expect(csv).toContain('"Ramen, + bar"');
    expect(csv).toContain('"Dinner ""late""\nwith friend"');
    expect(csv).toContain(',800,"JPY + 800",');
  });
});
