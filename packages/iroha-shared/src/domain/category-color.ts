const EXPENSE_CATEGORY_COLOR_VARS: Record<string, string> = {
  food: "--mark-amber",
  groceries: "--ring-exercise",
  transport: "--sport-swim",
  shopping: "--accent-2",
  housing: "--sport-ride",
  utilities: "--ring-stand",
  health: "--ring-move",
  entertainment: "--sport-run",
  subscriptions: "--accent",
  work: "--sport-walk",
  other: "--text-muted",
};

export function categoryColor(category: string): string | undefined {
  const colorVar = EXPENSE_CATEGORY_COLOR_VARS[category];
  return colorVar ? `var(${colorVar})` : undefined;
}

// Bare custom-property name (no var() wrapper), for consumers like
// PanelRow.colorVar that resolve it themselves.
export function categoryColorVar(category: string): string | undefined {
  return EXPENSE_CATEGORY_COLOR_VARS[category];
}
