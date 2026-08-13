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

