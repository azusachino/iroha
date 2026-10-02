const EXPENSE_CATEGORY_COLOR_VARS: Record<string, string> = {
  food: "--mark-amber",
  groceries: "--category-groceries",
  transport: "--sport-swim",
  shopping: "--category-shopping",
  housing: "--sport-ride",
  utilities: "--category-utilities",
  health: "--category-health",
  entertainment: "--sport-run",
  subscriptions: "--category-subscriptions",
  work: "--sport-walk",
  other: "--text-muted",
};

export function categoryColor(category: string): string | undefined {
  return `var(${categoryColorVar(category)})`;
}

// Bare custom-property name (no var() wrapper), for consumers like
// PanelRow.colorVar that resolve it themselves.
export function categoryColorVar(category: string): string | undefined {
  return EXPENSE_CATEGORY_COLOR_VARS[category] ?? "--text-muted";
}
