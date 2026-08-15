export const DEFAULT_TIMEZONE = "Asia/Tokyo";

export function isValidTimezone(timezone: string): boolean {
  try {
    new Intl.DateTimeFormat("en-US", { timeZone: timezone }).format();
    return true;
  } catch {
    return false;
  }
}

export function resolveTimezone(timezone?: string): string {
  return timezone && isValidTimezone(timezone) ? timezone : DEFAULT_TIMEZONE;
}

/**
 * Return the canonical calendar day for an instant in Iroha's personal
 * timezone. Date-only API values must not be passed through this function.
 */
export function todayInTimezone(
  date = new Date(),
  timezone = DEFAULT_TIMEZONE,
): string {
  const parts = new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: timezone,
  }).formatToParts(date);
  const part = (type: string) => parts.find((item) => item.type === type)?.value;
  return `${part("year")}-${part("month")}-${part("day")}`;
}
