import { DEFAULT_TIMEZONE, todayInTimezone } from "@iroha/shared/date";

function previousDay(day: string): string {
  const date = new Date(`${day}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() - 1);
  return date.toISOString().slice(0, 10);
}

// Counts consecutive calendar days ending today. Multiple activities on the
// same day count once; a streak is inactive until there is an activity today.
export function currentActivityStreak(
  startedAt: string[],
  now: Date = new Date(),
  timezone = DEFAULT_TIMEZONE,
): number {
  const activeDays = new Set<string>();
  for (const value of startedAt) {
    const date = new Date(value);
    if (!Number.isNaN(date.getTime()))
      activeDays.add(todayInTimezone(date, timezone));
  }

  let streak = 0;
  for (
    let day = todayInTimezone(now, timezone);
    activeDays.has(day);
    day = previousDay(day)
  ) {
    streak++;
  }
  return streak;
}
