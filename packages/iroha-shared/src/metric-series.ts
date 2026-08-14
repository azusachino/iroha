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

export type MetricPoint =
  | {
      period: string;
      value: number | null;
      observed_days: number;
      value_minor?: never;
    }
  | {
      period: string;
      value_minor: number | null;
      observed_days: number;
      value?: never;
    };

export interface MetricSeriesResponse {
  schema: "metric-series.v1";
  metric_id: string;
  label: string;
  unit: string;
  value_type: string;
  period: {
    grain: "day" | "month" | "year";
    from: string;
    to: string;
    timezone: string;
  };
  series: {
    dimensions: Record<string, string>;
    points: MetricPoint[];
    coverage: {
      expected_periods: number;
      observed_periods: number;
    };
    source: {
      kind: string;
      method: string;
      source_kinds: string[];
    };
  }[];
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
  if (typeof value !== "string") return text;
  return `"${text.replaceAll('"', '""')}"`;
}
