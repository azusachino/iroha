export interface SharedMetricPoint {
  period: string;
  observed_days: number;
  value?: number | null;
  value_minor?: number | null;
}

export interface SharedMetricSeries {
  dimensions: Record<string, string>;
  points: SharedMetricPoint[];
}

export function pointValue(point: SharedMetricPoint): number | null {
  if (point.value_minor !== undefined) return point.value_minor;
  return point.value ?? null;
}

export function pointHasValue(point: SharedMetricPoint): boolean {
  return pointValue(point) !== null;
}

export function csvCell(value: string | number | null | undefined): string {
  const text = value == null ? "" : String(value);
  return /[",\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}

export function metricSeriesCsv(
  metricId: string,
  unit: string,
  series: SharedMetricSeries[],
): string {
  const rows: (string | number | null)[][] = [
    ["metric_id", "unit", "dimensions", "period", "value", "observed_days"],
  ];
  for (const item of series) {
    const dimensions = Object.entries(item.dimensions)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, value]) => `${key}=${value}`)
      .join(",");
    for (const point of item.points) {
      rows.push([
        metricId,
        unit,
        dimensions,
        point.period,
        pointValue(point),
        point.observed_days,
      ]);
    }
  }
  return rows.map((row) => row.map(csvCell).join(",")).join("\n") + "\n";
}
