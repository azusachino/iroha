import type { PanelCoverage, PanelRow } from "./metric-panel";
import { csvCell } from "./metric-series";
import type { Expense, ExpenseCategory, ExpenseCurrency } from "./expense";

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

export function expenseMetricDimensions(
  currency: ExpenseCurrency,
  category?: ExpenseCategory,
): string[] {
  return [
    `currency:${currency}`,
    ...(category ? [`category:${category}`] : []),
  ];
}

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
  return value.slice(0, 10);
}

// Canonical ledger export: raw money remains numeric and the display amount is
// supplementary. Text cells are quoted by the shared CSV helper so merchant
// notes, plus signs, currency labels, and item JSON round-trip safely.
export function expenseLedgerCsv(
  expenses: Expense[],
  formatMoney: (
    amountMinor: number,
    currency: string,
    exponent: number,
  ) => string,
): string {
  const rows: (string | number | null)[][] = [
    [
      "id",
      "occurred_on",
      "currency",
      "currency_exponent",
      "amount_minor",
      "display_amount",
      "category",
      "merchant",
      "note",
      "items_json",
      "source_kind",
      "source_ref",
      "created_at",
      "updated_at",
    ],
    ...expenses.map((expense) => [
      expense.id,
      expense.occurred_on,
      expense.currency,
      expense.currency_exponent,
      expense.amount_minor,
      formatMoney(
        expense.amount_minor,
        expense.currency,
        expense.currency_exponent,
      ),
      expense.category,
      expense.merchant,
      expense.note,
      JSON.stringify(expense.items),
      expense.source.kind,
      expense.source.ref,
      expense.created_at,
      expense.updated_at,
    ]),
  ];
  return rows.map((row) => row.map(csvCell).join(",")).join("\n") + "\n";
}
