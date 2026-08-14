import { csvCell, pointValue, type SharedMetricSeries } from "./metric-series";

// One exact row behind a displayed chart. `value` stays raw for analysis while
// `display` preserves what the chart actually rendered.
export interface PanelRow {
  label: string;
  breakdown?: string;
  value: number | null;
  display: string;
  observed?: number;
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
