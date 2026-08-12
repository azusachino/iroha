import type { MetricSeriesResponse, MonthlyReport } from "$lib/api";

export type ReportTrendPoint = {
  month: string;
  label: string;
  amount: number | null;
  count: number | null;
};

export type ReportThemeProps = {
  month: string;
  timezone: string;
  report: MonthlyReport;
  trend: ReportTrendPoint[];
  trendSeries: MetricSeriesResponse | null;
  primaryCurrency: string;
  primaryExponent: number;
  categoryTotals: MonthlyReport["sections"]["expenses"]["data"] extends infer T
    ? T extends { by_category: infer C }
      ? C
      : never
    : never;
  currentTotal: number;
  previousTotal: number;
  expenseRecordCount: number;
  topCategory: string;
  currencyCount: number;
  comparisonLabel: string;
  formatMoney: (
    amountMinor: number,
    currency: string,
    exponent: number,
  ) => string;
  formatDuration: (seconds: number) => string;
};
