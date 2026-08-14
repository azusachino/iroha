export const DESIGN_COMPOSITIONS = [
  {
    id: "editorial",
    index: "A",
    label: "Editorial",
    intent: "Read the day as a composed story.",
  },
  {
    id: "command",
    index: "B",
    label: "Command center",
    intent: "Keep the important signals visible and actionable.",
  },
  {
    id: "chronicle",
    index: "C",
    label: "Chronicle",
    intent: "Follow the day as a lived sequence of events.",
  },
  {
    id: "cover",
    index: "D",
    label: "Cover page",
    intent: "Give one day a strong, legible visual identity.",
  },
  {
    id: "workspace",
    index: "E",
    label: "Personal OS",
    intent: "Arrange the day as a calm working surface.",
  },
  {
    id: "journal",
    index: "F",
    label: "Field journal",
    intent: "Preserve observations with continuity and provenance.",
  },
  {
    id: "quiet",
    index: "G",
    label: "Quiet",
    intent: "Notice a few meaningful signals without dashboard noise.",
  },
] as const;

export type DesignCompositionId = (typeof DESIGN_COMPOSITIONS)[number]["id"];

export function isDesignComposition(
  value: string | null | undefined,
): value is DesignCompositionId {
  return DESIGN_COMPOSITIONS.some((composition) => composition.id === value);
}

export function designComposition(
  value: string | null | undefined,
): DesignCompositionId {
  return isDesignComposition(value) ? value : "editorial";
}
