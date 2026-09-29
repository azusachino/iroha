import type { ActivitySummaryBucket } from "../../domain/activity";
import { shiftMonth } from "../../format/month";

export interface MonthlyDistancePoint {
  key: string;
  distance_m: number;
}

export function buildMonthlyDistanceSeries(
  buckets: Pick<ActivitySummaryBucket, "key" | "distance_m">[],
  endMonth: string,
  months: number,
): MonthlyDistancePoint[] {
  if (buckets.length === 0 || months < 1) return [];

  const distanceByMonth = new Map(
    buckets.map(({ key, distance_m }) => [key, distance_m]),
  );
  const firstMonth = shiftMonth(endMonth, -(months - 1));

  return Array.from({ length: months }, (_, index) => {
    const key = shiftMonth(firstMonth, index);
    return { key, distance_m: distanceByMonth.get(key) ?? 0 };
  });
}
