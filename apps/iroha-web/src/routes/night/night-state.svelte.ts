import { onMount } from "svelte";
import { replaceState } from "$app/navigation";
import { page } from "$app/state";
import {
  getSleepBounds,
  listSleep,
  listSleepAggregates,
  type SleepAggregateBucket,
  type SleepSession,
  type Page,
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

  let selectedYear = $state(initialYear);
  let selectedMonth = $state(initialMonth);
  const sessionsResource = createAsyncResource<{
    date?: string;
    value: Page<SleepSession>;
  }>();
  const yearResource = createAsyncResource<SleepAggregateBucket[]>();
  const monthResource = createAsyncResource<SleepAggregateBucket[]>();
  const lifetimeResource = createAsyncResource<SleepAggregateBucket[]>();
  const boundsResource = createAsyncResource<DateBounds>();
  const aggregatesResource = $derived(
    selectedMonth
      ? monthResource
      : selectedYear
        ? yearResource
        : lifetimeResource,
  );
  const sessions = $derived(sessionsResource.data?.value.items ?? []);
  let selected = $state<SleepSession | null>(null);
  const pageResource = createAsyncResource<Page<SleepSession>>();
  let pendingPage = $state<typeof sessionsResource.data>(null);
  const loadMoreError = $derived(
    pendingPage === sessionsResource.data ? pageResource.error : null,
  );
  const loadingMore = $derived(
    pageResource.loading &&
      pendingPage != null &&
      pendingPage === sessionsResource.data &&
      pendingPage.date === selectedScope().date,
  );
  const hasMore = $derived(
    (sessionsResource.data?.value.has_more ?? false) &&
      !sessionsResource.loading &&
      sessionsResource.data?.date === selectedScope().date,
  );
  let loadMoreSentinel = $state<HTMLDivElement>();
  let nightListContainer = $state<HTMLDivElement>();
  const yearBuckets = $derived(yearResource.data ?? []);
  const monthBuckets = $derived(monthResource.data ?? []);
  const lifetimeBucket = $derived(lifetimeResource.data?.[0] ?? null);
  // The real data range (fetched once, independent of the current
  // selection) -- drives the picker option lists and arrow-key clamping.
  // yearBuckets/monthBuckets above stay chart data; this is navigation only.
  const bounds = $derived(boundsResource.data ?? {});
  let selectedStage = $state("Core");
  let hoveredStage = $state<string | null>(null);

  const loadedMainSleep = $derived(
    sessions.filter((session) => session.is_main_sleep),
  );
  const monthlyBuckets = $derived(monthBuckets.slice().reverse());
  const yearlyBuckets = $derived(yearBuckets.slice().reverse());
  const periodYears = $derived(
    bounds.min
      ? yearOptionsInRange(bounds)
      : selectedYear
        ? [selectedYear]
        : [],
  );
  // Full "YYYY-MM" period values (this page's own month convention), not
  // month.ts's bare 1-12 -- built directly from bounds rather than reusing
  // monthOptionsInRange, which returns the other convention. Newest first,
  // matching yearOptionsInRange's and monthOptionsInRange's own convention.
  const periodMonths = $derived.by(() => {
    if (!selectedYear) return [];
    if (!bounds.min || !bounds.max)
      return selectedMonth
        ? [
            {
              value: selectedMonth,
              label: formatPeriod(selectedMonth, "month"),
            },
          ]
        : [];
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
  // These reads cover ALL history, so a successful inventory can project a
  // new selected period immediately. Missing buckets are zero only on success.
  const sleepSummary = $derived.by<SleepAggregateBucket | null>(() => {
    if (aggregatesResource.data == null) return null;
    const bucket = selectedMonth
      ? monthBuckets.find((bucket) => bucket.period === selectedMonth)
      : selectedYear
        ? yearBuckets.find((bucket) => bucket.period === selectedYear)
        : lifetimeBucket;
    return bucket ?? emptyBucket(selectedMonth || selectedYear || "lifetime");
  });
  const availableSummary = $derived.by(() => {
    if (sleepSummary) return null;
    if (selectedYear && yearResource.data) {
      const year =
        yearResource.data.find((bucket) => bucket.period === selectedYear) ??
        emptyBucket(selectedYear);
      return { scope: selectedYear, summary: year };
    }
    if (lifetimeResource.data)
      return {
        scope: "Lifetime",
        summary: lifetimeBucket ?? emptyBucket("lifetime"),
      };
    return null;
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
    sleepSummary && sleepSummary.main_sleep_count > 0
      ? sleepSummary.average_asleep_s
      : null,
  );
  const averageEfficiency = $derived(
    sleepSummary && sleepSummary.main_sleep_count > 0
      ? sleepSummary.average_efficiency
      : null,
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
    const scope = selectedScope();
    if (append) {
      const previous = sessionsResource.data;
      if (!hasMore || !previous?.value.next_cursor || loadingMore) return;
      pendingPage = previous;
      const next = await pageResource.run(() =>
        listSleep({
          limit: PAGE_SIZE,
          cursor: previous.value.next_cursor!,
          ...scope,
        }),
      );
      if (
        !next ||
        sessionsResource.data !== previous ||
        selectedScope().date !== scope.date ||
        sessionsResource.loading
      )
        return;
      sessionsResource.mutate(() => ({
        date: scope.date,
        value: { ...next, items: [...previous.value.items, ...next.items] },
      }));
      pendingPage = sessionsResource.data;
      return;
    }
    pageResource.invalidate();
    pendingPage = null;
    const result = await sessionsResource.run(async () => ({
      date: scope.date,
      value: await listSleep({ limit: PAGE_SIZE, ...scope }),
    }));
    if (result) selected = result.value.items[0] ?? null;
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
    if (!loadMoreSentinel || !hasMore || loadMoreError) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!loadMoreError && entries.some((entry) => entry.isIntersecting))
          void loadSessions(true);
      },
      { root: nightListContainer ?? null, rootMargin: "120px" },
    );
    observer.observe(loadMoreSentinel);
    return () => observer.disconnect();
  });

  async function loadAggregates(granularity: "year" | "month" | "lifetime") {
    const resource =
      granularity === "year"
        ? yearResource
        : granularity === "month"
          ? monthResource
          : lifetimeResource;
    await resource.run(
      async () => (await listSleepAggregates(granularity)).buckets,
    );
  }

  async function loadBounds() {
    const result = await boundsResource.run(getSleepBounds);
    if (!result) return;
    const before = selectedScope().date;
    const validYears = new Set(yearOptionsInRange(result));
    if (validYears.size && selectedYear && !validYears.has(selectedYear)) {
      selectedYear = "";
      selectedMonth = "";
    } else if (
      result.min &&
      selectedMonth &&
      !periodMonths.some((option) => option.value === selectedMonth)
    )
      selectedMonth = "";
    syncPeriodUrl();
    if (before !== selectedScope().date) void loadSessions(false);
  }

  onMount(() => {
    void loadBounds();
    for (const granularity of ["year", "month", "lifetime"] as const)
      void loadAggregates(granularity);
    void loadSessions(false);
  });

  return {
    sessionsResource,
    yearResource,
    monthResource,
    lifetimeResource,
    boundsResource,
    get aggregatesResource() {
      return aggregatesResource;
    },
    get availableSummary() {
      return availableSummary;
    },
    get recordsScope() {
      return sessionsResource.data
        ? sessionsResource.data.date || "Lifetime"
        : "";
    },
    loadAggregates,
    loadBounds,
    get sessions() {
      return sessions;
    },
    get selected() {
      return selected;
    },
    get loadingMore() {
      return loadingMore;
    },
    get loadMoreError() {
      return loadMoreError;
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

function emptyBucket(period: string): SleepAggregateBucket {
  return {
    period,
    session_count: 0,
    main_sleep_count: 0,
    nap_count: 0,
    observed_wake_dates: 0,
    average_asleep_s: 0,
    average_time_in_bed_s: 0,
    average_efficiency: 0,
    core_s: 0,
    deep_s: 0,
    rem_s: 0,
    awake_s: 0,
    unspecified_s: 0,
  };
}

function formatPeriod(period: string, granularity: "month" | "year"): string {
  if (granularity === "year") return period.slice(0, 4);
  return formatMonth(period.slice(0, 7));
}
