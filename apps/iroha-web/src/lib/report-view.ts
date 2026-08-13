import type { MonthlyReport } from "$lib/api";

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
