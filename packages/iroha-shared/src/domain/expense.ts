export type ExpenseCurrency = "JPY" | "USD" | "EUR" | "GBP";

export type ExpenseCategory =
  | "food"
  | "groceries"
  | "transport"
  | "shopping"
  | "housing"
  | "utilities"
  | "health"
  | "entertainment"
  | "subscriptions"
  | "work"
  | "other";

export interface ExpenseItem {
  name: string;
  amount_minor?: number;
}

export interface ExpenseSource {
  kind: string;
  ref: string;
}

export interface Expense {
  id: string;
  occurred_on: string;
  currency: ExpenseCurrency;
  currency_exponent: number;
  amount_minor: number;
  category: ExpenseCategory;
  merchant: string;
  note: string;
  items: ExpenseItem[];
  source: ExpenseSource;
  created_at: string;
  updated_at: string;
}

export interface ExpenseInput {
  occurred_on: string;
  currency: ExpenseCurrency;
  amount_minor: number;
  category: ExpenseCategory;
  merchant?: string;
  note?: string;
  items?: ExpenseItem[];
}

export interface CreateExpenseInput extends ExpenseInput {
  source: ExpenseSource;
}

export interface ListExpensesParams {
  date?: string;
  from?: string;
  to?: string;
  currency?: ExpenseCurrency;
  category?: ExpenseCategory;
  limit?: number;
  cursor?: string;
}
