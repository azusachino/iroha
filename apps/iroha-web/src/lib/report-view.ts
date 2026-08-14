import type { MonthlyReport } from "$lib/api";

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
