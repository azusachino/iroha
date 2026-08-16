const EXPENSE_CATEGORY_COLORS: Record<string, string> = {
  food: "var(--mark-amber)",
  groceries: "var(--ring-exercise)",
  transport: "var(--sport-swim)",
  shopping: "var(--accent-2)",
  housing: "var(--sport-ride)",
  utilities: "var(--ring-stand)",
  health: "var(--ring-move)",
  entertainment: "var(--sport-run)",
  subscriptions: "var(--accent)",
  work: "var(--sport-walk)",
  other: "var(--text-muted)",
};

export function categoryColor(category: string): string | undefined {
  return EXPENSE_CATEGORY_COLORS[category];
}
