import type { PanelCoverage, PanelRow } from "@iroha/shared/metric-panel";
import type { Expense, ExpenseCategory, ExpenseCurrency } from "$lib/api";

// Everything a theme needs to hand a chart to the shared MetricPanel. The
// route owns it so provenance and exact rows cannot diverge per theme.
export type ExpensePanel = {
  metricId: string;
  unit: string;
  method: string;
  coverage?: PanelCoverage;
  sourceKinds: string[];
  rowHeader: string;
  rows: PanelRow[];
};

export type ExpenseCurrencyTotal = {
  currency: ExpenseCurrency;
  amountMinor: number;
  exponent: number;
  count: number;
};

export type ExpenseCategoryTotal = {
  category: string;
  amount: number;
};

export type ExpenseDailyTotal = [period: string, amountMinor: number | null];

export type ExpenseThemeProps = {
  month: string;
  primaryCurrency: ExpenseCurrency;
  primaryExponent: number;
  currencyTotals: ExpenseCurrencyTotal[];
  categoryTotals: ExpenseCategoryTotal[];
  dailyTotals: ExpenseDailyTotal[];
  dailyPanel: ExpensePanel;
  categoryPanel: ExpensePanel;
  expenses: Expense[];
  selected: Expense | null;
  selectedId: string;
  detailLoading: boolean;
  onSelect: (id: string) => void;
  onRemove: (expense: Expense) => void;
  formatMoney: (
    amountMinor: number,
    currency: string,
    exponent: number,
  ) => string;
};

export const expenseCategoryLabel: Record<ExpenseCategory, string> = {
  food: "Food",
  groceries: "Groceries",
  transport: "Transport",
  shopping: "Shopping",
  housing: "Housing",
  utilities: "Utilities",
  health: "Health",
  entertainment: "Entertainment",
  subscriptions: "Subscriptions",
  work: "Work",
  other: "Other",
};

export function formatExpenseDay(value: string): string {
  return new Date(`${value}T00:00:00Z`).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
}
