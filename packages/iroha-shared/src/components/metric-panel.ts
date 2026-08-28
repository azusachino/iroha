import type { Component } from "svelte";
import { csvCell, pointValue, type SharedMetricSeries } from "./metric-series";
import type { HealthMetricRange } from "../domain/health-metric-ranges";

// One exact row behind a displayed chart. `value` stays raw for analysis while
// `display` preserves what the chart actually rendered.
export interface PanelRow {
  label: string;
  // Icon-set-agnostic on purpose -- callers (e.g. healthMetricIcon in
  // domain/health-metric-icons.ts) own which icon set a row's icon comes
  // from; this file stays generic.
  icon?: Component<any>;
  // A CSS custom property name (e.g. "--accent"), not a resolved color --
  // lets a row's accent follow the active theme instead of being baked in.
  colorVar?: string;
  breakdown?: string;
  value: number | null;
  display: string;
  observed?: number;
  // Only present for rows with a real, citable reference range (see
  // domain/health-metric-ranges.ts) -- absence means "no range exists",
  // not "not loaded yet".
  range?: HealthMetricRange;
}

export interface PanelCoverage {
  expected_periods: number;
  observed_periods: number;
}

export function panelCsv(
  metricId: string,
  unit: string,
  rowHeader: string,
  rows: PanelRow[],
): string {
  const out: (string | number | null)[][] = [
    [
      "metric_id",
      "unit",
      rowHeader.toLowerCase(),
      "breakdown",
      "value",
      "display",
      "observed_days",
    ],
  ];
  for (const row of rows) {
    out.push([
      metricId,
      unit,
      row.label,
      row.breakdown ?? "",
      row.value,
      row.display,
      row.observed ?? "",
    ]);
  }
  return out.map((row) => row.map(csvCell).join(",")).join("\n") + "\n";
}

export function seriesPanelRows(
  series: SharedMetricSeries[],
  format: (value: number | null) => string = (value) =>
    value == null ? "—" : String(value),
): PanelRow[] {
  const rows: PanelRow[] = [];
  for (const item of series) {
    const breakdown = Object.entries(item.dimensions)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, value]) => `${key}: ${value}`)
      .join(", ");
    for (const point of item.points) {
      const value = pointValue(point);
      rows.push({
        label: point.period,
        breakdown: breakdown || undefined,
        value,
        display: format(value),
        observed: point.observed_days,
      });
    }
  }
  return rows;
}
