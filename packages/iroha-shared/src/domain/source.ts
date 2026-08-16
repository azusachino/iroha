export type SourceBrandId =
  | "apple-health"
  | "apple-watch"
  | "garmin"
  | "strava"
  | "fitbit"
  | "oura"
  | "unknown";

export type SourceBrand = {
  id: SourceBrandId;
  label: string;
  mark: string;
  raw: string;
};

function fallbackLabel(source: string): string {
  return source
    .replaceAll("_", " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function sourceBrand(source?: string | null): SourceBrand {
  const raw = source?.trim() ?? "";
  const normalized = raw.toLowerCase().replace(/[\s-]+/g, "_");

  if (normalized.includes("apple_health") || normalized.includes("healthkit")) {
    return { id: "apple-health", label: "Apple Health", mark: "", raw };
  }
  if (normalized.includes("apple") || normalized.includes("watch")) {
    return { id: "apple-watch", label: "Apple Watch", mark: "", raw };
  }
  if (normalized.includes("garmin")) {
    return { id: "garmin", label: "Garmin", mark: "G", raw };
  }
  if (normalized.includes("strava")) {
    return { id: "strava", label: "Strava", mark: "S", raw };
  }
  if (normalized.includes("fitbit")) {
    return { id: "fitbit", label: "Fitbit", mark: "F", raw };
  }
  if (normalized.includes("oura")) {
    return { id: "oura", label: "Oura", mark: "O", raw };
  }

  return {
    id: "unknown",
    label: raw ? fallbackLabel(raw) : "Unknown source",
    mark: raw ? Array.from(raw)[0]?.toUpperCase() || "?" : "?",
    raw,
  };
}
