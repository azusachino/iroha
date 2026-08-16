import { onMount } from "svelte";
import { replaceState } from "$app/navigation";
import { page } from "$app/state";
import {
  getDailyBounds,
  listDaily,
  listDailyAggregates,
  type DailyRow,
  type DailyAggregateBucket,
} from "$lib/api";
import type { Ring } from "@iroha/shared/theme-ui/components/RingGauge.svelte";
import type { SmallMultiple } from "@iroha/shared/theme-ui/components/DailySmallMultiples.svelte";
import {
  formatDateOnly,
  formatMonth as formatCanonicalMonth,
} from "$lib/format";
import { currentYear, yearOptionsInRange } from "@iroha/shared/format/month";
import {
  currentCalendarScope,
  parseCalendarScope,
  readCalendarScope,
  scopeBounds,
  scopeFromParts,
  serializeCalendarScope,
  writeCalendarScope,
  type DateBounds,
} from "@iroha/shared/format/scope";
import { IROHA_TIMEZONE } from "$lib/config";
import { createAsyncResource } from "$lib/asyncResource.svelte";

type Gran = "day" | "month" | "year";

// A granularity-agnostic display row so the table + trends share one shape.
interface Disp {
  label: string;
  // Raw, unformatted period ("2026-08-10" / "2026-08" / "2026") -- what
  // drillIntoPeriod acts on; label is display-only and not safe to parse.
  period: string;
  days: number | null;
  move: number | null;
  exercise: number | null;
  stand: number | null;
  moveClosedPct: number | null;
  steps: number | null;
  distance: number | null;
  resting_hr: number | null;
  hrv_sdnn: number | null;
  spo2_avg: number | null;
  respiratory_rate: number | null;
  vo2max: number | null;
  body_mass_kg: number | null;
}

// All state, derivations, and data loading for the Patterns route, kept out
// of the .svelte file so the template isn't interleaved with ~450 lines of
// business logic. `theme` (a Svelte context lookup) stays in the component.
export function createPatternsState() {
  const monthlyResource = createAsyncResource<DailyAggregateBucket[]>();
  const yearlyResource = createAsyncResource<DailyAggregateBucket[]>();
  const daysResource = createAsyncResource<DailyRow[]>();
  const latestDayResource = createAsyncResource<DailyRow | null>();
  const monthly = $derived(monthlyResource.data ?? []);
  const yearly = $derived(yearlyResource.data ?? []);
  const dayRows = $derived(daysResource.data ?? []);
  const latestDay = $derived(latestDayResource.data ?? null);
  const error = $derived(
    monthlyResource.error ??
      latestDayResource.error ??
      yearlyResource.error ??
      daysResource.error ??
      null,
  );
  function granFromUrl(): Gran {
    const value = page.url.searchParams.get("gran");
    return value === "day" || value === "year" ? value : "month";
  }

  let gran = $state<Gran>(granFromUrl());
  // Set directly by drilling into a bar/row (see drillIntoPeriod), and reset
  // to "" -- meaning "default to the latest" -- by the day/month/year tabs
  // themselves, so a tab always means "zoom all the way out to this level."
  const defaultScope = currentCalendarScope("year", new Date(), IROHA_TIMEZONE);
  const requestedScope = readCalendarScope(page.url.searchParams, {
    fallback: defaultScope,
    allowDay: false,
  });
  const initialMonth =
    requestedScope.kind === "month"
      ? (serializeCalendarScope(requestedScope) as string)
      : "";
  const initialYear =
    requestedScope.kind === "year" || requestedScope.kind === "month"
      ? String(requestedScope.year)
      : requestedScope.kind === "lifetime"
        ? ""
        : currentYear(new Date(), IROHA_TIMEZONE);
  let selectedMonth = $state(initialMonth);
  let selectedYear = $state(initialYear);
  let rangeFrom = $state<string | undefined>(undefined);
  let rangeTo = $state<string | undefined>(undefined);
  let monthlyLoadedKey = "";
  let yearlyLoadedKey = "";
  let loadedDayMonth = "";

  // The real data range (fetched once, independent of the current
  // selection/granularity) -- drives which years/months are navigable, as
  // a continuous range rather than only the periods with a chart bucket.
  // That's what lets `scopedYear`/`scopedMonth` below equal the real
  // selection whenever it's genuinely within history, instead of silently
  // substituting whatever period happens to have data.
  let dailyBounds = $state<DateBounds>({});
  const availableYears = $derived(yearOptionsInRange(dailyBounds));
  const availableMonths = $derived.by(() => {
    if (!dailyBounds.min || !dailyBounds.max) return [];
    const months: string[] = [];
    let year = Number(dailyBounds.max.slice(0, 4));
    let month = Number(dailyBounds.max.slice(5, 7));
    const minYear = Number(dailyBounds.min.slice(0, 4));
    const minMonth = Number(dailyBounds.min.slice(5, 7));
    while (year > minYear || (year === minYear && month >= minMonth)) {
      months.push(`${year}-${String(month).padStart(2, "0")}`);
      month -= 1;
      if (month === 0) {
        month = 12;
        year -= 1;
      }
    }
    return months;
  });
  const scopedYear = $derived(
    availableYears.includes(selectedYear) ? selectedYear : "",
  );
  const activeYear = $derived(scopedYear || availableYears[0] || "");
  const monthsInScope = $derived(
    availableMonths.filter((month) => month.startsWith(activeYear)),
  );
  const periodYears = $derived(
    availableYears.map((year) => ({ value: year, label: year })),
  );
  const periodMonths = $derived(
    monthsInScope.map((month) => ({
      value: month,
      label: formatCanonicalMonth(month),
    })),
  );
  const scopedMonth = $derived(
    monthsInScope.includes(selectedMonth) ? selectedMonth : "",
  );
  const activeMonth = $derived(scopedMonth || monthsInScope[0] || "");
  const periodLabel = $derived(
    gran === "year"
      ? scopedYear || "Lifetime"
      : gran === "month"
        ? scopedMonth || `${activeYear} · all months`
        : activeMonth
          ? formatCanonicalMonth(activeMonth)
          : "No month selected",
  );

  // Hero uses the latest real ring day, independent of the chosen granularity.
  const latestRingDay = $derived(latestDay);
  const latestRing = $derived(latestRingDay?.ring);
  const ringData = $derived<Ring[]>(
    latestRing
      ? [
          {
            label: "Move",
            value: latestRing.move_kcal,
            goal: latestRing.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: latestRing.exercise_min,
            goal: latestRing.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: latestRing.stand_hours,
            goal: latestRing.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  function fmtPeriod(iso: string): string {
    const d = new Date(iso);
    if (gran === "year") return String(d.getUTCFullYear());
    return formatCanonicalMonth(iso.slice(0, 7));
  }
  function dayToDisp(r: DailyRow): Disp {
    const ring = r.ring;
    const hasRing = ring != null && ring.move_goal_kcal > 0;
    return {
      label: formatDateOnly(r.day),
      period: r.day.slice(0, 10),
      days: null,
      move: hasRing ? ring.move_kcal : null,
      exercise: hasRing ? ring.exercise_min : null,
      stand: hasRing ? ring.stand_hours : null,
      moveClosedPct: hasRing
        ? ring.move_kcal >= ring.move_goal_kcal
          ? 100
          : 0
        : null,
      steps: r.steps ?? null,
      distance: r.distance_km ?? null,
      resting_hr: r.resting_hr ?? null,
      hrv_sdnn: r.hrv_sdnn ?? null,
      spo2_avg: r.spo2_avg ?? null,
      respiratory_rate: r.respiratory_rate ?? null,
      vo2max: r.vo2max ?? null,
      body_mass_kg: r.body_mass_kg ?? null,
    };
  }
  function aggToDisp(b: DailyAggregateBucket): Disp {
    const metricValue = (metric: string): number | null =>
      b.metrics.find((item) => item.metric === metric)?.value ?? null;
    const move = b.move_kcal_avg === 0 ? null : b.move_kcal_avg;
    return {
      label: fmtPeriod(b.period),
      period: gran === "year" ? b.period.slice(0, 4) : b.period.slice(0, 7),
      days: b.days,
      move,
      exercise: b.exercise_min_avg === 0 ? null : b.exercise_min_avg,
      stand: b.stand_hours_avg === 0 ? null : b.stand_hours_avg,
      moveClosedPct: move == null ? null : Math.round(b.move_closed_pct),
      steps: metricValue("steps"),
      distance: metricValue("distance_km"),
      resting_hr: metricValue("resting_hr"),
      hrv_sdnn: metricValue("hrv_sdnn"),
      spo2_avg: metricValue("spo2_avg"),
      respiratory_rate: metricValue("respiratory_rate"),
      vo2max: metricValue("vo2max"),
      body_mass_kg: metricValue("body_mass_kg"),
    };
  }

  // Chronological (oldest→newest) for sparklines; table is its reverse.
  const chrono = $derived.by<Disp[]>(() => {
    if (gran === "day") {
      return [...dayRows]
        .filter((row) => !activeMonth || row.day.startsWith(activeMonth))
        .reverse()
        .map(dayToDisp);
    }
    if (gran === "month") {
      return monthly
        .filter(
          (bucket) =>
            (!activeYear || bucket.period.startsWith(activeYear)) &&
            (!scopedMonth || bucket.period.startsWith(scopedMonth)),
        )
        .map(aggToDisp);
    }
    return yearly
      .filter((bucket) => !scopedYear || bucket.period.startsWith(scopedYear))
      .map(aggToDisp);
  });
  const table = $derived([...chrono].reverse());
  const aggregated = $derived(gran !== "day");

  function ser(pick: (d: Disp) => number | null): (number | null)[] {
    return chrono.map((d) => pick(d));
  }
  const trendCharts = $derived.by<SmallMultiple[]>(() => [
    {
      label: "Steps",
      values: ser((d) => d.steps),
      color: "var(--accent)",
      unit: "steps",
    },
    {
      label: "Resting HR",
      values: ser((d) => d.resting_hr),
      color: "var(--sport-run)",
      unit: "bpm",
    },
    {
      label: "HRV",
      values: ser((d) => d.hrv_sdnn),
      color: "var(--sport-cycling)",
      unit: "ms",
    },
    {
      label: "Move ring closed",
      values: ser((d) => d.moveClosedPct),
      color: "var(--ring-move)",
      unit: "%",
    },
  ]);
  function fmt(v: number | null | undefined, digits: number): string {
    if (typeof v !== "number" || !Number.isFinite(v)) return "—";
    return v.toLocaleString(undefined, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    });
  }

  async function loadMonthly() {
    const key = selectedYear || "lifetime";
    if (monthlyLoadedKey === key) return;
    const result = await monthlyResource.run(() =>
      listDailyAggregates(
        "month",
        selectedYear ? { date: selectedYear } : {},
      ).then((r) => r.buckets),
    );
    if (result) monthlyLoadedKey = key;
  }

  async function loadBounds() {
    try {
      dailyBounds = await getDailyBounds();
    } catch {
      dailyBounds = {};
    }
  }

  async function loadLatestDay() {
    await latestDayResource.run(async () => {
      const result = await listDaily({ limit: 1 });
      return result.items[0] ?? null;
    });
  }

  async function loadYearly() {
    const key = selectedYear || "lifetime";
    if (yearlyLoadedKey === key) return;
    const result = await yearlyResource.run(() =>
      listDailyAggregates(
        "year",
        selectedYear ? { date: selectedYear } : {},
      ).then((r) => r.buckets),
    );
    if (result) yearlyLoadedKey = key;
  }

  async function loadDays(month: string) {
    if (!month || loadedDayMonth === month || daysResource.loading) return;
    const bounds = scopeBounds(parseCalendarScope(month)!);
    if (!bounds) return;
    rangeFrom = bounds.from;
    rangeTo = bounds.to;
    const result = await daysResource.run(async () => {
      const r = await listDaily({ date: month, limit: 31 });
      return r.items;
    });
    if (result) loadedDayMonth = month;
  }

  // A day/month/year tab always means "zoom all the way out to this level" --
  // reset any scope a bar/row click drilled into, rather than keeping it.
  async function changeGranularity(value: Gran) {
    gran = value;
    if (value === "year") await loadYearly();
    if (value === "day") await loadDays(activeMonth);
    syncUrl();
  }

  function selectYear(value: string) {
    selectedYear = value;
    selectedMonth = "";
    monthlyLoadedKey = "";
    yearlyLoadedKey = "";
    void loadMonthly();
    if (gran === "year") void loadYearly();
    if (gran === "day") void loadDays(activeMonth);
    syncUrl();
  }

  function selectMonth(value: string) {
    selectedMonth = value;
    if (value && selectedYear !== value.slice(0, 4)) {
      selectedYear = value.slice(0, 4);
      monthlyLoadedKey = "";
      yearlyLoadedKey = "";
      void loadMonthly();
      if (gran === "year") void loadYearly();
    }
    if (gran === "day") void loadDays(activeMonth);
    syncUrl();
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    url.searchParams.set("gran", gran);
    writeCalendarScope(
      url.searchParams,
      scopedMonth
        ? (parseCalendarScope(scopedMonth) ?? scopeFromParts(scopedYear))
        : scopeFromParts(scopedYear),
    );
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  // Clicking a bar or table row zooms in one level: a year scopes month
  // view to it, a month scopes day view to it. Day is already the finest
  // granularity, so a day period has nothing further to drill into.
  function drillIntoPeriod(period: string) {
    if (gran === "year") {
      selectedYear = period;
      gran = "month";
    } else if (gran === "month") {
      selectedMonth = period;
      gran = "day";
      void loadDays(period);
    }
    syncUrl();
  }

  function drillIntoIndex(index: number) {
    const period = chrono[index]?.period;
    if (period) drillIntoPeriod(period);
  }

  onMount(async () => {
    await Promise.all([loadMonthly(), loadLatestDay(), loadBounds()]);
    if (gran === "year") await loadYearly();
    if (gran === "day") await loadDays(activeMonth);
  });

  // Only the resources the current granularity actually fetches -- yearly
  // and day-row resources are never run outside their own tab, so including
  // them unconditionally would leave the boundary waiting on data that's
  // never coming.
  const activeResources = $derived(
    gran === "year"
      ? [monthlyResource, latestDayResource, yearlyResource]
      : gran === "day"
        ? [monthlyResource, latestDayResource, daysResource]
        : [monthlyResource, latestDayResource],
  );

  return {
    get gran() {
      return gran;
    },
    get error() {
      return error;
    },
    get chrono() {
      return chrono;
    },
    get table() {
      return table;
    },
    get aggregated() {
      return aggregated;
    },
    get trendCharts() {
      return trendCharts;
    },
    get ringData() {
      return ringData;
    },
    get latestRingDay() {
      return latestRingDay;
    },
    get dayRows() {
      return dayRows;
    },
    get activeResources() {
      return activeResources;
    },
    get periodYears() {
      return periodYears;
    },
    get periodMonths() {
      return periodMonths;
    },
    get selectedYear() {
      return selectedYear;
    },
    get selectedMonth() {
      return selectedMonth;
    },
    get activeYear() {
      return activeYear;
    },
    get activeMonth() {
      return activeMonth;
    },
    get dailyBounds() {
      return dailyBounds;
    },
    get rangeFrom() {
      return rangeFrom;
    },
    get rangeTo() {
      return rangeTo;
    },
    get periodLabel() {
      return periodLabel;
    },
    changeGranularity,
    selectYear,
    selectMonth,
    drillIntoPeriod,
    drillIntoIndex,
    fmt,
  };
}
