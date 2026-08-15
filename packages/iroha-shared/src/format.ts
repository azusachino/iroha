import { DEFAULT_TIMEZONE } from "./date";

const DASH = "—";

export function formatCanonicalMonth(period?: string): string {
  if (!period) return DASH;
  const match = /^(\d{4})-(\d{1,2})$/.exec(period);
  if (!match) return period;
  const month = Number(match[2]);
  return month >= 1 && month <= 12
    ? `${match[1]}-${String(month).padStart(2, "0")}`
    : period;
}

export function formatDistance(meters?: number): string {
  if (meters == null) return DASH;
  if (meters < 1000) return `${Math.round(meters)} m`;
  return `${(meters / 1000).toFixed(2)} km`;
}

export function formatDuration(seconds?: number): string {
  if (seconds == null) return DASH;
  const rounded = Math.round(seconds);
  const hours = Math.floor(rounded / 3600);
  const minutes = Math.floor((rounded % 3600) / 60);
  const remainder = rounded % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  if (hours > 0) return `${hours}:${pad(minutes)}:${pad(remainder)}`;
  return `${minutes}:${pad(remainder)}`;
}

export function formatPace(secondsPerKm?: number): string {
  if (
    secondsPerKm == null ||
    !Number.isFinite(secondsPerKm) ||
    secondsPerKm <= 0
  ) {
    return DASH;
  }
  const minutes = Math.floor(secondsPerKm / 60);
  const seconds = Math.round(secondsPerKm % 60);
  return `${minutes}:${String(seconds).padStart(2, "0")} /km`;
}

export function formatSwimmingPace(
  distanceM?: number | null,
  durationS?: number | null,
): string {
  if (
    distanceM == null ||
    durationS == null ||
    !Number.isFinite(distanceM) ||
    !Number.isFinite(durationS) ||
    distanceM <= 0 ||
    durationS <= 0
  ) {
    return DASH;
  }
  const rounded = Math.round(durationS / (distanceM / 100));
  const minutes = Math.floor(rounded / 60);
  const seconds = rounded % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")} /100m`;
}

export function formatElevation(meters?: number): string {
  if (meters == null) return DASH;
  return `${Math.round(meters)} m`;
}

export function formatHr(bpm?: number): string {
  if (bpm == null) return DASH;
  return `${Math.round(bpm)} bpm`;
}

export function formatMetricValue(
  value?: number | null,
  unit?: string | null,
): string {
  if (value == null || !Number.isFinite(value)) return DASH;
  const maximumFractionDigits = unit?.trim().toLowerCase() === "count" ? 0 : 1;
  return value.toLocaleString(undefined, { maximumFractionDigits });
}

export function formatDate(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("sv-SE", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
      timeZone: timezone || DEFAULT_TIMEZONE,
    }).format(date);
  } catch {
    return date.toISOString().slice(0, 19).replace("T", " ");
  }
}

export function formatDateOnly(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) return iso;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("sv-SE", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      timeZone: timezone || DEFAULT_TIMEZONE,
    }).format(date);
  } catch {
    return date.toISOString().slice(0, 10);
  }
}

export function formatDateShort(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
    const [, year, month, day] = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso) ?? [];
    if (year && month && day) {
      return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        timeZone: "UTC",
      }).format(new Date(`${year}-${month}-${day}T00:00:00Z`));
    }
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("en-US", {
      month: "short",
      day: "numeric",
      timeZone: timezone || DEFAULT_TIMEZONE,
    }).format(date);
  } catch {
    return date.toISOString().slice(5, 10);
  }
}

export function formatSport(sport?: string): string {
  if (!sport) return DASH;
  return sport
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
