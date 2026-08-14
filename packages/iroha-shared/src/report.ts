import type { ExpenseCategory, ExpenseCurrency } from "./expense";

export type ReportEvidenceRow = {
  label: string;
  value: string;
  detail?: string;
};

export interface MovementReportData {
  activity_count: number;
  distance_m: number;
  distance_activity_count: number;
  duration_s: number;
  by_sport: {
    sport: string;
    activity_count: number;
    distance_m: number;
    distance_activity_count: number;
    duration_s: number;
  }[];
}

export interface SleepReportData {
  session_count: number;
  main_sleep_count: number;
  nap_count: number;
  average_asleep_s: number;
  average_time_in_bed_s: number;
  average_efficiency: number;
  stage_seconds: {
    core: number;
    deep: number;
    rem: number;
    awake: number;
    unspecified: number;
  };
}

export interface DailyHealthReportData {
  observed_days: number;
  metric_averages: {
    metric: string;
    value: number;
    unit: string;
    observed_days: number;
  }[];
}

export interface MediaReportData {
  event_count: number;
  completed_count: number;
  rated_count: number;
  average_rating: number | null;
  by_kind: {
    kind: string;
    event_count: number;
    completed_count: number;
  }[];
  completed_items: {
    id: string;
    title: string;
    media_type: string;
    completed_at: string;
  }[];
}

export interface ExpensesReportData {
  expense_count: number;
  totals_by_currency: {
    currency: ExpenseCurrency;
    currency_exponent: number;
    amount_minor: number;
    expense_count: number;
  }[];
  by_category: {
    category: ExpenseCategory;
    currency: ExpenseCurrency;
    currency_exponent: number;
    amount_minor: number;
    expense_count: number;
  }[];
}

export interface ReportSection<T> {
  schema: string;
  state: "available" | "empty";
  data: T | null;
}

export interface MonthlyReport {
  schema: string;
  period: {
    kind: "month";
    month: string;
    from: string;
    to: string;
    timezone: string;
  };
  generated_at: string;
  sections: {
    movement: ReportSection<MovementReportData>;
    sleep: ReportSection<SleepReportData>;
    daily_health: ReportSection<DailyHealthReportData>;
    media: ReportSection<MediaReportData>;
    expenses: ReportSection<ExpensesReportData>;
  };
}

export interface MonthlyReportSeriesPoint {
  month: string;
  completeness: "complete" | "partial";
  movement: { distance_m: number } | null;
  sleep: { average_asleep_s: number } | null;
  daily_health: { observed_days: number } | null;
  media: { event_count: number; completed_count: number } | null;
  expenses: {
    totals_by_currency: {
      currency: ExpenseCurrency;
      currency_exponent: number;
      amount_minor: number;
      expense_count: number;
    }[];
  } | null;
}

export interface MonthlyReportSeries {
  schema: "monthly-report-series.v2";
  end_month: string;
  requested_months: number;
  from_month: string;
  to_month: string;
  generated_at: string;
  current_report: MonthlyReport;
  reports: MonthlyReportSeriesPoint[];
  empty_months: string[];
}

export type ReportSectionKey = keyof MonthlyReport["sections"];

export function reportSectionData<T>(
  report: MonthlyReport,
  section: ReportSectionKey,
): T | null {
  const value = report.sections[section];
  return value.state === "available" ? (value.data as T | null) : null;
}

export function reportPeriodDays(report: MonthlyReport): number {
  return Math.round(
    (Date.parse(`${report.period.to}T00:00:00Z`) -
      Date.parse(`${report.period.from}T00:00:00Z`)) /
      86_400_000,
  );
}

export type ReportThemeProps = {
  month: string;
  report: MonthlyReport;
  primaryCurrency: string;
  primaryExponent: number;
  formatMoney: (
    amountMinor: number,
    currency: string,
    exponent: number,
  ) => string;
  formatDuration: (seconds: number) => string;
};
