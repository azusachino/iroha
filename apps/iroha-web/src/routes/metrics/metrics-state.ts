import type { MetricDefinition, MetricSeriesResponse } from "$lib/api";
import { pointHasValue } from "@iroha/shared/components/metric-series";
import { writeCalendarScope } from "@iroha/shared/format/scope";

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
    (definition.dimensions ?? []).flatMap((dimension) => {
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

export function metricSelectionIsComplete(
  definition: MetricDefinition | null,
  dimensions: Record<string, string>,
): boolean {
  return missingRequiredMetricDimensions(definition, dimensions).length === 0;
}

export function metricSeriesDimensions(
  definition: MetricDefinition | null,
  dimensions: Record<string, string>,
): string[] | null {
  if (!definition || !metricSelectionIsComplete(definition, dimensions)) {
    return null;
  }
  return Object.entries(dimensions)
    .filter(([, value]) => value)
    .map(([key, value]) => `${key}:${value}`);
}

export function metricSearchParams(
  currentSearch: string,
  metricId: string,
  month: string,
  definition: MetricDefinition | null,
  dimensions: Record<string, string>,
): URLSearchParams | null {
  if (!metricSelectionIsComplete(definition, dimensions)) return null;
  const params = new URLSearchParams(currentSearch);
  params.set("metric", metricId);
  writeCalendarScope(params, {
    kind: "month",
    year: Number(month.slice(0, 4)),
    month: Number(month.slice(5, 7)),
  });
  params.delete("dimension");
  for (const [id, value] of Object.entries(dimensions)) {
    if (value) params.append("dimension", `${id}:${value}`);
  }
  return params;
}

export function metricSeriesHasValues(
  response: MetricSeriesResponse | null,
): boolean {
  return Boolean(
    response?.series.some((series) => series.points.some(pointHasValue)),
  );
}
