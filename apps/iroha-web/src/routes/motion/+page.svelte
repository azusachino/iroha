<script lang="ts">
  import { replaceState } from "$app/navigation";
  import { onMount, untrack } from "svelte";
  import { page } from "$app/state";
  import {
    getActivitySummary,
    getMetricSeries,
    listActivities,
    type Activity,
    type ListActivitiesParams,
    type MetricSeriesResponse,
    type Summary,
  } from "$lib/api";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
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
  import { currentYear, MONTH_OPTIONS, monthBounds } from "@iroha/shared/month";
  import { sportColor, sportLabel } from "$lib/sport";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  // Draft filter inputs (bound to the form); committed to `applied` on submit
  // so "Load more" keeps paging the same query the user actually ran.
  const initialSport = page.url.searchParams.get("sport") ?? "";
  const initialYearParam = page.url.searchParams.get("year") ?? "";
  const initialYear = /^\d{4}$/.test(initialYearParam)
    ? initialYearParam
    : currentYear();
  const initialMonthParam = page.url.searchParams.get("month") ?? "";
  const initialMonth = /^(?:[1-9]|1[0-2])$/.test(initialMonthParam)
    ? initialMonthParam
    : "";
  let sportType = $state(initialSport);
  let selectedYear = $state(initialYear);
  let selectedMonth = $state(initialMonth);
  let applied = $state<ListActivitiesParams>({});

  let activities = $state<Activity[]>([]);
  let loading = $state(true);
  let loadingMore = $state(false);
  let cursor = $state<string | null>(null);
  let hasMore = $state(false);
  let error = $state<string | null>(null);
  let summary = $state<Summary | null>(null);
  let summaryLoading = $state(true);
  let activitySeries = $state<MetricSeriesResponse | null>(null);
  let activityDurationSeries = $state<MetricSeriesResponse | null>(null);
  let activitySeriesLoading = $state(true);
  let activitySeriesError = $state<string | null>(null);
  let activitySeriesRequest = 0;
  const theme = useTheme();
  const sportOptions = $derived(
    summary ? summary.by_sport.map((b) => b.key).sort() : [],
  );

  const months = MONTH_OPTIONS;

  const years = $derived(
    summary
      ? summary.by_year.map((b) => b.key).sort((a, b) => b.localeCompare(a))
      : [new Date().getFullYear().toString()],
  );

  function handleYearChange() {
    if (!selectedYear) {
      selectedMonth = "";
    }
    syncUrl();
  }

  function handleMonthChange() {
    syncUrl();
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
    if (!window) {
      activitySeries = null;
      activityDurationSeries = null;
      activitySeriesLoading = false;
      activitySeriesError = null;
      return;
    }
    const requestId = ++activitySeriesRequest;
    activitySeriesLoading = true;
    activitySeriesError = null;
    try {
      const params = {
        ...window,
        dimensions: sportType ? [`sport:${metricSport(sportType)}`] : [],
      };
      const [distance, duration] = await Promise.all([
        getMetricSeries("movement.distance_m", params),
        getMetricSeries("movement.duration_s", params),
      ]);
      if (requestId === activitySeriesRequest) {
        activitySeries = distance;
        activityDurationSeries = duration;
      }
    } catch (cause) {
      if (requestId !== activitySeriesRequest) return;
      activitySeries = null;
      activityDurationSeries = null;
      activitySeriesError =
        cause instanceof Error ? cause.message : String(cause);
    } finally {
      if (requestId === activitySeriesRequest) activitySeriesLoading = false;
    }
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    if (sportType) url.searchParams.set("sport", sportType);
    else url.searchParams.delete("sport");
    if (selectedYear) url.searchParams.set("year", selectedYear);
    else url.searchParams.delete("year");
    if (selectedMonth) url.searchParams.set("month", selectedMonth);
    else url.searchParams.delete("month");
    if (url.search !== window.location.search) replaceState(url, page.state);
  }

  // Smaller first page than the server's 50 default — lighter initial paint,
  // and infinite scroll pages the rest in as you go.
  const PAGE_SIZE = 24;

  function buildParams(): ListActivitiesParams {
    const params: ListActivitiesParams = { limit: PAGE_SIZE };
    if (sportType) params.sport_type = sportType;

    if (selectedYear) {
      const y = Number(selectedYear);
      if (selectedMonth) {
        const m = Number(selectedMonth);
        const start = new Date(Date.UTC(y, m - 1, 1, 0, 0, 0));
        const end = new Date(Date.UTC(y, m, 0, 23, 59, 59));
        params.started_from = start.toISOString();
        params.started_to = end.toISOString();
      } else {
        params.started_from = new Date(
          Date.UTC(y, 0, 1, 0, 0, 0),
        ).toISOString();
        params.started_to = new Date(
          Date.UTC(y, 12, 0, 23, 59, 59),
        ).toISOString();
      }
    }
    return params;
  }

  // Fetch one page. `append` distinguishes a fresh query (replace) from
  // "Load more" (accumulate). Cursor + has_more drive the keyset walk.
  async function load(append: boolean) {
    if (append) loadingMore = true;
    else loading = true;
    error = null;
    try {
      const params: ListActivitiesParams = { ...applied };
      if (append && cursor) params.cursor = cursor;
      const page = await listActivities(params);
      activities = append ? [...activities, ...page.items] : page.items;
      cursor = page.next_cursor;
      hasMore = page.has_more;
      // Handled statically via derived summary key mapping
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      if (!append) activities = [];
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  function clear() {
    sportType = "";
    selectedYear = "";
    selectedMonth = "";
    applied = {};
    cursor = null;
    syncUrl();
    load(false);
  }

  async function loadSummary() {
    try {
      summary = await getActivitySummary();
    } finally {
      summaryLoading = false;
    }
  }

  interface DisplaySummary {
    activity_count: number;
    distance_m: number;
    duration_s: number;
  }

  const displaySummary = $derived.by<DisplaySummary>(() => {
    if (!summary) {
      return { activity_count: 0, distance_m: 0, duration_s: 0 };
    }

    if (!sportType && !selectedYear) {
      return {
        activity_count: summary.totals.activity_count,
        distance_m: summary.totals.distance_m,
        duration_s: summary.totals.moving_time_s || summary.totals.duration_s,
      };
    }

    if (sportType && !selectedYear) {
      const bucket = summary.by_sport.find(
        (b) => b.key.toLowerCase() === sportType.toLowerCase(),
      );
      if (bucket) {
        return {
          activity_count: bucket.activity_count,
          distance_m: bucket.distance_m,
          duration_s: bucket.moving_time_s || bucket.duration_s,
        };
      }
    }

    if (selectedYear && !sportType) {
      if (selectedMonth) {
        const monthKey = `${selectedYear}-${selectedMonth.padStart(2, "0")}`;
        const bucket = summary.by_month.find((b) => b.key === monthKey);
        if (bucket) {
          return {
            activity_count: bucket.activity_count,
            distance_m: bucket.distance_m,
            duration_s: bucket.moving_time_s || bucket.duration_s,
          };
        }
      } else {
        const bucket = summary.by_year.find((b) => b.key === selectedYear);
        if (bucket) {
          return {
            activity_count: bucket.activity_count,
            distance_m: bucket.distance_m,
            duration_s: bucket.moving_time_s || bucket.duration_s,
          };
        }
      }
    }

    let count = 0;
    let dist = 0;
    let dur = 0;
    for (const act of activities) {
      count++;
      dist += act.distance_m || 0;
      dur += act.moving_time_s || act.duration_s || 0;
    }
    return {
      activity_count: count,
      distance_m: dist,
      duration_s: dur,
    };
  });

  const trackedSports = $derived(summary?.by_sport.length ?? 0);

  // Reactive filtering effect: automatically re-runs query when filter states change.
  $effect(() => {
    const _s = sportType;
    const _y = selectedYear;
    const _m = selectedMonth;
    const _summary = summary;

    untrack(() => {
      applied = buildParams();
      cursor = null;
      void load(false);
      if (_summary) void loadActivitySeries();
    });
  });

  onMount(() => {
    void loadSummary();
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
        if (entries[0].isIntersecting && hasMore && !loading && !loadingMore) {
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
    <ThemeRouteRenderer
      route="activities"
      props={{
        activities,
        displaySummary,
        sportType,
        sportOptions,
        loading,
        error,
        hasMore,
        loadingMore,
        activitySeries,
        activityDurationSeries,
        activitySeriesLoading,
        activitySeriesError,
        activitySeriesScope: selectedMonth
          ? `${selectedYear}-${selectedMonth.padStart(2, "0")}`
          : selectedYear || "Lifetime",
        onSportType: (value: string) => {
          sportType = value;
          syncUrl();
        },
        onLoadMore: () => void load(true),
      }}
    >
      {#snippet children()}
        <PeriodToolbar title="Motion archive scope" ariaLabel="Motion period">
          <PeriodSelector
            year={selectedYear}
            month={selectedMonth}
            {years}
            {months}
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
        value={summaryLoading
          ? "—"
          : displaySummary.activity_count.toLocaleString()}
        sub={sportType || selectedYear ? "Filtered count" : "Imported sessions"}
      />
      <StatTile
        label="Distance"
        value={summaryLoading ? "—" : formatDistance(displaySummary.distance_m)}
        sub={sportType || selectedYear
          ? "Filtered distance"
          : "Across all activities"}
      />
      <StatTile
        label="Total time"
        value={summaryLoading ? "—" : formatDuration(displaySummary.duration_s)}
        sub={sportType || selectedYear
          ? "Filtered duration"
          : "Recorded duration"}
      />
      <StatTile
        label="Sports"
        value={summaryLoading ? "—" : trackedSports.toLocaleString()}
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

    {#if loading}
      <p class="muted">Loading activities…</p>
    {:else if error}
      <p class="error">Failed to load activities: {error}</p>
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
  {#if hasMore}
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
  @media (max-width: 800px) {
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
  @media (max-width: 560px) {
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
