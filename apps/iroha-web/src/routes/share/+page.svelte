<script lang="ts">
  import { untrack } from "svelte";
  import {
    getPublicSummary,
    listPublicActivities,
    getPublicRoutes,
    type Summary,
    type SummaryBucket,
    type PublicActivity,
    type RouteFeatureCollection,
    type RouteFeature,
  } from "$lib/api";
  import {
    formatDistance,
    formatDuration,
    formatDate,
    formatDateOnly,
    formatSport,
    formatPace,
    formatHr,
  } from "$lib/format";
  import RoutesMap from "$lib/components/RoutesMap.svelte";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
  import YearProgressChart from "$lib/components/YearProgressChart.svelte";
  import { sportColor } from "$lib/sport";

  const MONTH_LABELS = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];

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
    return sport.toLowerCase().includes("swim");
  }

  function formatCyclingSpeed(distanceM?: number, durationS?: number): string {
    if (distanceM == null || durationS == null || durationS <= 0) return "—";
    const km = distanceM / 1000;
    const hours = durationS / 3600;
    return `${(km / hours).toFixed(1)} km/h`;
  }

  function formatSwimmingPace(distanceM?: number, durationS?: number): string {
    if (distanceM == null || durationS == null || durationS <= 0) return "—";
    const pace100m = durationS / (distanceM / 100);
    const m = Math.floor(pace100m / 60);
    const s = Math.round(pace100m % 60);
    return `${m}:${String(s).padStart(2, "0")} /100m`;
  }

  // All-years summary (fetched once, unfiltered) — feeds the year-over-year
  // cumulative curve, which needs both the selected year and the prior year.
  let fullSummary = $state<Summary | null>(null);

  // All-years monthly buckets for the cumulative curve, scoped to the active
  // sport filter so "Run" gives a run-only year-over-year race.
  async function loadFullSummary(sport: string | null) {
    try {
      fullSummary = await getPublicSummary({ sport });
    } catch {
      fullSummary = null;
    }
  }

  // --- Summary (totals / year tabs / month chart / sport breakdown) ---
  let summary = $state<Summary | null>(null);
  let summaryLoading = $state(true);
  let summaryError = $state<string | null>(null);
  let selectedYear = $state<string | null>(null);
  let sportFilter = $state<string | null>(null);

  let initialLoaded = $state(false);
  let loadedYear = $state<string | null>(null);
  let currentFetchPromise = $state<Promise<any> | null>(null);

  async function loadSummary(year: string | null) {
    if (year === loadedYear && summary) return;

    if (!summary) {
      summaryLoading = true;
    }
    summaryError = null;

    const promise = getPublicSummary({ year });
    currentFetchPromise = promise;

    try {
      const res = await promise;
      if (currentFetchPromise !== promise) return; // Discard stale responses

      summary = res;
      loadedYear = year;
      // by_year is sorted desc by the server — the first entry is the latest.
      if (res.by_year.length > 0 && !selectedYear) {
        selectedYear = res.by_year[0].key;
      }
      initialLoaded = true;
    } catch (e) {
      if (currentFetchPromise === promise) {
        summaryError = e instanceof Error ? e.message : String(e);
      }
    } finally {
      if (currentFetchPromise === promise) {
        summaryLoading = false;
      }
    }
  }

  const years = $derived(summary?.by_year.map((y) => y.key) ?? []);

  const selectedYearTotals = $derived.by(() => {
    if (!summary || !selectedYear) return null;
    return summary.by_year.find((y) => y.key === selectedYear) || null;
  });

  // 12 slots (Jan..Dec) for the selected year, filled in from by_month where
  // present. Missing months render as zero-height bars.
  const monthSlots = $derived.by(() => {
    if (!summary || !selectedYear) return [];
    const byKey = new Map(summary.by_month.map((b) => [b.key, b]));
    return MONTH_LABELS.map((label, idx) => {
      const key = `${selectedYear}-${String(idx + 1).padStart(2, "0")}`;
      const bucket = byKey.get(key);
      return { label, key, bucket };
    });
  });

  // Bars are sized by distance; if a whole year has no distance data (e.g. an
  // all-strength-training year), fall back to activity count so bars aren't
  // all flat.
  const monthMetric = $derived.by(() => {
    const totalDistance = monthSlots.reduce(
      (sum, m) => sum + (m.bucket?.distance_m ?? 0),
      0,
    );
    return totalDistance > 0 ? "distance_m" : "activity_count";
  });

  const monthMax = $derived(
    Math.max(
      1,
      ...monthSlots.map((m) => (m.bucket ? m.bucket[monthMetric] : 0)),
    ),
  );

  const sportMax = $derived(
    Math.max(1, ...(summary?.by_sport.map((s) => s.activity_count) ?? [])),
  );

  function selectYear(year: string) {
    selectedYear = year;
  }

  function toggleSport(sport: string) {
    sportFilter = sportFilter === sport ? null : sport;
  }

  // --- Activity table (paginated) ---
  let activities = $state<PublicActivity[]>([]);
  let activitiesLoading = $state(true);
  let activitiesLoadingMore = $state(false);
  let activitiesError = $state<string | null>(null);
  let cursor = $state<string | null>(null);
  let hasMore = $state(false);

  async function loadActivities(append: boolean) {
    if (append) activitiesLoadingMore = true;
    else activitiesLoading = true;
    activitiesError = null;
    try {
      const params: any = {
        limit: 20,
        sport_type: sportFilter ?? undefined,
        cursor: append && cursor ? cursor : undefined,
      };
      if (selectedYear) {
        params.started_from = `${selectedYear}-01-01T00:00:00Z`;
        params.started_to = `${selectedYear}-12-31T23:59:59Z`;
      }
      const page = await listPublicActivities(params);
      activities = append ? [...activities, ...page.items] : page.items;
      cursor = page.next_cursor;
      hasMore = page.has_more;
    } catch (e) {
      activitiesError = e instanceof Error ? e.message : String(e);
      if (!append) activities = [];
    } finally {
      activitiesLoading = false;
      activitiesLoadingMore = false;
    }
  }

  // --- All-routes map ---
  let routes = $state<RouteFeatureCollection | null>(null);
  let routesLoading = $state(true);
  let routesError = $state<string | null>(null);

  async function loadRoutes() {
    routesLoading = true;
    routesError = null;
    try {
      routes = await getPublicRoutes();
    } catch (e) {
      routesError = e instanceof Error ? e.message : String(e);
    } finally {
      routesLoading = false;
    }
  }

  // Routes load once.
  $effect(() => {
    void loadRoutes();
  });

  // The cumulative curve re-fetches when the sport filter changes so it can
  // race run-only (or any-sport) totals year over year.
  $effect(() => {
    const s = sportFilter;
    untrack(() => void loadFullSummary(s));
  });

  // Reactive filtering effect: automatically re-runs query when filter states change.
  $effect(() => {
    const y = selectedYear;
    const s = sportFilter;

    untrack(() => {
      void loadSummary(y);
      activities = [];
      cursor = null;
      void loadActivities(false);
    });
  });
  const selectedYearRoutes = $derived.by<RouteFeature[]>(() => {
    if (!routes || !routes.features || !selectedYear) return [];
    return routes.features.filter(
      (f: any) => f.properties && f.properties.year === selectedYear,
    );
  });

  // Routes for the current year narrowed by the active sport filter, so the
  // map and cities focus on the sport you care about (running by default).
  const filteredRoutes = $derived.by<RouteFeature[]>(() => {
    if (!sportFilter) return selectedYearRoutes;
    return selectedYearRoutes.filter(
      (f) => f.properties.sport_type === sportFilter,
    );
  });

  const cityGroups = $derived.by(() => {
    const groups: Record<
      string,
      { city: string; count: number; runCount: number; sports: Set<string> }
    > = {};

    for (const r of filteredRoutes) {
      const city = r.properties.city || "Unknown";

      if (!groups[city]) {
        groups[city] = { city, count: 0, runCount: 0, sports: new Set() };
      }
      groups[city].count++;
      if (r.properties.sport_type === "run") {
        groups[city].runCount++;
      }
      if (r.properties.sport_type) {
        groups[city].sports.add(r.properties.sport_type);
      }
    }

    // Sort primarily by running count, then by total count
    return Object.values(groups).sort(
      (a, b) => b.runCount - a.runCount || b.count - a.count,
    );
  });

  const maxRunCount = $derived(
    Math.max(1, ...cityGroups.map((g) => g.runCount)),
  );

  // Clicking a city card narrows the map to that city's routes.
  let cityFilter = $state<string | null>(null);

  function toggleCity(city: string) {
    cityFilter = cityFilter === city ? null : city;
  }

  const mappedRoutes = $derived.by<RouteFeature[]>(() => {
    if (!cityFilter) return filteredRoutes;
    return filteredRoutes.filter(
      (f) => (f.properties.city || "Unknown") === cityFilter,
    );
  });

  // A city selection only makes sense within one year/sport view; drop it when
  // the surrounding filters change.
  $effect(() => {
    selectedYear;
    sportFilter;
    untrack(() => {
      cityFilter = null;
    });
  });

  // Infinite scroll for the activity table; the button stays as a fallback.
  let sentinel = $state<HTMLElement | null>(null);
  $effect(() => {
    const el = sentinel;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (
          entries[0].isIntersecting &&
          hasMore &&
          !activitiesLoading &&
          !activitiesLoadingMore
        ) {
          void loadActivities(true);
        }
      },
      { rootMargin: "600px" },
    );
    io.observe(el);
    return () => io.disconnect();
  });
</script>

<h1>Public activity log</h1>

{#if summaryLoading}
  <p class="muted">Loading summary…</p>
{:else if summaryError}
  <p class="error">Failed to load summary: {summaryError}</p>
{:else if summary}
  <!-- Totals hero -->
  <div class="mb-2 text-sm text-text-muted">Totals — {selectedYear}</div>
  <div class="mb-8 grid grid-cols-1 gap-3 sm:grid-cols-3">
    <StatTile
      label="Total distance"
      value={formatDistance(selectedYearTotals?.distance_m ?? 0)}
    />
    <StatTile
      label="Activities"
      value={(selectedYearTotals?.activity_count ?? 0).toLocaleString()}
    />
    <StatTile
      label="Total time"
      value={formatDuration(selectedYearTotals?.duration_s ?? 0)}
    />
  </div>

  <!-- Year tabs -->
  {#if years.length > 0}
    <div class="mb-4 flex flex-wrap gap-2">
      {#each years as year (year)}
        <button
          class="rounded-full border px-4 py-1 text-sm font-medium transition-colors"
          class:border-accent={selectedYear === year}
          class:text-text={selectedYear === year}
          class:bg-surface-2={selectedYear === year}
          class:border-border={selectedYear !== year}
          class:text-text-muted={selectedYear !== year}
          onclick={() => selectYear(year)}
        >
          {year}
        </button>
      {/each}
    </div>

    <!-- Year-over-year cumulative distance -->
    {#if fullSummary && selectedYear}
      <div class="mb-8">
        <YearProgressChart
          byMonth={fullSummary.by_month}
          year={selectedYear}
          sportName={sportFilter ? formatSport(sportFilter) : undefined}
        />
      </div>
    {/if}

    <!-- Per-month bar chart -->
    <div class="mb-8 rounded-lg border border-border bg-surface p-4">
      <div class="mb-3 text-sm text-text-muted">
        Monthly {monthMetric === "distance_m" ? "distance" : "activities"} — {selectedYear}
      </div>
      <div class="flex h-40 items-stretch gap-2">
        {#each monthSlots as slot, idx (slot.key)}
          {@const value = slot.bucket ? slot.bucket[monthMetric] : 0}
          {@const heightPct = Math.max(2, (value / monthMax) * 100)}
          <div
            class="group relative flex h-full flex-1 flex-col items-center justify-end gap-1"
          >
            <div
              class="pointer-events-none absolute -top-6 rounded border border-border bg-surface-2 px-1.5 py-0.5 text-[10px] whitespace-nowrap text-text opacity-0 transition-opacity group-hover:opacity-100"
            >
              {monthMetric === "distance_m"
                ? formatDistance(slot.bucket?.distance_m)
                : `${slot.bucket?.activity_count ?? 0} activities`}
            </div>
            <div
              class="w-full rounded-t transition-all"
              style={`height: ${heightPct}%; background: color-mix(in srgb, var(--sport-run) ${35 + (idx % 4) * 15}%, var(--surface-2))`}
            ></div>
            <div class="text-[10px] text-text-muted">{slot.label}</div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- By-sport breakdown -->
  {#if summary.by_sport.length > 0}
    <div class="mb-8 rounded-lg border border-border bg-surface p-4">
      <div class="mb-3 text-sm text-text-muted">By sport</div>
      <div class="flex flex-col gap-2">
        {#each summary.by_sport as sport (sport.key)}
          <button
            class="flex items-center gap-3 rounded p-1 text-left transition-colors hover:bg-surface-2"
            onclick={() => toggleSport(sport.key)}
          >
            <div class="w-48 shrink-0 flex items-center min-w-0">
              <SportBadge sport={sport.key} />
            </div>
            <div class="h-2 flex-1 overflow-hidden rounded-full bg-surface-2">
              <div
                class="h-full rounded-full"
                style={`width: ${Math.max(2, (sport.activity_count / sportMax) * 100)}%; background: ${sportColor(sport.key)}`}
              ></div>
            </div>
            <span class="w-14 shrink-0 text-right text-xs text-text-muted"
              >{sport.activity_count}×</span
            >
            <span class="w-20 shrink-0 text-right text-xs text-text-muted">
              {#if isNonDistanceSport(sport.key, sport.distance_m)}
                {formatDuration(sport.duration_s)}
              {:else}
                {formatDistance(sport.distance_m)}
              {/if}
            </span>
          </button>
        {/each}
      </div>
      {#if sportFilter}
        <button
          class="mt-2 text-xs text-accent underline"
          onclick={() => toggleSport(sportFilter!)}
        >
          Clear sport filter ({formatSport(sportFilter)})
        </button>
      {/if}
    </div>
  {/if}
{/if}

<!-- All-routes map & Cities heatmap -->
<h2>Routes & Cities</h2>
{#if routesLoading}
  <p class="muted">Loading routes…</p>
{:else if routesError}
  <p class="error">Failed to load routes: {routesError}</p>
{:else if !routes || routes.features.length === 0}
  <p class="muted">No routes found.</p>
{:else}
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-3 mb-8">
    <div class="lg:col-span-2">
      <RoutesMap data={{ type: "FeatureCollection", features: mappedRoutes }} />
    </div>
    <div
      class="rounded-lg border border-border bg-surface p-4 flex flex-col max-h-[400px]"
    >
      <div class="mb-3 flex items-baseline justify-between gap-2">
        <h3 class="text-sm font-semibold text-text-muted">
          {sportFilter
            ? `${formatSport(sportFilter)} cities`
            : "Cities visited"} in {selectedYear}
        </h3>
        {#if cityFilter}
          <button
            class="text-xs text-accent underline"
            onclick={() => (cityFilter = null)}
          >
            Show all
          </button>
        {/if}
      </div>
      {#if cityGroups.length === 0}
        <p class="text-xs text-text-muted">
          No route coordinates available for this year.
        </p>
      {:else}
        <div class="grid grid-cols-2 gap-2 overflow-y-auto pr-1">
          {#each cityGroups as group}
            {@const intensity = group.runCount / maxRunCount}
            {@const heatClass =
              intensity >= 0.8
                ? "bg-accent text-white border-accent shadow-lg shadow-accent/15"
                : intensity >= 0.5
                  ? "bg-accent/40 border-accent/40 text-text"
                  : intensity >= 0.25
                    ? "bg-accent/20 border-accent/25 text-text"
                    : group.runCount > 0
                      ? "bg-accent/10 border-accent/15 text-text"
                      : "bg-surface-2 border-border text-text-muted"}
            {@const selected = cityFilter === group.city}
            <button
              type="button"
              onclick={() => toggleCity(group.city)}
              aria-pressed={selected}
              class={`flex flex-col justify-between rounded-lg border p-3 text-left transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm focus:outline-none ${heatClass} ${selected ? "ring-2 ring-accent ring-offset-1 ring-offset-surface" : ""} ${cityFilter && !selected ? "opacity-50" : ""}`}
            >
              <div class="min-w-0">
                <div class="font-bold text-sm truncate" title={group.city}>
                  {group.city}
                </div>
                <div class="text-[9px] opacity-75 truncate mt-1">
                  {Array.from(group.sports)
                    .map((s) => formatSport(s))
                    .join(", ")}
                </div>
              </div>
              <div class="mt-3 flex items-center justify-between">
                <span
                  class="text-[10px] font-semibold uppercase tracking-wider opacity-90"
                >
                  {group.runCount}
                  {group.runCount === 1 ? "run" : "runs"}
                </span>
                {#if group.count > group.runCount}
                  <span class="text-[9px] opacity-60">
                    (+{group.count - group.runCount} other)
                  </span>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Activity table -->
<h2>Activities</h2>
{#if activitiesLoading}
  <p class="muted">Loading activities…</p>
{:else if activitiesError}
  <p class="error">Failed to load activities: {activitiesError}</p>
{:else if activities.length === 0}
  <p class="muted">No activities found.</p>
{:else}
  <div class="overflow-x-auto rounded-lg border border-border">
    <table class="w-full border-collapse text-sm">
      <thead>
        <tr
          class="border-b border-border bg-surface-2 text-left text-text-muted"
        >
          <th class="px-3 py-2 font-medium">Date</th>
          <th class="px-3 py-2 font-medium">Activity</th>
          <th class="px-3 py-2 font-medium">Distance / Duration</th>
          <th class="px-3 py-2 font-medium">Pace / HR</th>
        </tr>
      </thead>
      <tbody>
        {#each activities as activity (activity.id)}
          <tr
            class="border-b border-border last:border-0 odd:bg-surface even:bg-surface/60"
          >
            <td class="px-3 py-2 whitespace-nowrap text-text"
              >{formatDateOnly(activity.started_at, activity.timezone)}</td
            >
            <td class="px-3 py-2">
              <div class="flex items-center gap-2">
                <SportBadge sport={activity.sport_type} />
                {#if activity.title && formatSport(activity.title) !== formatSport(activity.sport_type)}
                  <span class="font-medium text-text">{activity.title}</span>
                {/if}
              </div>
            </td>
            <td class="px-3 py-2 text-text">
              {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                {formatDuration(activity.duration_s ?? activity.moving_time_s)}
              {:else}
                {formatDistance(activity.distance_m)}
              {/if}
            </td>
            <td class="px-3 py-2 text-text">
              {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                {#if activity.avg_hr}
                  Avg HR: {formatHr(activity.avg_hr)}
                {:else if activity.max_hr}
                  Max HR: {formatHr(activity.max_hr)}
                {:else}
                  —
                {/if}
              {:else}
                {#if isCycling(activity.sport_type)}
                  {formatCyclingSpeed(
                    activity.distance_m,
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                {:else if isSwimming(activity.sport_type)}
                  {formatSwimmingPace(
                    activity.distance_m,
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                {:else}
                  {formatPace(activity.avg_pace_s_per_km)}
                {/if}
                <span class="text-xs text-text-muted"
                  >({formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )})</span
                >
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  {#if hasMore}
    <div bind:this={sentinel}>
      <button
        class="load-more"
        onclick={() => loadActivities(true)}
        disabled={activitiesLoadingMore}
      >
        {activitiesLoadingMore ? "Loading…" : "Load more"}
      </button>
    </div>
  {/if}
{/if}
