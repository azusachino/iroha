import type { LucideIcon } from "@lucide/svelte";
import { csvCell, pointValue, type SharedMetricSeries } from "./metric-series";
import type { HealthMetricRange } from "../domain/health-metric-ranges";

// One exact row behind a displayed chart. `value` stays raw for analysis while
// `display` preserves what the chart actually rendered.
export interface PanelRow {
  label: string;
  // Every current producer (sportIcon, expenseCategoryIcon, healthMetricIcon)
  // draws from @lucide/svelte, so the field is typed to that icon set
  // directly instead of a generic `Component<any>`.
  icon?: LucideIcon;
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

// Shared by CoverageBars (visual, aria-hidden) and MetricTable (the
// accessible fallback) so a row's in/out-of-range status can never read
// differently between the two -- computed once, not duplicated per consumer.
export function isOutOfRange(row: PanelRow): boolean {
  return (
    row.range != null &&
    row.value != null &&
    (row.value < row.range.min || row.value > row.range.max)
  );
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
