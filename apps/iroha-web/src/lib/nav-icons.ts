import type { LucideIcon } from "@lucide/svelte";
import {
  Activity,
  BookMarked,
  BookOpen,
  Ellipsis,
  FileText,
  HeartPulse,
  Layers,
  LayoutDashboard,
  ListTodo,
  Moon,
  Receipt,
  Shield,
  Sparkles,
  TrendingUp,
} from "@lucide/svelte";

export const navGroupIcons: Record<string, LucideIcon> = {
  primary: LayoutDashboard,
  domains: Layers,
  analysis: TrendingUp,
  tools: Ellipsis,
};

export const navItemIcons: Record<string, LucideIcon> = {
  today: HeartPulse,
  overview: LayoutDashboard,
  motion: Activity,
  night: Moon,
  library: BookOpen,
  expenses: Receipt,
  patterns: Sparkles,
  reports: FileText,
  "to-go": ListTodo,
  admin: Shield,
  manual: BookMarked,
};
