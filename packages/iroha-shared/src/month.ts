export function currentMonth(date = new Date()): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
}

export function currentYear(date = new Date()): string {
  return String(date.getFullYear());
}

export const MONTH_OPTIONS = Array.from({ length: 12 }, (_, index) => ({
  value: String(index + 1),
  label: new Date(Date.UTC(2000, index, 1)).toLocaleDateString("en-US", {
    month: "long",
    timeZone: "UTC",
  }),
}));

export function yearOptions(
  firstYear = 2015,
  lastYear = new Date().getFullYear(),
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
  const shifted = new Date(year, monthNumber - 1 + delta, 1);
  return currentMonth(shifted);
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
