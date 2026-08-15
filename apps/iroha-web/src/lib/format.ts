// Presentation helpers. All inputs may be undefined; missing data renders as
// an em dash so the UI degrades gracefully.

export {
  cleanDescription,
  formatProgressCount,
  mediaEventLabel,
  mediaEventVerb,
  mediaWorkTotal,
  progressPercent,
} from "@iroha/shared/media";
import { DEFAULT_TIMEZONE } from "@iroha/shared/date";
import { IROHA_TIMEZONE } from "./config";

const DASH = "—";

export function boundPercent(value?: number | null): number {
  if (value == null || !Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

export function formatPercent(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return DASH;
  return `${Math.round(boundPercent(value))}%`;
}

export { formatMetricValue } from "@iroha/shared/format";

export function formatDistance(meters?: number): string {
  if (meters == null) return DASH;
  if (meters < 1000) return `${Math.round(meters)} m`;
  return `${(meters / 1000).toFixed(2)} km`;
}

export function formatDuration(seconds?: number): string {
  if (seconds == null) return DASH;
  const s = Math.round(seconds);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  if (h > 0) return `${h}:${pad(m)}:${pad(sec)}`;
  return `${m}:${pad(sec)}`;
}

export function formatPace(secPerKm?: number): string {
  if (secPerKm == null || !isFinite(secPerKm) || secPerKm <= 0) return DASH;
  const m = Math.floor(secPerKm / 60);
  const s = Math.round(secPerKm % 60);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${m}:${pad(s)} /km`;
}

export { formatSwimmingPace } from "@iroha/shared/format";

export function formatElevation(meters?: number): string {
  if (meters == null) return DASH;
  return `${Math.round(meters)} m`;
}

export function formatHr(bpm?: number): string {
  if (bpm == null) return DASH;
  return `${Math.round(bpm)} bpm`;
}

// Full timestamp as `yyyy-MM-dd HH:mm:ss` in the activity's timezone. The
// sv-SE locale renders exactly this ISO-like shape with a 24-hour clock.
export function formatDate(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("sv-SE", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
      timeZone: timezone || IROHA_TIMEZONE || DEFAULT_TIMEZONE,
    }).format(d);
  } catch {
    return d.toISOString().slice(0, 19).replace("T", " ");
  }
}

// Date only as `yyyy-MM-dd` in the activity's timezone.
export function formatDateOnly(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) return iso;
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("sv-SE", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      timeZone: timezone || IROHA_TIMEZONE || DEFAULT_TIMEZONE,
    }).format(d);
  } catch {
    return d.toISOString().slice(0, 10);
  }
}

// Canonical month period as `yyyy-MM`. This deliberately keeps the machine
// period visible in selectors and chart tables instead of replacing it with a
// locale-dependent month name.
export function formatMonth(period?: string): string {
  if (!period) return DASH;
  const match = /^(\d{4})-(\d{1,2})$/.exec(period);
  if (!match) return period;
  const month = Number(match[2]);
  return month >= 1 && month <= 12
    ? `${match[1]}-${String(month).padStart(2, "0")}`
    : period;
}

// Short date for narrow chart axis labels (e.g. "Aug 4") where a full
// yyyy-MM-dd would always get truncated to an identical, useless prefix.
export function formatDateShort(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("en-US", {
      month: "short",
      day: "numeric",
      timeZone: timezone || IROHA_TIMEZONE || DEFAULT_TIMEZONE,
    }).format(d);
  } catch {
    return d.toISOString().slice(5, 10);
  }
}

// Normalize a sport type for display: iroha stores a mix of short lowercase
// codes (run, walk, ride) and raw Apple PascalCase (FitnessGaming,
// HighIntensityIntervalTraining). Render all of them as uniform Title Case.
export function formatSport(sport?: string): string {
  if (!sport) return DASH;
  return sport
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
