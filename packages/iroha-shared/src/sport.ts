import { formatSport } from "./format";

const SPORT_COLOR_RUN = "var(--sport-run)";
const SPORT_COLOR_WALK = "var(--sport-walk)";
const SPORT_COLOR_HIKE = "var(--sport-hike)";
const SPORT_COLOR_RIDE = "var(--sport-ride)";
const SPORT_COLOR_SWIM = "var(--sport-swim)";
const SPORT_COLOR_OTHER = "var(--sport-other)";

export function sportColor(sport?: string | null): string {
  const normalized = sport?.toLowerCase() ?? "";
  if (normalized.includes("run")) return SPORT_COLOR_RUN;
  if (normalized.includes("walk")) return SPORT_COLOR_WALK;
  if (normalized.includes("hik")) return SPORT_COLOR_HIKE;
  if (normalized.includes("swim")) return SPORT_COLOR_SWIM;
  if (
    normalized.includes("ride") ||
    normalized.includes("cycl") ||
    normalized.includes("bik")
  ) {
    return SPORT_COLOR_RIDE;
  }
  return SPORT_COLOR_OTHER;
}

export function sportLabel(sport?: string | null): string {
  return formatSport(sport ?? undefined);
}

export function isSwimming(sport?: string | null): boolean {
  return sport?.toLowerCase().includes("swim") ?? false;
}
