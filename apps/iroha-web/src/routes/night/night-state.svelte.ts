import { onMount } from "svelte";
import { replaceState } from "$app/navigation";
import { page } from "$app/state";
import {
  getSleepBounds,
  listSleep,
  listSleepAggregates,
  type SleepAggregateBucket,
  type SleepSession,
} from "$lib/api";
import { formatDateOnly, formatMonth } from "$lib/format";
import { currentYear, yearOptionsInRange } from "@iroha/shared/format/month";
import {
  currentCalendarScope,
  parseCalendarScope,
  readCalendarScope,
  scopeFromParts,
  serializeCalendarScope,
  writeCalendarScope,
  type DateBounds,
} from "@iroha/shared/format/scope";
import { IROHA_TIMEZONE } from "$lib/config";
import { createAsyncResource } from "$lib/asyncResource.svelte";

// All state, derivations, and data loading for the Night route, kept out of
// the .svelte file so the template isn't interleaved with ~350 lines of
// business logic. `theme` (a Svelte context lookup) stays in the component.
export function createNightState() {
  const PAGE_SIZE = 31;
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

  const sessionsResource = createAsyncResource<SleepSession[]>();
  const aggregatesResource = createAsyncResource<{
    years: SleepAggregateBucket[];
    months: SleepAggregateBucket[];
    lifetime: SleepAggregateBucket | null;
    bounds: DateBounds;
  }>();
  const sessions = $derived(sessionsResource.data ?? []);
  let selected = $state<SleepSession | null>(null);
  let loadingMore = $state(false);
  let cursor = $state<string | null>(null);
  let hasMore = $state(false);
  let loadMoreSentinel = $state<HTMLDivElement>();
  let nightListContainer = $state<HTMLDivElement>();
  const yearBuckets = $derived(aggregatesResource.data?.years ?? []);
  const monthBuckets = $derived(aggregatesResource.data?.months ?? []);
  const lifetimeBucket = $derived(aggregatesResource.data?.lifetime ?? null);
  // The real data range (fetched once, independent of the current
  // selection) -- drives the picker option lists and arrow-key clamping.
  // yearBuckets/monthBuckets above stay chart data; this is navigation only.
  const bounds = $derived(aggregatesResource.data?.bounds ?? {});
  let selectedYear = $state(initialYear);
  let selectedMonth = $state(initialMonth);
  let selectedStage = $state("Core");
  let hoveredStage = $state<string | null>(null);

  const loadedMainSleep = $derived(
    sessions.filter((session) => session.is_main_sleep),
  );
  const loadedAverageAsleep = $derived(
    loadedMainSleep.length
      ? loadedMainSleep.reduce(
          (total, session) => total + session.asleep_s,
          0,
        ) / loadedMainSleep.length
      : 0,
  );
  const loadedAverageEfficiency = $derived(
    loadedMainSleep.length
      ? loadedMainSleep.reduce(
          (total, session) => total + session.efficiency,
          0,
        ) / loadedMainSleep.length
      : 0,
  );
  const monthlyBuckets = $derived(monthBuckets.slice().reverse());
  const yearlyBuckets = $derived(yearBuckets.slice().reverse());
  const periodYears = $derived(yearOptionsInRange(bounds));
  // Full "YYYY-MM" period values (this page's own month convention), not
  // month.ts's bare 1-12 -- built directly from bounds rather than reusing
  // monthOptionsInRange, which returns the other convention. Newest first,
  // matching yearOptionsInRange's and monthOptionsInRange's own convention.
  const periodMonths = $derived.by(() => {
    if (!selectedYear || !bounds.min || !bounds.max) return [];
    const minYear = bounds.min.slice(0, 4);
    const maxYear = bounds.max.slice(0, 4);
    if (selectedYear < minYear || selectedYear > maxYear) return [];
    const start = selectedYear === minYear ? Number(bounds.min.slice(5, 7)) : 1;
    const end = selectedYear === maxYear ? Number(bounds.max.slice(5, 7)) : 12;
    const options: { value: string; label: string }[] = [];
    for (let month = end; month >= start; month--) {
      const period = `${selectedYear}-${String(month).padStart(2, "0")}`;
      options.push({
        value: period,
        label: formatPeriod(`${period}-01T00:00:00Z`, "month"),
      });
    }
    return options;
  });
  const visibleYears = $derived(
    selectedYear === ""
      ? yearlyBuckets
      : yearlyBuckets.filter(
          (bucket) => bucket.period.slice(0, 4) === selectedYear,
        ),
  );
  const visibleMonths = $derived(
    monthlyBuckets.filter((bucket) => {
      const period = bucket.period.slice(0, 7);
      return (
        (selectedYear === "" || period.slice(0, 4) === selectedYear) &&
        (selectedMonth === "" || period === selectedMonth)
      );
    }),
  );
  const focusedBucket = $derived(
    selectedMonth !== ""
      ? monthBuckets.find(
          (bucket) => bucket.period.slice(0, 7) === selectedMonth,
        )
      : selectedYear !== ""
        ? yearBuckets.find(
            (bucket) => bucket.period.slice(0, 4) === selectedYear,
          )
        : null,
  );
  const isPeriodFiltered = $derived(selectedYear !== "");
  const sleepSummary = $derived.by<SleepAggregateBucket | null>(() => {
    if (selectedMonth !== "") {
      return (
        monthBuckets.find(
          (bucket) => bucket.period.slice(0, 7) === selectedMonth,
        ) ?? null
      );
    }
    if (selectedYear !== "") {
      return (
        yearBuckets.find(
          (bucket) => bucket.period.slice(0, 4) === selectedYear,
        ) ?? null
      );
    }
    return lifetimeBucket;
  });
  const sleepScope = $derived(selectedMonth || selectedYear || "Lifetime");
  const rollupGranularity = $derived<"month" | "year">(
    selectedYear === "" ? "year" : "month",
  );
  const rollupBuckets = $derived.by(() => {
    if (selectedMonth !== "") return [];
    if (selectedYear !== "") {
      return monthBuckets.filter(
        (bucket) => bucket.period.slice(0, 4) === selectedYear,
      );
    }
    return yearBuckets;
  });
  const averageAsleep = $derived(
    sleepSummary?.average_asleep_s ?? loadedAverageAsleep,
  );
  const averageEfficiency = $derived(
    sleepSummary?.average_efficiency ?? loadedAverageEfficiency,
  );
  const heroEyebrow = $derived.by(() => {
    if (!isPeriodFiltered) {
      return `Last night · ${selected ? formatDateOnly(selected.wake_date) : ""}`;
    }
    const periodLabel = selectedMonth
      ? formatPeriod(`${selectedMonth}-01T00:00:00Z`, "month")
      : selectedYear;
    return `Selected night · ${periodLabel}`;
  });
  const nightsHeading = $derived(
    selectedMonth !== ""
      ? "Sessions in selected month"
      : "Recent session detail",
  );
  const monthMaxAsleep = $derived(
    Math.max(1, ...monthBuckets.map((bucket) => bucket.average_asleep_s)),
  );
  const yearMaxSessions = $derived(
    Math.max(1, ...yearBuckets.map((bucket) => bucket.session_count)),
  );
  const architectureStages = $derived([
    { name: "Core", value: selected?.core_s ?? 0, color: "var(--accent)" },
    { name: "Deep", value: selected?.deep_s ?? 0, color: "var(--accent-2)" },
    { name: "REM", value: selected?.rem_s ?? 0, color: "var(--ring-move)" },
    {
      name: "Awake",
      value: selected?.awake_s ?? 0,
      color: "var(--ring-exercise)",
    },
    ...(selected?.unspecified_s
      ? [
          {
            name: "Unspecified",
            value: selected.unspecified_s,
            color: "var(--text-muted)",
          },
        ]
      : []),
  ]);
  const activeStage = $derived(hoveredStage ?? selectedStage);
  const activeStageSeconds = $derived(
    activeStage === "Core"
      ? (selected?.core_s ?? 0)
      : activeStage === "Deep"
        ? (selected?.deep_s ?? 0)
        : activeStage === "REM"
          ? (selected?.rem_s ?? 0)
          : activeStage === "Awake"
            ? (selected?.awake_s ?? 0)
            : (selected?.unspecified_s ?? 0),
  );

  function syncPeriodUrl() {
    const url = new URL(window.location.href);
    writeCalendarScope(
      url.searchParams,
      selectedMonth
        ? (parseCalendarScope(selectedMonth) ?? scopeFromParts(selectedYear))
        : scopeFromParts(selectedYear),
    );
    if (url.href !== window.location.href) replaceState(url, page.state);
  }

  function selectSession(session: SleepSession) {
    selected = session;
  }

  function selectedScope(): { date?: string } {
    if (selectedMonth !== "") return { date: selectedMonth };
    if (selectedYear !== "") return { date: selectedYear };
    return {};
  }

  async function loadSessions(append = false) {
    if (append) {
      if (!hasMore || !cursor || loadingMore) return;
      loadingMore = true;
      try {
        const page = await listSleep({
          limit: PAGE_SIZE,
          cursor: cursor ?? undefined,
          ...selectedScope(),
        });
        sessionsResource.mutate((current) => [
          ...(current ?? []),
          ...page.items,
        ]);
        cursor = page.next_cursor;
        hasMore = page.has_more;
      } catch {
        // Load-more failures are retry-safe -- keep the rows already
        // showing rather than replacing a working view with an error.
      } finally {
        loadingMore = false;
      }
      return;
    }
    cursor = null;
    hasMore = false;
    const items = await sessionsResource.run(async () => {
      const page = await listSleep({ limit: PAGE_SIZE, ...selectedScope() });
      cursor = page.next_cursor;
      hasMore = page.has_more;
      return page.items;
    });
    if (items) {
      if (items[0]) selectSession(items[0]);
      else selected = null;
    }
  }

  function changeYear(value: string) {
    selectedYear = value;
    selectedMonth = "";
    syncPeriodUrl();
    void loadSessions(false);
  }

  function changeMonth(value: string) {
    selectedMonth = value;
    if (value) selectedYear = value.slice(0, 4);
    syncPeriodUrl();
    void loadSessions(false);
  }

  $effect(() => {
    if (!loadMoreSentinel || !hasMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting))
          void loadSessions(true);
      },
      { root: nightListContainer ?? null, rootMargin: "120px" },
    );
    observer.observe(loadMoreSentinel);
    return () => observer.disconnect();
  });

  async function loadAggregates() {
    const result = await aggregatesResource.run(async () => {
      const [years, months, lifetime, nextBounds] = await Promise.all([
        listSleepAggregates("year"),
        listSleepAggregates("month"),
        listSleepAggregates("lifetime"),
        getSleepBounds().catch(() => ({}) as DateBounds),
      ]);
      return {
        years: years.buckets,
        months: months.buckets,
        lifetime: lifetime.buckets[0] ?? null,
        bounds: nextBounds,
      };
    });
    if (!result) return;
    const validYears = new Set(yearOptionsInRange(result.bounds));
    if (selectedMonth) selectedYear = selectedMonth.slice(0, 4);
    if (selectedYear && !validYears.has(selectedYear)) selectedYear = "";
    if (
      selectedMonth &&
      !periodMonths.some((option) => option.value === selectedMonth)
    ) {
      selectedMonth = "";
    }
    syncPeriodUrl();
  }

  onMount(async () => {
    await loadAggregates();
    void loadSessions(false);
  });

  return {
    sessionsResource,
    aggregatesResource,
    get sessions() {
      return sessions;
    },
    get selected() {
      return selected;
    },
    get loadingMore() {
      return loadingMore;
    },
    get hasMore() {
      return hasMore;
    },
    get loadMoreSentinel() {
      return loadMoreSentinel;
    },
    set loadMoreSentinel(value: HTMLDivElement | undefined) {
      loadMoreSentinel = value;
    },
    get nightListContainer() {
      return nightListContainer;
    },
    set nightListContainer(value: HTMLDivElement | undefined) {
      nightListContainer = value;
    },
    get yearBuckets() {
      return yearBuckets;
    },
    get bounds() {
      return bounds;
    },
    get selectedYear() {
      return selectedYear;
    },
    get selectedMonth() {
      return selectedMonth;
    },
    get selectedStage() {
      return selectedStage;
    },
    set selectedStage(value: string) {
      selectedStage = value;
    },
    set hoveredStage(value: string | null) {
      hoveredStage = value;
    },
    get loadedMainSleep() {
      return loadedMainSleep;
    },
    get periodYears() {
      return periodYears;
    },
    get periodMonths() {
      return periodMonths;
    },
    get visibleYears() {
      return visibleYears;
    },
    get visibleMonths() {
      return visibleMonths;
    },
    get focusedBucket() {
      return focusedBucket;
    },
    get sleepSummary() {
      return sleepSummary;
    },
    get sleepScope() {
      return sleepScope;
    },
    get rollupGranularity() {
      return rollupGranularity;
    },
    get rollupBuckets() {
      return rollupBuckets;
    },
    get averageAsleep() {
      return averageAsleep;
    },
    get averageEfficiency() {
      return averageEfficiency;
    },
    get heroEyebrow() {
      return heroEyebrow;
    },
    get nightsHeading() {
      return nightsHeading;
    },
    get monthMaxAsleep() {
      return monthMaxAsleep;
    },
    get yearMaxSessions() {
      return yearMaxSessions;
    },
    get architectureStages() {
      return architectureStages;
    },
    get activeStage() {
      return activeStage;
    },
    get activeStageSeconds() {
      return activeStageSeconds;
    },
    changeYear,
    changeMonth,
    selectSession,
    loadSessions,
    formatPeriod,
  };
}

function formatPeriod(period: string, granularity: "month" | "year"): string {
  if (granularity === "year") return period.slice(0, 4);
  return formatMonth(period.slice(0, 7));
}
