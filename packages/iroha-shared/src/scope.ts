import { DEFAULT_TIMEZONE, todayInTimezone } from "./date";

export type CalendarScope =
  | { kind: "lifetime" }
  | { kind: "year"; year: number }
  | { kind: "month"; year: number; month: number }
  | { kind: "day"; year: number; month: number; day: number };

export type CalendarScopeKind = CalendarScope["kind"];

export interface ScopeBounds {
  from: string;
  to: string;
}

export interface ScopeQueryOptions {
  fallback: CalendarScope;
  allowLifetime?: boolean;
  allowDay?: boolean;
}

const YEAR_PATTERN = /^(\d{4})$/;
const MONTH_PATTERN = /^(\d{4})-(\d{2})$/;
const DAY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

function isValidDate(year: number, month: number, day: number): boolean {
  const value = new Date(Date.UTC(year, month - 1, day));
  return (
    value.getUTCFullYear() === year &&
    value.getUTCMonth() === month - 1 &&
    value.getUTCDate() === day
  );
}

function parseYear(value: string): CalendarScope | null {
  const match = YEAR_PATTERN.exec(value);
  if (!match) return null;
  const year = Number(match[1]);
  return year > 0 ? { kind: "year", year } : null;
}

function parseMonth(value: string): CalendarScope | null {
  const match = MONTH_PATTERN.exec(value);
  if (!match) return null;
  const year = Number(match[1]);
  const month = Number(match[2]);
  return year > 0 && month >= 1 && month <= 12
    ? { kind: "month", year, month }
    : null;
}

function parseDay(value: string): CalendarScope | null {
  const match = DAY_PATTERN.exec(value);
  if (!match) return null;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  return year > 0 && isValidDate(year, month, day)
    ? { kind: "day", year, month, day }
    : null;
}

export function parseCalendarScope(
  value: string | null | undefined,
  options: { allowDay?: boolean } = {},
): CalendarScope | null {
  if (!value) return null;
  const parsed =
    parseDay(value) ?? parseMonth(value) ?? parseYear(value);
  if (parsed?.kind === "day" && options.allowDay === false) return null;
  return parsed;
}

export function serializeCalendarScope(scope: CalendarScope): string | null {
  if (scope.kind === "lifetime") return null;
  const year = String(scope.year).padStart(4, "0");
  if (scope.kind === "year") return year;
  const month = String(scope.month).padStart(2, "0");
  if (scope.kind === "month") return `${year}-${month}`;
  return `${year}-${month}-${String(scope.day).padStart(2, "0")}`;
}

export function readCalendarScope(
  params: URLSearchParams,
  options: ScopeQueryOptions,
): CalendarScope {
  const scope = params.get("scope");
  if (options.allowLifetime !== false && scope === "lifetime") {
    return { kind: "lifetime" };
  }

  const date = parseCalendarScope(params.get("date"), {
    allowDay: options.allowDay,
  });
  if (date && (date.kind !== "day" || options.allowDay !== false)) return date;

  // Legacy URLs are read-only compatibility input. All writes use date/scope.
  const legacyMonth = parseCalendarScope(params.get("month"), {
    allowDay: false,
  });
  if (legacyMonth?.kind === "month") return legacyMonth;
  const legacyMonthNumber = params.get("month");
  const legacyYearValue = params.get("year");
  if (/^(?:[1-9]|1[0-2])$/.test(legacyMonthNumber ?? "")) {
    const legacyYear = parseCalendarScope(legacyYearValue, {
      allowDay: false,
    });
    if (legacyYear?.kind === "year") {
      return {
        kind: "month",
        year: legacyYear.year,
        month: Number(legacyMonthNumber),
      };
    }
  }
  const legacyYear = parseCalendarScope(params.get("year"), {
    allowDay: false,
  });
  if (legacyYear?.kind === "year") return legacyYear;
  return options.fallback;
}

export function writeCalendarScope(
  params: URLSearchParams,
  scope: CalendarScope,
): void {
  params.delete("month");
  params.delete("year");
  if (scope.kind === "lifetime") {
    params.delete("date");
    params.set("scope", "lifetime");
    return;
  }
  params.delete("scope");
  params.set("date", serializeCalendarScope(scope) as string);
}

export function scopeBounds(scope: CalendarScope): ScopeBounds | null {
  if (scope.kind === "lifetime") return null;
  let from: Date;
  if (scope.kind === "year") {
    from = new Date(Date.UTC(scope.year, 0, 1));
  } else if (scope.kind === "month") {
    from = new Date(Date.UTC(scope.year, scope.month - 1, 1));
  } else {
    from = new Date(Date.UTC(scope.year, scope.month - 1, scope.day));
  }
  const to = new Date(from);
  if (scope.kind === "year") to.setUTCFullYear(to.getUTCFullYear() + 1);
  else if (scope.kind === "month") to.setUTCMonth(to.getUTCMonth() + 1);
  else to.setUTCDate(to.getUTCDate() + 1);
  return { from: formatDate(from), to: formatDate(to) };
}

export function scopeFromParts(year: string, month = ""): CalendarScope {
  if (!year) return { kind: "lifetime" };
  if (/^\d{4}-(?:0[1-9]|1[0-2])$/.test(month)) {
    return parseCalendarScope(month, { allowDay: false }) ?? { kind: "lifetime" };
  }
  const parsed = parseCalendarScope(month ? `${year}-${month.padStart(2, "0")}` : year);
  return parsed ?? { kind: "lifetime" };
}

export function scopeParts(scope: CalendarScope): { year: string; month: string } {
  if (scope.kind === "lifetime") return { year: "", month: "" };
  return {
    year: String(scope.year),
    month: scope.kind === "month" ? String(scope.month) : "",
  };
}

export function currentCalendarScope(
  kind: Exclude<CalendarScopeKind, "lifetime"> = "month",
  date = new Date(),
  timezone = DEFAULT_TIMEZONE,
): CalendarScope {
  const today = todayInTimezone(date, timezone);
  const [year, month, day] = today.split("-").map(Number);
  if (kind === "year") return { kind, year };
  if (kind === "day") return { kind, year, month, day };
  return { kind: "month", year, month };
}

export function isFutureScope(
  scope: CalendarScope,
  now = new Date(),
  timezone = DEFAULT_TIMEZONE,
): boolean {
  const value = scope as {
    year?: number;
    month?: number;
    day?: number;
  };
  if (value.year == null) return false;
  const today = currentCalendarScope("day", now, timezone) as {
    year: number;
    month: number;
    day: number;
  };
  if (value.month == null) return value.year > today.year;
  if (value.day == null) {
    return value.year > today.year ||
      (value.year === today.year && value.month > today.month);
  }
  return value.year > today.year ||
    (value.year === today.year &&
      (value.month > today.month ||
        (value.month === today.month && value.day > today.day)));
}

export function shiftCalendarScope(
  scope: CalendarScope,
  delta: number,
  now = new Date(),
  timezone = DEFAULT_TIMEZONE,
): CalendarScope {
  if (scope.kind === "lifetime") return scope;
  let value: Date;
  if (scope.kind === "year") {
    value = new Date(Date.UTC(scope.year, 0, 1));
  } else if (scope.kind === "month") {
    value = new Date(Date.UTC(scope.year, scope.month - 1, 1));
  } else {
    value = new Date(Date.UTC(scope.year, scope.month - 1, scope.day));
  }
  if (scope.kind === "year") value.setUTCFullYear(value.getUTCFullYear() + delta);
  else if (scope.kind === "month") value.setUTCMonth(value.getUTCMonth() + delta);
  else value.setUTCDate(value.getUTCDate() + delta);

  let next: CalendarScope;
  if (scope.kind === "year") next = { kind: "year", year: value.getUTCFullYear() };
  else if (scope.kind === "month") {
    next = {
      kind: "month",
      year: value.getUTCFullYear(),
      month: value.getUTCMonth() + 1,
    };
  } else {
    next = {
      kind: "day",
      year: value.getUTCFullYear(),
      month: value.getUTCMonth() + 1,
      day: value.getUTCDate(),
    };
  }
  return isFutureScope(next, now, timezone)
    ? currentCalendarScope(scope.kind, now, timezone)
    : next;
}

function formatDate(value: Date): string {
  return [
    value.getUTCFullYear(),
    String(value.getUTCMonth() + 1).padStart(2, "0"),
    String(value.getUTCDate()).padStart(2, "0"),
  ].join("-");
}
