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
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";
  import StatTile from "@iroha/shared/components/StatTile.svelte";
  import {
    formatDate,
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
    formatHr,
    formatElevation,
  } from "$lib/format";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
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
  import { sportColor, sportLabel } from "$lib/sport";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
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
  let loadingMore = $state(false);
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
  const theme = useTheme();
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

  async function loadActivitySeries() {
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
        getMetricSeries("movement.distance_m", params),
        getMetricSeries("movement.duration_s", params),
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

  // Fetch one page. `append` distinguishes a fresh query (replace) from
  // "Load more" (accumulate). Cursor + has_more drive the keyset walk.
  async function load(append: boolean) {
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
      loadingMore = true;
      try {
        const page = await listActivities({ ...applied, cursor });
        if (activitiesResource.loading || activitiesResource.data !== previous)
          return;
        if (!matchesScope(previous, applied)) return;
        activitiesResource.mutate(() => ({
          scope: previous.scope,
          value: { ...page, items: [...previous.value.items, ...page.items] },
        }));
      } catch {
        // Load-more failures are retry-safe -- keep the rows already
        // showing rather than replacing a working view with an error.
      } finally {
        loadingMore = false;
      }
      return;
    }
    const params = { ...applied };
    const scope = params;
    await activitiesResource.run(async () => ({
      scope,
      value: await listActivities(params),
    }));
  }

  function clear() {
    sportType = "";
    selectedYear = "";
    selectedMonth = "";
    applied = buildParams();
    syncUrl();
    load(false);
    void loadSummary();
  }

  async function loadSummary() {
    const params = buildParams();
    const scope = params;
    await summaryResource.run(async () => ({
      scope,
      value: await getActivitySummary({
        date: params.date,
        sport: params.sport_type || null,
        timezone: IROHA_TIMEZONE,
      }),
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

  const trackedSports = $derived(summary?.by_sport.length ?? 0);

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
        if (entries[0].isIntersecting && canLoadMore && !loadingMore) {
          void load(true);
        }
      },
      { rootMargin: "600px" },
    );
    io.observe(el);
    return () => io.disconnect();
  });

  function isNonDistanceSport(sport?: string, distanceM?: number): boolean {
    if (!sport) return true;
    if (distanceM == null || distanceM <= 0) return true;
    const s = sport.toLowerCase();
    return !["run", "walk", "hike", "ride", "cycl", "swim"].some((k) =>
      s.includes(k),
    );
  }

  function isCycling(sport?: string): boolean {
    if (!sport) return false;
    const s = sport.toLowerCase();
    return s.includes("ride") || s.includes("cycl") || s.includes("bik");
  }

  function isSwimming(sport?: string): boolean {
    if (!sport) return false;
    const s = sport.toLowerCase();
    return s.includes("swim");
  }

  function formatCyclingSpeed(distanceM?: number, durationS?: number): string {
    if (!distanceM || !durationS) return "—";
    const km = distanceM / 1000;
    const hours = durationS / 3600;
    const kmh = km / hours;
    return `${kmh.toFixed(1)} km/h`;
  }

  function formatSwimmingPace(distanceM?: number, durationS?: number): string {
    if (!distanceM || !durationS || distanceM <= 0) return "—";
    const secPer100m = (durationS / distanceM) * 100;
    const m = Math.floor(secPer100m / 60);
    const s = Math.round(secPer100m % 60);
    return `${m}:${String(s).padStart(2, "0")} /100m`;
  }
  function displayTitle(title?: string, sport?: string): string {
    if (!title) return sportLabel(sport);
    if (sportLabel(title) === sportLabel(sport)) return sportLabel(sport);
    return title;
  }
</script>

<svelte:head>
  <title>Motion · iroha</title>
</svelte:head>

<section class="activities-shell">
  {#if hasThemeRoute(theme.definition(), "activities")}
    <LoadingBoundary
      resource={activitiesResource}
      preserveLayout
      label="Loading activities…"
    >
      <ThemeRouteRenderer
        route="activities"
        props={{
          activities,
          displaySummary,
          summaryLoading: summaryResource.loading,
          summaryError: summaryResource.error,
          summaryDistanceUnknownCount:
            summaryTotals?.distance_unknown_count ?? 0,
          boundsError: boundsResource.error,
          boundsLoading: boundsResource.loading,
          onRetryBounds: () => loadBounds(),
          summaryScope: summaryResource.data
            ? scopeLabel(summaryResource.data.scope)
            : "",
          recordsScope: activitiesResource.data
            ? scopeLabel(activitiesResource.data.scope)
            : "",
          recordsReady: activitiesResource.ready,
          onRetrySummary: () => loadSummary(),
          onRetryRecords: () => load(false),
          onRetrySeries: () => {
            if (
              !selectedYear &&
              !matchesScope(summaryResource.data, buildParams())
            )
              return loadSummary();
            return loadActivitySeries();
          },
          sportType,
          sportOptions,
          loading: activitiesResource.loading,
          error: activitiesResource.error,
          hasMore: canLoadMore,
          loadingMore,
          activitySeries,
          activityDurationSeries,
          activitySeriesLoading:
            seriesResource.loading ||
            (!selectedYear && summaryResource.loading),
          activitySeriesError:
            seriesResource.error ??
            (!selectedYear && summaryResource.error
              ? "Could not determine the movement series scope."
              : null),
          activitySeriesScope: seriesResource.data
            ? scopeLabel(seriesResource.data.scope)
            : "",
          onSportType: (value: string) => {
            sportType = value;
            syncUrl();
            void loadSummary();
          },
          onLoadMore: () => void load(true),
          onOpenDetail: (id: string) => void goto(`/motion/${id}`),
        }}
      >
        {#snippet children()}
          <PeriodToolbar title="Motion archive scope" ariaLabel="Motion period">
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
      </ThemeRouteRenderer>
    </LoadingBoundary>
  {:else}
    <RouteIntro
      eyebrow="Motion / activity archive"
      title="Every session, in one place."
      description="Find a movement session quickly, narrow the archive, and open the record when you want its route and measurements."
      actionHref="/"
      actionLabel="Back to Today"
    />

    <PeriodToolbar title="Motion archive scope" ariaLabel="Motion period">
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

    <div class="stat-strip" aria-label="Activity summary">
      <StatTile
        label="Activities"
        value={displaySummary
          ? displaySummary.activity_count.toLocaleString()
          : "—"}
        sub={sportType || selectedYear ? "Filtered count" : "Imported sessions"}
      />
      <StatTile
        label="Distance"
        value={displaySummary ? formatDistance(displaySummary.distance_m) : "—"}
        sub={sportType || selectedYear
          ? "Filtered distance"
          : "Across all activities"}
      />
      <StatTile
        label="Total time"
        value={displaySummary ? formatDuration(displaySummary.duration_s) : "—"}
        sub={sportType || selectedYear
          ? "Filtered duration"
          : "Recorded duration"}
      />
      <StatTile
        label="Sports"
        value={summary ? trackedSports.toLocaleString() : "—"}
        sub="Activity types tracked"
      />
    </div>

    <form class="activity-toolbar tile" onsubmit={(e) => e.preventDefault()}>
      <div class="filter-fields">
        <label
          >Sport
          <select
            value={sportType}
            onchange={(event) => {
              sportType = (event.currentTarget as HTMLSelectElement).value;
              syncUrl();
              void loadSummary();
            }}
          >
            <option value="">All sports</option>
            {#each sportOptions as option (option)}
              <option value={option}>{sportLabel(option)}</option>
            {/each}
          </select>
        </label>
      </div>
      <div class="toolbar-actions">
        <button type="button" class="secondary" onclick={clear}
          >Clear filters</button
        >
      </div>
    </form>

    {#if activitiesResource.loading && activities.length === 0}
      <p class="muted">Loading activities…</p>
    {:else if activitiesResource.error}
      <p class="error">
        Failed to load activities: {activitiesResource.error}
      </p>
    {:else if activities.length === 0}
      <p class="muted">No activities found.</p>
    {:else}
      <p class="muted result-count">
        {activities.length} shown{hasMore ? " (more available)" : ""}
      </p>
      <ul class="activity-grid">
        {#each activities as activity (activity.id)}
          <li>
            <a
              class="activity-card tile tile-interactive"
              href={`/motion/${activity.id}`}
              style={`--sport-color: ${sportColor(activity.sport_type)}`}
            >
              <span class="accent" aria-hidden="true"></span>
              <div class="card-top">
                <SportBadge sport={activity.sport_type} /><span
                  class="activity-date"
                  >{formatDateOnly(
                    activity.started_at,
                    activity.timezone,
                  )}</span
                >
              </div>
              <h2>{displayTitle(activity.title, activity.sport_type)}</h2>
              {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                <div class="primary-metric">
                  {formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                </div>
                <div class="card-metrics">
                  <span>Avg HR: {formatHr(activity.avg_hr)}</span>
                  <span>Max HR: {formatHr(activity.max_hr)}</span>
                </div>
              {:else}
                <div class="primary-metric">
                  {formatDistance(activity.distance_m)}
                </div>
                <div class="card-metrics">
                  {#if isCycling(activity.sport_type)}
                    <span
                      >{formatCyclingSpeed(
                        activity.distance_m,
                        activity.duration_s ?? activity.moving_time_s,
                      )}</span
                    >
                  {:else if isSwimming(activity.sport_type)}
                    <span
                      >{formatSwimmingPace(
                        activity.distance_m,
                        activity.duration_s ?? activity.moving_time_s,
                      )}</span
                    >
                  {:else}
                    <span>{formatPace(activity.avg_pace_s_per_km)}</span>
                  {/if}
                  <span
                    >{formatDuration(
                      activity.duration_s ?? activity.moving_time_s,
                    )}</span
                  >
                  {#if activity.avg_hr}<span
                      >Avg HR: {formatHr(activity.avg_hr)}</span
                    >{/if}
                </div>
              {/if}
            </a>
          </li>
        {/each}
      </ul>
      {#if hasMore}
        <div class="load-sentinel">
          <button
            class="load-more"
            onclick={() => load(true)}
            disabled={loadingMore}
            >{loadingMore ? "Loading…" : "Load more activities"}</button
          >
        </div>
      {/if}
    {/if}
  {/if}
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
  .stat-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }
  .activity-toolbar {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem;
  }
  .filter-fields {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    gap: 0.75rem;
  }
  .filter-fields label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    font-weight: 650;
  }
  .toolbar-actions {
    display: flex;
    gap: 0.5rem;
  }
  .toolbar-actions button {
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--bg);
    padding: 0.5rem 0.75rem;
    font: inherit;
    font-size: 0.84rem;
    cursor: pointer;
  }
  .toolbar-actions .secondary {
    border-color: var(--border);
    background: var(--surface-2);
    color: var(--text);
  }
  .activity-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .activity-card {
    position: relative;
    display: grid;
    gap: 0.75rem;
    min-height: 13rem;
    padding: 1rem;
    overflow: hidden;
    color: var(--text);
    text-decoration: none;
    container-type: inline-size;
    background:
      linear-gradient(
        157deg,
        color-mix(in srgb, var(--sport-color) 13%, transparent) 0%,
        transparent 55%
      ),
      var(--tile-surface);
  }
  .activity-card:hover {
    text-decoration: none;
  }
  .accent {
    position: absolute;
    inset: 0 auto 0 0;
    width: 0.25rem;
    background: var(--sport-color);
  }
  .card-top {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .activity-date {
    color: var(--text-muted);
    font-size: 0.72rem;
    text-align: right;
  }
  .activity-card h2 {
    margin: 0;
    font-size: 1rem;
    line-height: 1.25;
  }
  .primary-metric {
    align-self: end;
    color: var(--text);
    font-size: clamp(1.3rem, 13cqi, 2rem);
    font-weight: 750;
    line-height: 1;
    white-space: nowrap;
  }
  .card-metrics {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  @media (max-width: 1024px) {
    .stat-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .activity-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .activity-toolbar {
      align-items: stretch;
      flex-direction: column;
    }
  }
  @media (max-width: 640px) {
    .activity-grid {
      grid-template-columns: 1fr;
    }
    .toolbar-actions {
      width: 100%;
    }
    .toolbar-actions button {
      flex: 1;
    }
    .activity-date {
      max-width: 9rem;
    }
  }
</style>
