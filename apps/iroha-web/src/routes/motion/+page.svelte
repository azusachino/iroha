<script lang="ts">
  import { goto, replaceState } from "$app/navigation";
  import { onMount, untrack } from "svelte";
  import { page } from "$app/state";
  import {
    getActivityBounds,
    getActivitySummary,
    getMetricSeries,
    listActivities,
    type Activity,
    type ActivityDisplaySummary,
    type ActivitySummary,
    type ListActivitiesParams,
    type MetricSeriesResponse,
    type Page as ActivityPage,
  } from "$lib/api";
  import type { ActivitySummaryTotals } from "@iroha/shared/domain/activity";
  import PeriodSelector from "@iroha/shared/components/PeriodSelector.svelte";
  import PeriodToolbar from "@iroha/shared/components/PeriodToolbar.svelte";
  import {
    currentYear,
    MONTH_OPTIONS,
    monthBounds,
    monthOptionsInRange,
    yearOptionsInRange,
  } from "@iroha/shared/format/month";
  import {
    currentCalendarScope,
    readCalendarScope,
    scopeFromParts,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/format/scope";
  import { IROHA_TIMEZONE } from "$lib/config";
  import { sportLabel } from "$lib/sport";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import GrapherActivities from "@iroha/shared/theme-ui/grapher/Activities.svelte";
  import { createAsyncResource } from "$lib/asyncResource.svelte";

  // Scoped reads follow the selection. `applied` snapshots the records query
  // so "Load more" cannot append a page from a superseded selection.
  const initialSport = page.url.searchParams.get("sport") ?? "";
  const defaultScope = currentCalendarScope("year", new Date(), IROHA_TIMEZONE);
  const requestedScope = readCalendarScope(page.url.searchParams, {
    fallback: defaultScope,
    allowDay: false,
  });
  const initialYear =
    requestedScope.kind === "year" || requestedScope.kind === "month"
      ? String(requestedScope.year)
      : requestedScope.kind === "lifetime"
        ? ""
        : currentYear(new Date(), IROHA_TIMEZONE);
  const initialMonth =
    requestedScope.kind === "month" ? String(requestedScope.month) : "";
  let sportType = $state(initialSport);
  let selectedYear = $state(initialYear);
  let selectedMonth = $state(initialMonth);
  let applied = $state<ListActivitiesParams>({});

  type ScopedData<T> = { scope: ListActivitiesParams; value: T };
  const activitiesResource =
    createAsyncResource<ScopedData<ActivityPage<Activity>>>();
  const summaryResource = createAsyncResource<ScopedData<ActivitySummary>>();
  const seriesResource = createAsyncResource<
    ScopedData<{
      distance: MetricSeriesResponse | null;
      duration: MetricSeriesResponse | null;
    }>
  >();
  const pageResource = createAsyncResource<ActivityPage<Activity>>();
  let pageFor = $state<ScopedData<ActivityPage<Activity>> | null>(null);
  const loadingMore = $derived(
    pageFor === activitiesResource.data && pageResource.loading,
  );
  const loadMoreError = $derived(
    pageFor === activitiesResource.data ? pageResource.error : null,
  );
  const cursor = $derived(activitiesResource.data?.value.next_cursor ?? null);
  const hasMore = $derived(activitiesResource.data?.value.has_more ?? false);
  const canLoadMore = $derived(
    hasMore &&
      !activitiesResource.loading &&
      matchesScope(activitiesResource.data, applied),
  );
  const activities = $derived(activitiesResource.data?.value.items ?? []);
  const summary = $derived(summaryResource.data?.value ?? null);
  const activitySeries = $derived(seriesResource.data?.value.distance ?? null);
  const activityDurationSeries = $derived(
    seriesResource.data?.value.duration ?? null,
  );
  const sportOptions = $derived(
    [
      ...new Set(
        [
          sportType,
          ...(summary?.by_sport.map((bucket) => bucket.key) ?? []),
        ].filter(Boolean),
      ),
    ].sort(),
  );

  // The real data range (fetched once, independent of the current
  // selection) -- not "today". A scoped summary request failing (e.g. a
  // stale/tampered URL naming an out-of-range period) must never collapse
  // these option lists, so they never read from `summary`.
  const boundsResource = createAsyncResource<DateBounds>();
  const bounds = $derived(boundsResource.data ?? {});
  const years = $derived(
    boundsResource.ready
      ? yearOptionsInRange(bounds)
      : selectedYear
        ? [selectedYear]
        : [],
  );
  const months = $derived(
    boundsResource.ready
      ? monthOptionsInRange(selectedYear, bounds)
      : MONTH_OPTIONS.filter((option) => option.value === selectedMonth),
  );

  async function loadBounds() {
    const loaded = await boundsResource.run(() => getActivityBounds());
    if (!loaded) return;
    const previous = buildParams();
    const validYears = new Set(yearOptionsInRange(loaded));
    if (selectedYear && !validYears.has(selectedYear)) {
      selectedYear = "";
      selectedMonth = "";
    } else if (selectedMonth) {
      const validMonths = new Set(
        monthOptionsInRange(selectedYear, loaded).map((option) => option.value),
      );
      if (!validMonths.has(selectedMonth)) selectedMonth = "";
    }
    syncUrl();
    if (previous.date !== buildParams().date) void loadSummary();
  }

  function handleYearChange() {
    if (!selectedYear) {
      selectedMonth = "";
    }
    syncUrl();
    void loadSummary();
  }

  function handleMonthChange() {
    syncUrl();
    void loadSummary();
  }

  function metricSport(value: string): string {
    const normalized = value.toLowerCase();
    if (normalized.includes("hik")) return "hike";
    if (
      normalized.includes("ride") ||
      normalized.includes("cycl") ||
      normalized.includes("bik")
    )
      return "ride";
    if (normalized.includes("run")) return "run";
    if (normalized.includes("swim")) return "swim";
    if (normalized.includes("walk")) return "walk";
    return "other";
  }

  function chartWindow(): {
    from: string;
    to: string;
    grain: "day" | "month";
  } | null {
    if (selectedYear && selectedMonth) {
      const month = `${selectedYear}-${selectedMonth.padStart(2, "0")}`;
      const bounds = monthBounds(month);
      return { ...bounds, grain: "day" };
    }
    if (selectedYear) {
      return {
        from: `${selectedYear}-01-01`,
        to: `${Number(selectedYear) + 1}-01-01`,
        grain: "month",
      };
    }
    if (!matchesScope(summaryResource.data, buildParams())) return null;
    const months = (summary?.by_month ?? []).map((bucket) => bucket.key).sort();
    if (months.length === 0) return null;
    return {
      from: `${months[0]}-01`,
      to: monthBounds(months[months.length - 1]).to,
      grain: "month",
    };
  }

  async function loadActivitySeries(forceRefresh = false) {
    const window = chartWindow();
    const scope = buildParams();
    if (!window) {
      if (matchesScope(summaryResource.data, scope)) {
        await seriesResource.run(async () => ({
          scope,
          value: { distance: null, duration: null },
        }));
      } else {
        seriesResource.invalidate();
      }
      return;
    }
    const params = {
      ...window,
      timezone: IROHA_TIMEZONE,
      dimensions: sportType ? [`sport:${metricSport(sportType)}`] : [],
    };
    await seriesResource.run(async () => {
      const [distance, duration] = await Promise.all([
        getMetricSeries("movement.distance_m", params, { forceRefresh }),
        getMetricSeries("movement.duration_s", params, { forceRefresh }),
      ]);
      return { scope, value: { distance, duration } };
    });
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    if (sportType) url.searchParams.set("sport", sportType);
    else url.searchParams.delete("sport");
    writeCalendarScope(
      url.searchParams,
      scopeFromParts(selectedYear, selectedMonth),
    );
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  // Smaller first page than the server's 50 default — lighter initial paint,
  // and infinite scroll pages the rest in as you go.
  const PAGE_SIZE = 24;

  function buildParams(): ListActivitiesParams {
    const params: ListActivitiesParams = { limit: PAGE_SIZE };
    if (sportType) params.sport_type = sportType;
    const scope = scopeFromParts(selectedYear, selectedMonth);
    const date = serializeCalendarScope(scope);
    if (date) params.date = date;
    return params;
  }

  function matchesScope(
    snapshot: { scope: ListActivitiesParams } | null,
    requested: ListActivitiesParams,
  ): boolean {
    return (
      !!snapshot &&
      snapshot.scope.date === requested.date &&
      snapshot.scope.sport_type === requested.sport_type
    );
  }

  function scopeLabel(params: ListActivitiesParams): string {
    const period = params.date || "Lifetime";
    return params.sport_type
      ? `${period} · ${sportLabel(params.sport_type)}`
      : period;
  }

  async function hardRefresh() {
    await Promise.all([
      load(false, true),
      loadSummary(true),
      loadActivitySeries(true),
    ]);
  }

  // Fetch one page. `append` distinguishes a fresh query (replace) from
  // "Load more" (accumulate). Cursor + has_more drive the keyset walk.
  async function load(append: boolean, forceRefresh = false) {
    if (append) {
      const previous = activitiesResource.data;
      if (
        !previous ||
        !hasMore ||
        !cursor ||
        loadingMore ||
        activitiesResource.loading
      )
        return;
      if (!matchesScope(previous, applied)) return;
      pageFor = previous;
      const page = await pageResource.run(() =>
        listActivities({
          ...previous.scope,
          cursor: previous.value.next_cursor!,
        }),
      );
      if (
        !page ||
        activitiesResource.loading ||
        activitiesResource.data !== previous ||
        !matchesScope(previous, applied)
      )
        return;
      activitiesResource.mutate(() => ({
        scope: previous.scope,
        value: { ...page, items: [...previous.value.items, ...page.items] },
      }));
      pageFor = activitiesResource.data;
      return;
    }
    pageResource.invalidate();
    pageFor = null;
    const params = { ...applied };
    const scope = params;
    await activitiesResource.run(async () => ({
      scope,
      value: await listActivities(params, { forceRefresh }),
    }));
  }

  async function loadSummary(forceRefresh = false) {
    const params = buildParams();
    const scope = params;
    await summaryResource.run(async () => ({
      scope,
      value: await getActivitySummary(
        {
          date: params.date,
          sport: params.sport_type || null,
          timezone: IROHA_TIMEZONE,
        },
        { forceRefresh },
      ),
    }));
  }

  // The HTTP summary is year-wide even for a month date. Derive the bucket
  // from the OBSERVED request, never the current (possibly pending) selection.
  const summaryTotals = $derived.by<ActivitySummaryTotals | null>(() => {
    if (!summary) return null;
    const date = summaryResource.data?.scope.date;
    if (date && /^\d{4}-\d{2}$/.test(date)) {
      return (
        summary.by_month.find((bucket) => bucket.key === date) ?? {
          activity_count: 0,
          distance_m: 0,
          duration_s: 0,
          distance_known_count: 0,
          distance_unknown_count: 0,
          elevation_gain_m: 0,
        }
      );
    }
    return summary.totals;
  });

  const displaySummary = $derived.by<ActivityDisplaySummary | null>(() => {
    const totals = summaryTotals;
    if (!totals) return null;
    return {
      activity_count: totals.activity_count,
      distance_m:
        totals.distance_unknown_count > 0 && totals.distance_known_count === 0
          ? null
          : totals.distance_m,
      duration_s: totals.moving_time_s || totals.duration_s,
    };
  });

  // Re-read URL searchParams on browser history popstate (e.g. Back button)
  onMount(() => {
    const handlePopState = () => {
      const requested = readCalendarScope(page.url.searchParams, {
        fallback: defaultScope,
        allowDay: false,
      });
      const urlSport = page.url.searchParams.get("sport") ?? "";
      const urlYear =
        requested.kind === "year" || requested.kind === "month"
          ? String(requested.year)
          : requested.kind === "lifetime"
            ? ""
            : currentYear(new Date(), IROHA_TIMEZONE);
      const urlMonth =
        requested.kind === "month"
          ? (serializeCalendarScope(requested) as string)
          : "";

      if (
        urlSport !== sportType ||
        urlYear !== selectedYear ||
        urlMonth !== selectedMonth
      ) {
        sportType = urlSport;
        selectedYear = urlYear;
        selectedMonth = urlMonth;
      }
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  });

  // Reactive filtering effect: automatically re-runs query when filter states change.
  $effect(() => {
    const _s = sportType;
    const _y = selectedYear;
    const _m = selectedMonth;

    untrack(() => {
      applied = buildParams();
      void load(false);
    });
  });

  $effect(() => {
    const _s = sportType;
    const _y = selectedYear;
    const _m = selectedMonth;
    const _summary = selectedYear ? null : summaryResource.data;

    untrack(() => {
      void loadActivitySeries();
    });
  });

  onMount(() => {
    void loadSummary();
    void loadBounds();
  });

  // Infinite scroll: when the sentinel below the grid scrolls near the
  // viewport, page the next keyset window automatically. The "Load more"
  // button stays as a manual fallback.
  let sentinel = $state<HTMLElement | null>(null);
  $effect(() => {
    const el = sentinel;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (
          entries[0].isIntersecting &&
          canLoadMore &&
          !loadingMore &&
          !loadMoreError
        ) {
          void load(true);
        }
      },
      { rootMargin: "600px" },
    );
    io.observe(el);
    return () => io.disconnect();
  });
</script>

<svelte:head>
  <title>iroha · Motion</title>
</svelte:head>

<section class="activities-shell">
  <LoadingBoundary
    resource={activitiesResource}
    preserveLayout
    label="Loading activities…"
  >
    <GrapherActivities
      theme="grapher"
      {activities}
      {displaySummary}
      summaryLoading={summaryResource.loading}
      summaryError={summaryResource.error}
      summaryDistanceUnknownCount={summaryTotals?.distance_unknown_count ?? 0}
      boundsError={boundsResource.error}
      boundsLoading={boundsResource.loading}
      onRetryBounds={() => loadBounds()}
      summaryScope={summaryResource.data
        ? scopeLabel(summaryResource.data.scope)
        : ""}
      recordsScope={activitiesResource.data
        ? scopeLabel(activitiesResource.data.scope)
        : ""}
      recordsReady={activitiesResource.ready}
      onRetrySummary={() => loadSummary()}
      onRetryRecords={() => load(false)}
      onRetrySeries={() => {
        if (!selectedYear && !matchesScope(summaryResource.data, buildParams()))
          return loadSummary();
        return loadActivitySeries();
      }}
      {sportType}
      {sportOptions}
      loading={activitiesResource.loading}
      error={activitiesResource.error}
      hasMore={canLoadMore}
      {loadingMore}
      {loadMoreError}
      {activitySeries}
      {activityDurationSeries}
      activitySeriesLoading={seriesResource.loading ||
        (!selectedYear && summaryResource.loading)}
      activitySeriesError={seriesResource.error ??
        (!selectedYear && summaryResource.error
          ? "Could not determine the movement series scope."
          : null)}
      activitySeriesScope={seriesResource.data
        ? scopeLabel(seriesResource.data.scope)
        : ""}
      onSportType={(value: string) => {
        sportType = value;
        syncUrl();
        void loadSummary();
      }}
      onLoadMore={() => load(true)}
      onOpenDetail={(id: string) => void goto(`/motion/${id}`)}
    >
      {#snippet children()}
        <PeriodToolbar
          title="Motion archive scope"
          ariaLabel="Motion period"
          onRefresh={() => void hardRefresh()}
          refreshing={activitiesResource.loading || summaryResource.loading}
        >
          <PeriodSelector
            year={selectedYear}
            month={selectedMonth}
            {years}
            {months}
            {bounds}
            monthDisabled={!selectedYear}
            surface="inline"
            onYear={(value) => {
              selectedYear = value;
              handleYearChange();
            }}
            onMonth={(value) => {
              selectedMonth = value;
              handleMonthChange();
            }}
          />
        </PeriodToolbar>
      {/snippet}
    </GrapherActivities>
  </LoadingBoundary>
  {#if canLoadMore}
    <div
      bind:this={sentinel}
      class="motion-load-sentinel"
      data-testid="motion-load-sentinel"
      aria-hidden="true"
    ></div>
  {/if}
</section>

<style>
  .activities-shell {
    display: grid;
    gap: 1.25rem;
  }
</style>
