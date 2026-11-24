import { allNavigationItems, type NavigationItem } from "./navigation";

export type PaletteCommand =
  | {
      id: string;
      label: string;
      hint: string;
      group: "Navigation";
      keywords: readonly string[];
      action: { type: "navigate"; href: string };
    }
  | {
      id: string;
      label: string;
      hint: string;
      group: "Action";
      keywords: readonly string[];
      action: { type: "cycle-theme" } | { type: "navigate"; href: string };
    };

export function buildPaletteCommands(
  items: readonly NavigationItem[] = allNavigationItems(),
): PaletteCommand[] {
  const navCommands: PaletteCommand[] = items
    .filter((item) => item.id !== "public-archive")
    .map((item) => ({
      id: item.id,
      label: item.label,
      hint: item.hint,
      group: "Navigation",
      keywords: [item.id, item.href, item.kind],
      action: { type: "navigate", href: item.href },
    }));

  const quickActions: PaletteCommand[] = [
    {
      id: "action-cycle-theme",
      label: "Cycle theme",
      hint: "Switch between auto, light, and dark",
      group: "Action",
      keywords: [
        "theme",
        "dark",
        "light",
        "auto",
        "appearance",
        "color scheme",
      ],
      action: { type: "cycle-theme" },
    },
    {
      id: "public-archive",
      label: "Open public showcase",
      hint: "View Miles & Margins public archive",
      group: "Action",
      keywords: [
        "public",
        "archive",
        "showcase",
        "miles",
        "margins",
        "/showcase",
      ],
      action: { type: "navigate", href: "/showcase" },
    },
  ];

  return [...navCommands, ...quickActions];
}

export function filterPaletteCommands(
  commands: readonly PaletteCommand[],
  query: string,
): PaletteCommand[] {
  const tokens = query
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .filter((token) => token.length > 0);
  if (tokens.length === 0) return [...commands];

  return commands.filter((command) => {
    const haystack = [
      command.label,
      command.hint,
      command.group,
      ...command.keywords,
    ]
      .join(" ")
      .toLowerCase();
    return tokens.every((token) => haystack.includes(token));
  });
}

export function nextSelectedIndex(
  current: number,
  delta: 1 | -1,
  count: number,
): number {
  if (count <= 0) return -1;
  if (current < 0 || current >= count) return delta > 0 ? 0 : count - 1;
  return (current + delta + count) % count;
}
