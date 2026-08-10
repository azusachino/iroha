// Presentation helpers. All inputs may be undefined; missing data renders as
// an em dash so the UI degrades gracefully.

const DASH = "—";

export function boundPercent(value?: number | null): number {
  if (value == null || !Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

export function formatPercent(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return DASH;
  return `${Math.round(boundPercent(value))}%`;
}

// A "completed" item's position IS its total by definition, even when the
// provider never reported a numeric total -- Bangumi in particular often
// omits it for an otherwise-finished season. Without this, a completed
// item with no recorded total looks identical to one with unknown
// progress: same bare count, same empty-looking bar.
function effectiveTotal(
  status?: string | null,
  position?: number | null,
  total?: number | null,
): number | undefined {
  if (total != null && Number.isFinite(total) && total > 0) return total;
  if (status === "completed" && position != null && Number.isFinite(position)) {
    return position;
  }
  return undefined;
}

// A percentage implies a known total; most media (ongoing manga, an
// unfinished anime season) never has one. formatProgressCount shows what's
// actually known instead: a done/all count when a total exists or can be
// inferred from completion, just the done count otherwise, and only falls
// back to the dash when there's no position at all.
export function formatProgressCount(
  position?: number | null,
  total?: number | null,
  unit?: string | null,
  status?: string | null,
): string {
  if (position == null || !Number.isFinite(position)) return DASH;
  const suffix = unit ? ` ${unit}` : "";
  const knownTotal = effectiveTotal(status, position, total);
  if (knownTotal != null) {
    return `${position}/${knownTotal}${suffix}`;
  }
  return `${position}${suffix}`;
}

// The bar/ring-fill counterpart to formatProgressCount: an explicit percent
// wins if present, otherwise derive one from position/total (falling back
// to a completed item's own position as its total, same as above), and 0
// only when truly nothing is known.
export function progressPercent(
  status?: string | null,
  position?: number | null,
  total?: number | null,
  percent?: number | null,
): number {
  if (percent != null && Number.isFinite(percent)) return boundPercent(percent);
  const knownTotal = effectiveTotal(status, position, total);
  if (position != null && Number.isFinite(position) && knownTotal != null) {
    return boundPercent((position / knownTotal) * 100);
  }
  return 0;
}

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
      timeZone: timezone || undefined,
    }).format(d);
  } catch {
    return d.toISOString().slice(0, 19).replace("T", " ");
  }
}

// Date only as `yyyy-MM-dd` in the activity's timezone.
export function formatDateOnly(iso?: string, timezone?: string): string {
  if (!iso) return DASH;
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  try {
    return new Intl.DateTimeFormat("sv-SE", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      timeZone: timezone || undefined,
    }).format(d);
  } catch {
    return d.toISOString().slice(0, 10);
  }
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
      timeZone: timezone || undefined,
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
