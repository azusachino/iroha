const DAY_MS = 24 * 60 * 60 * 1000;

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function dayKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

// Counts consecutive calendar days ending today. Multiple activities on the
// same day count once; a streak is inactive until there is an activity today.
export function currentActivityStreak(
  startedAt: string[],
  now: Date = new Date(),
): number {
  const activeDays = new Set<string>();
  for (const value of startedAt) {
    const date = new Date(value);
    if (!Number.isNaN(date.getTime())) activeDays.add(dayKey(startOfDay(date)));
  }

  let streak = 0;
  for (
    let date = startOfDay(now);
    activeDays.has(dayKey(date));
    date = new Date(date.getTime() - DAY_MS)
  ) {
    streak++;
  }
  return streak;
}
