import type { Component } from "svelte";
import {
  Briefcase,
  Ellipsis,
  Film,
  HeartPulse,
  House,
  Plug,
  Repeat,
  ShoppingBag,
  ShoppingCart,
  Car,
  Utensils,
} from "@lucide/svelte";

const EXPENSE_CATEGORY_ICONS: Record<string, Component<any>> = {
  food: Utensils,
  groceries: ShoppingCart,
  transport: Car,
  shopping: ShoppingBag,
  housing: House,
  utilities: Plug,
  health: HeartPulse,
  entertainment: Film,
  subscriptions: Repeat,
  work: Briefcase,
  other: Ellipsis,
};

export function expenseCategoryIcon(category: string): Component<any> | undefined {
  return EXPENSE_CATEGORY_ICONS[category];
}
