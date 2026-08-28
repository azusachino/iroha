const EXPENSE_CATEGORY_LABELS: Record<string, string> = {
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

export function expenseCategoryLabel(category: string): string {
  return (
    EXPENSE_CATEGORY_LABELS[category] ??
    category.charAt(0).toUpperCase() + category.slice(1)
  );
}
