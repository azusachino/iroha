import type { MetricDefinition, MetricSeriesResponse } from "$lib/api";
import { pointHasValue } from "@iroha/shared/components/metric-series";

export function metricDimensionsFromUrl(
  definition: MetricDefinition,
  encodedDimensions: string[],
): Record<string, string> {
  const requested = new Map(
    encodedDimensions.map((value) => {
      const separator = value.indexOf(":");
      return separator > 0
        ? [value.slice(0, separator), value.slice(separator + 1)]
        : ["", ""];
    }),
  );

  return Object.fromEntries(
    definition.dimensions.flatMap((dimension) => {
      const value = requested.get(dimension.id) ?? "";
      return dimension.values.includes(value) ? [[dimension.id, value]] : [];
    }),
  );
}

export function missingRequiredMetricDimensions(
  definition: MetricDefinition | null,
  dimensions: Record<string, string>,
) {
  return (definition?.dimensions ?? []).filter(
    (dimension) => dimension.required && !dimensions[dimension.id],
  );
}

export function metricSeriesHasValues(
  response: MetricSeriesResponse | null,
): boolean {
  return Boolean(
    response?.series.some((series) => series.points.some(pointHasValue)),
  );
}
