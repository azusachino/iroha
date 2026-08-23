import { describe, expect, it } from "vitest";
import type { MetricDefinition, MetricSeriesResponse } from "$lib/api";
import {
  metricDimensionsFromUrl,
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

function series(value: number | null): MetricSeriesResponse {
  return {
    series: [
      {
        points: [{ period: "2026-08", value_minor: value, observed_days: 1 }],
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
    expect(
      missingRequiredMetricDimensions(canonicalDefinition, {}),
    ).toEqual([]);
  });

  it("preserves an explicit valid dimension without inventing a default", () => {
    expect(
      metricDimensionsFromUrl(expenseDefinition, ["currency:EUR"]),
    ).toEqual({ currency: "EUR" });
    expect(metricDimensionsFromUrl(expenseDefinition, [])).toEqual({});
    expect(missingRequiredMetricDimensions(expenseDefinition, {})).toEqual([
      expenseDefinition.dimensions[0],
    ]);
  });

  it("rejects unknown URL dimension values", () => {
    expect(
      metricDimensionsFromUrl(expenseDefinition, ["currency:UNKNOWN"]),
    ).toEqual({});
  });

  it("distinguishes all-null series from observed zero", () => {
    expect(metricSeriesHasValues(series(null))).toBe(false);
    expect(metricSeriesHasValues(series(0))).toBe(true);
  });
});
