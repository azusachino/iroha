// Client-side derivations over the full, already-loaded activity/route
// snapshot. There is no live backend to re-query per filter change (this is
// a static export), and the dataset is personal-scale (hundreds to low
// thousands of rows), so recomputing on every year/sport change is fine.
import type { Activity, RouteFeature, SummaryBucket } from "./types";

export function yearsFromActivities(activities: Activity[]): string[] {
  const years = new Set<string>();
  for (const activity of activities) {
    const year = activity.started_at.slice(0, 4);
    if (year) years.add(year);
  }
  return Array.from(years).sort((a, b) => b.localeCompare(a));
}

// One bucket per "YYYY-MM" across every year present, feeding
// YearProgressChart's own per-year slicing.
export function monthlyBuckets(activities: Activity[]): SummaryBucket[] {
  const byKey = new Map<string, SummaryBucket>();
  for (const activity of activities) {
    const key = activity.started_at.slice(0, 7);
    let bucket = byKey.get(key);
    if (!bucket) {
      bucket = {
        key,
        activity_count: 0,
        distance_m: 0,
        distance_known_count: 0,
        distance_unknown_count: 0,
        duration_s: 0,
        elevation_gain_m: 0,
        moving_time_s: 0,
      };
      byKey.set(key, bucket);
    }
    bucket.activity_count += 1;
    if (activity.distance_m == null) bucket.distance_unknown_count += 1;
    else {
      bucket.distance_known_count += 1;
      bucket.distance_m += activity.distance_m;
    }
    bucket.duration_s += activity.duration_s ?? 0;
    bucket.elevation_gain_m += activity.elevation_gain_m ?? 0;
    bucket.moving_time_s =
      (bucket.moving_time_s ?? 0) + (activity.moving_time_s ?? 0);
  }
  return Array.from(byKey.values()).sort((a, b) => a.key.localeCompare(b.key));
}

export function filterByYearAndSport(
  activities: Activity[],
  year: string | null,
  sport: string | null,
): Activity[] {
  return activities.filter((activity) => {
    if (year && activity.started_at.slice(0, 4) !== year) return false;
    if (sport && activity.sport_type !== sport) return false;
    return true;
  });
}

export interface CityGroup {
  city: string;
  status: string;
  count: number;
  runCount: number;
  sports: Set<string>;
}

export function cityGroupsForRoutes(features: RouteFeature[]): CityGroup[] {
  const groups: Record<string, CityGroup> = {};
  for (const feature of features) {
    const city = feature.properties.city || "Unknown";
    const status =
      feature.properties.city_status ||
      (city === "Unknown" ? "pending" : "resolved");
    if (!groups[city]) {
      groups[city] = { city, status, count: 0, runCount: 0, sports: new Set() };
    }
    groups[city].count++;
    if (feature.properties.sport_type === "run") groups[city].runCount++;
    if (feature.properties.sport_type) {
      groups[city].sports.add(feature.properties.sport_type);
    }
  }
  return Object.values(groups).sort(
    (a, b) => b.runCount - a.runCount || b.count - a.count,
  );
}
