import { todayInTimezone } from "./date";
import type { DateBounds } from "./scope";

export function currentMonth(date = new Date(), timezone?: string): string {
  return todayInTimezone(date, timezone).slice(0, 7);
}

export function currentYear(date = new Date(), timezone?: string): string {
  return todayInTimezone(date, timezone).slice(0, 4);
}

export const MONTH_OPTIONS = Array.from({ length: 12 }, (_, index) => ({
  value: String(index + 1),
  label: new Date(Date.UTC(2000, index, 1)).toLocaleDateString("en-US", {
    month: "long",
    timeZone: "UTC",
  }),
}));

// Years with a real record, newest first. Empty when the domain has no
// data yet -- callers should fall back to lifetime-only in that case.
export function yearOptionsInRange(bounds: DateBounds): string[] {
  if (!bounds.min || !bounds.max) return [];
  const minYear = Number(bounds.min.slice(0, 4));
  const maxYear = Number(bounds.max.slice(0, 4));
  return Array.from({ length: Math.max(0, maxYear - minYear + 1) }, (_, index) =>
    String(maxYear - index),
  );
}

// Months within the domain's real range for the given year -- the full
// twelve for a year strictly inside the range, clipped to the actual
// min/max month at the boundary years. Empty when the year itself is
// outside the range (including when the domain has no data yet).
export function monthOptionsInRange(
  year: string,
  bounds: DateBounds,
): typeof MONTH_OPTIONS {
  if (!bounds.min || !bounds.max) return [];
  const minYear = bounds.min.slice(0, 4);
  const maxYear = bounds.max.slice(0, 4);
  if (year < minYear || year > maxYear) return [];
  const start = year === minYear ? Number(bounds.min.slice(5, 7)) : 1;
  const end = year === maxYear ? Number(bounds.max.slice(5, 7)) : 12;
  return MONTH_OPTIONS.filter((option) => {
    const month = Number(option.value);
    return month >= start && month <= end;
  });
}

export function yearOptions(
  firstYear = 2015,
  lastYear = Number(currentYear()),
): string[] {
  return Array.from(
    { length: Math.max(0, lastYear - firstYear + 1) },
    (_, index) => String(lastYear - index),
  );
}

export function canonicalMonth(
  value: string | null | undefined,
  fallback = currentMonth(),
): string {
  return /^\d{4}-(?:0[1-9]|1[0-2])$/.test(value ?? "")
    ? (value as string)
    : fallback;
}

export function shiftMonth(month: string, delta: number): string {
  const [year, monthNumber] = month.split("-").map(Number);
  const shifted = new Date(Date.UTC(year, monthNumber - 1 + delta, 1));
  return `${shifted.getUTCFullYear()}-${String(shifted.getUTCMonth() + 1).padStart(2, "0")}`;
}

export function shiftMonthWithin(
  month: string,
  delta: number,
  maximum = currentMonth(),
): string {
  const candidate = shiftMonth(month, delta);
  return delta > 0 && candidate > maximum ? maximum : candidate;
}

export function monthBounds(month: string): { from: string; to: string } {
  const [year, monthNumber] = month.split("-").map(Number);
  const nextMonth = new Date(Date.UTC(year, monthNumber, 1));
  return {
    from: `${month}-01`,
    to: `${nextMonth.getUTCFullYear()}-${String(nextMonth.getUTCMonth() + 1).padStart(2, "0")}-01`,
  };
}

export function formatMonth(month: string): string {
  return new Date(`${month}-01T00:00:00Z`).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    timeZone: "UTC",
  });
}
