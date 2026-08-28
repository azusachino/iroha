import { formatSport } from "../format/format";

// The canonical key behind --sport-<key> in apps/iroha-web/src/routes/app.css.
// sportColor/sportColorVar both resolve through this so a new caller (e.g.
// domain/sport-icons.ts) matches sport_type strings the same way the
// existing badge coloring already does, instead of a second guess at it.
export function canonicalSport(sport?: string | null): string {
  const normalized = sport?.toLowerCase() ?? "";
  if (normalized.includes("run")) return "run";
  if (normalized.includes("walk")) return "walk";
  if (normalized.includes("hik")) return "hike";
  if (normalized.includes("swim")) return "swim";
  if (
    normalized.includes("ride") ||
    normalized.includes("cycl") ||
    normalized.includes("bik")
  ) {
    return "ride";
  }
  return "other";
}

export function sportColor(sport?: string | null): string {
  return `var(--sport-${canonicalSport(sport)})`;
}

// Bare custom-property name (no var() wrapper), for consumers like
// PanelRow.colorVar that resolve it themselves.
export function sportColorVar(sport?: string | null): string {
  return `--sport-${canonicalSport(sport)}`;
}

export function sportLabel(sport?: string | null): string {
  return formatSport(sport ?? undefined);
}

export function isSwimming(sport?: string | null): boolean {
  return sport?.toLowerCase().includes("swim") ?? false;
}
