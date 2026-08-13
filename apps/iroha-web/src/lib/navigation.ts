export type NavigationItem = {
  id: string;
  label: string;
  href: string;
  kind: "primary" | "domain" | "analysis" | "tool";
  hint: string;
};

export type NavigationGroup = {
  id: string;
  label: string;
  items: readonly NavigationItem[];
};

export const navigationGroups: readonly NavigationGroup[] = [
  {
    id: "primary",
    label: "Primary",
    items: [
      {
        id: "today",
        label: "Today",
        href: "/",
        kind: "primary",
        hint: "Any-day cross-domain view",
      },
      {
        id: "overview",
        label: "Overview",
        href: "/overview",
        kind: "primary",
        hint: "Cross-domain overview",
      },
    ],
  },
  {
    id: "domains",
    label: "Domains",
    items: [
      {
        id: "motion",
        label: "Motion",
        href: "/motion",
        kind: "domain",
        hint: "Movement sessions and routes",
      },
      {
        id: "night",
        label: "Night",
        href: "/night",
        kind: "domain",
        hint: "Recovery and sleep sessions",
      },
      {
        id: "library",
        label: "Library",
        href: "/library",
        kind: "domain",
        hint: "Watch, reading, and game history",
      },
      {
        id: "expenses",
        label: "Expenses",
        href: "/expenses",
        kind: "domain",
        hint: "Canonical spending ledger",
      },
    ],
  },
  {
    id: "analysis",
    label: "Analyze",
    items: [
      {
        id: "patterns",
        label: "Patterns",
        href: "/patterns",
        kind: "analysis",
        hint: "Recurring cross-domain patterns",
      },
      {
        id: "reports",
        label: "Reports",
        href: "/reports",
        kind: "analysis",
        hint: "Monthly cross-domain report",
      },
    ],
  },
  {
    id: "tools",
    label: "More",
    items: [
      {
        id: "to-go",
        label: "To-go",
        href: "/to-go",
        kind: "tool",
        hint: "Tasks and background jobs",
      },
      {
        id: "admin",
        label: "Admin",
        href: "/admin",
        kind: "tool",
        hint: "System administration",
      },
      {
        id: "manual",
        label: "Manual",
        href: "/manual",
        kind: "tool",
        hint: "How to read Iroha",
      },
      {
        id: "design",
        label: "Design",
        href: "/design",
        kind: "tool",
        hint: "Design language",
      },
    ],
  },
] as const;

export const primaryNavigation = navigationGroups.flatMap(
  (group) => group.items,
);

export function allNavigationItems(): NavigationItem[] {
  return navigationGroups.flatMap((group) => [...group.items]);
}
