import { formatSport } from "$lib/format";
import { isDistanceSport } from "@iroha/shared/domain/sport";
import type { Activity } from "./types";

export type PublicSortKey =
  "started_at" | "sport_type" | "distance_m" | "avg_pace_s_per_km";

export type PublicSortEntry = { id: PublicSortKey; desc: boolean };

export type PublicActivityTotals = {
  activity_count: number;
  distance_m: number;
  distance_known_count: number;
  distance_unknown_count: number;
  duration_s: number;
  moving_time_s: number;
};

export type PublicSportBucket = {
  key: string;
  activity_count: number;
  distance_m: number;
  duration_s: number;
  moving_time_s: number;
};

export const MONTH_LABELS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
] as const;

export function isNonDistanceSport(
  sport?: string,
  distanceM?: number,
): boolean {
  if (!sport) return true;
  if (distanceM == null || distanceM <= 0) return true;
  const s = sport.toLowerCase();
  return !["run", "walk", "hike", "ride", "cycl", "swim"].some((k) =>
    s.includes(k),
  );
}

export function isCycling(sport?: string): boolean {
  const s = sport?.toLowerCase() ?? "";
  return s.includes("ride") || s.includes("cycl") || s.includes("bik");
}

export function isSwimming(sport?: string): boolean {
  return (sport?.toLowerCase() ?? "").includes("swim");
}

export function hasMeaningfulActivityTitle(activity: Activity): boolean {
  const title = formatSport(activity.title);
  const sport = formatSport(activity.sport_type);
  const genericTitles: Record<string, string[]> = {
    Run: ["Run", "Running"],
    Walk: ["Walk", "Walking"],
    Ride: ["Ride", "Riding", "Cycling"],
    Swim: ["Swim", "Swimming"],
  };
  return title !== "—" && !(genericTitles[sport] ?? [sport]).includes(title);
}

export function formatCyclingSpeed(
  distanceM?: number,
  durationS?: number,
): string {
  if (distanceM == null || durationS == null || durationS <= 0) return "—";
  return `${(distanceM / 1000 / (durationS / 3600)).toFixed(1)} km/h`;
}

export function formatSwimmingPace(
  distanceM?: number,
  durationS?: number,
): string {
  if (distanceM == null || durationS == null || durationS <= 0) return "—";
  const pace100m = durationS / (distanceM / 100);
  const m = Math.floor(pace100m / 60);
  const s = Math.round(pace100m % 60);
  return `${m}:${String(s).padStart(2, "0")} /100m`;
}

export function cityLabel(city: string, status?: string): string {
  if (status === "pending") return "Location pending";
  if (status === "unknown" || city === "Unknown") return "Location unavailable";
  return city;
}

export function summarizePublicActivities(
  scoped: Activity[],
): PublicActivityTotals {
  return scoped.reduce(
    (acc, activity) => {
      const hasDistance = activity.distance_m != null;
      const isUnknownDistance =
        !hasDistance && isDistanceSport(activity.sport_type);
      return {
        activity_count: acc.activity_count + 1,
        distance_m: acc.distance_m + (activity.distance_m ?? 0),
        distance_known_count: acc.distance_known_count + (hasDistance ? 1 : 0),
        distance_unknown_count:
          acc.distance_unknown_count + (isUnknownDistance ? 1 : 0),
        duration_s: acc.duration_s + (activity.duration_s ?? 0),
        moving_time_s: acc.moving_time_s + (activity.moving_time_s ?? 0),
      };
    },
    {
      activity_count: 0,
      distance_m: 0,
      distance_known_count: 0,
      distance_unknown_count: 0,
      duration_s: 0,
      moving_time_s: 0,
    },
  );
}

export function buildPublicSportBuckets(
  scoped: Activity[],
): PublicSportBucket[] {
  const buckets = new Map<string, PublicSportBucket>();
  for (const activity of scoped) {
    const bucket = buckets.get(activity.sport_type) ?? {
      key: activity.sport_type,
      activity_count: 0,
      distance_m: 0,
      duration_s: 0,
      moving_time_s: 0,
    };
    bucket.activity_count += 1;
    bucket.distance_m += activity.distance_m ?? 0;
    bucket.duration_s += activity.duration_s ?? 0;
    bucket.moving_time_s += activity.moving_time_s ?? 0;
    buckets.set(activity.sport_type, bucket);
  }
  return Array.from(buckets.values()).toSorted(
    (a, b) => b.activity_count - a.activity_count,
  );
}

export function sortPublicActivities(
  activities: Activity[],
  sorting: PublicSortEntry[],
): Activity[] {
  const sort = sorting[0];
  if (!sort) return [...activities];
  const numeric = ["distance_m", "avg_pace_s_per_km"].includes(sort.id);
  const key = sort.id as keyof Activity;
  return activities.toSorted((a, b) => {
    const left = numeric ? Number(a[key] ?? -Infinity) : String(a[key] ?? "");
    const right = numeric ? Number(b[key] ?? -Infinity) : String(b[key] ?? "");
    const result = left < right ? -1 : left > right ? 1 : 0;
    return sort.desc ? -result : result;
  });
}
