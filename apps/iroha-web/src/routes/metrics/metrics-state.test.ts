import { describe, expect, it } from "vitest";
import type { MetricDefinition, MetricSeriesResponse } from "$lib/api";
import {
  metricDimensionsFromUrl,
  metricSearchParams,
  metricSelectionIsComplete,
  metricSeriesDimensions,
  metricSeriesHasValues,
  missingRequiredMetricDimensions,
} from "./metrics-state";

const expenseDefinition = {
  id: "expenses.amount_minor",
  dimensions: [
    {
      id: "currency",
      label: "Currency",
      values: ["EUR", "JPY"],
      required: true,
      expand_by_default: false,
    },
  ],
} as MetricDefinition;

function series(...values: Array<number | null>): MetricSeriesResponse {
  return {
    series: [
      {
        points: values.map((value, index) => ({
          period: `2026-${String(index + 1).padStart(2, "0")}`,
          value_minor: value,
          observed_days: value == null ? 0 : 1,
        })),
      },
    ],
  } as MetricSeriesResponse;
}

describe("Metrics selection state", () => {
  it("treats the API's null dimensions for canonical metrics as empty", () => {
    const canonicalDefinition = {
      id: "health.steps",
      dimensions: null,
    } as unknown as MetricDefinition;

    expect(metricDimensionsFromUrl(canonicalDefinition, [])).toEqual({});
    expect(missingRequiredMetricDimensions(canonicalDefinition, {})).toEqual(
      [],
    );
  });

  it("preserves an explicit valid dimension without inventing a default", () => {
    expect(
      metricDimensionsFromUrl(expenseDefinition, ["currency:EUR"]),
    ).toEqual({ currency: "EUR" });
    expect(metricDimensionsFromUrl(expenseDefinition, [])).toEqual({});
    expect(missingRequiredMetricDimensions(expenseDefinition, {})).toEqual([
      expenseDefinition.dimensions[0],
    ]);
    expect(metricSelectionIsComplete(expenseDefinition, {})).toBe(false);
    expect(
      metricSelectionIsComplete(expenseDefinition, { currency: "EUR" }),
    ).toBe(true);
  });

  it("rejects unknown URL dimension values", () => {
    expect(
      metricDimensionsFromUrl(expenseDefinition, ["currency:UNKNOWN"]),
    ).toEqual({});
  });

  it("defers requests and URL writes until required dimensions are chosen", () => {
    expect(metricSeriesDimensions(expenseDefinition, {})).toBeNull();
    expect(
      metricSearchParams(
        "?source=manual",
        expenseDefinition.id,
        "2026-08",
        expenseDefinition,
        {},
      ),
    ).toBeNull();
  });

  it("writes the explicit selection to request and URL state", () => {
    const dimensions = { currency: "EUR" };

    expect(metricSeriesDimensions(expenseDefinition, dimensions)).toEqual([
      "currency:EUR",
    ]);
    expect(
      metricSearchParams(
        "?source=manual",
        expenseDefinition.id,
        "2026-08",
        expenseDefinition,
        dimensions,
      )?.toString(),
    ).toBe(
      "source=manual&metric=expenses.amount_minor&date=2026-08&dimension=currency%3AEUR",
    );
  });

  it("distinguishes all-null series from observed zero", () => {
    expect(metricSeriesHasValues(series(null))).toBe(false);
    expect(metricSeriesHasValues(series(0))).toBe(true);
  });

  it("treats a partial series as observed without hiding its gaps", () => {
    expect(metricSeriesHasValues(series(null, 1200, null))).toBe(true);
  });
});
