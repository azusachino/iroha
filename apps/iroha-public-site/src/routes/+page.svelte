<script lang="ts">
  import { untrack } from "svelte";
  import {
    getCoreRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    type ColumnDef,
    type SortingState,
  } from "@tanstack/table-core";
  import {
    cityGroupsForRoutes,
    filterByYearAndSport,
    monthlyBuckets,
    yearsFromActivities,
  } from "$lib/aggregate";
  import { createSvelteTable } from "$lib/table.svelte";
  import {
    formatDate,
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSport,
  } from "$lib/format";
  import { sportColor } from "$lib/sport";
  import type { Activity } from "$lib/types";
  import RoutesMap from "$lib/components/RoutesMap.svelte";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
  import YearProgressChart from "$lib/components/YearProgressChart.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();
  const summary = $derived(data.summary);
  const activities = $derived(data.activities);
  const routes = $derived(data.routes);
  const meta = $derived(data.meta);

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

  const years = $derived(yearsFromActivities(activities));
  let selectedYear = $state<string | null>(
    untrack(() => yearsFromActivities(data.activities)[0] ?? null),
  );
  let sportFilter = $state<string | null>(null);
  let cityFilter = $state<string | null>(null);

  function selectYear(year: string) {
    selectedYear = year;
    cityFilter = null;
  }

  function toggleSport(sport: string) {
    sportFilter = sportFilter === sport ? null : sport;
    cityFilter = null;
  }

  function toggleCity(city: string) {
    cityFilter = cityFilter === city ? null : city;
  }

  function isNonDistanceSport(sport?: string, distanceM?: number): boolean {
    if (!sport) return true;
    if (distanceM == null || distanceM <= 0) return true;
    const s = sport.toLowerCase();
    return !["run", "walk", "hike", "ride", "cycl", "swim"].some((k) =>
      s.includes(k),
    );
  }

  function isCycling(sport?: string): boolean {
    const s = sport?.toLowerCase() ?? "";
    return s.includes("ride") || s.includes("cycl") || s.includes("bik");
  }

  function isSwimming(sport?: string): boolean {
    return (sport?.toLowerCase() ?? "").includes("swim");
  }

  function formatCyclingSpeed(distanceM?: number, durationS?: number): string {
    if (distanceM == null || durationS == null || durationS <= 0) return "—";
    return `${(distanceM / 1000 / (durationS / 3600)).toFixed(1)} km/h`;
  }

  function formatSwimmingPace(distanceM?: number, durationS?: number): string {
    if (distanceM == null || durationS == null || durationS <= 0) return "—";
    const pace100m = durationS / (distanceM / 100);
    const m = Math.floor(pace100m / 60);
    const s = Math.round(pace100m % 60);
    return `${m}:${String(s).padStart(2, "0")} /100m`;
  }

  function cityLabel(city: string, status?: string): string {
    if (status === "pending") return "Location pending";
    if (status === "unknown" || city === "Unknown")
      return "Location unavailable";
    return city;
  }

  // --- Summary (all-time totals + by-sport are precomputed server-side;
  // everything scoped by year/sport is derived client-side from the full
  // activity list, since there is no live backend to re-query per filter). ---
  const selectedYearTotals = $derived.by(() => {
    const scoped = filterByYearAndSport(activities, selectedYear, sportFilter);
    return scoped.reduce(
      (acc, a) => ({
        activity_count: acc.activity_count + 1,
        distance_m: acc.distance_m + (a.distance_m ?? 0),
        duration_s: acc.duration_s + (a.duration_s ?? 0),
        moving_time_s: acc.moving_time_s + (a.moving_time_s ?? 0),
      }),
      { activity_count: 0, distance_m: 0, duration_s: 0, moving_time_s: 0 },
    );
  });

  const monthlyAll = $derived(
    monthlyBuckets(filterByYearAndSport(activities, null, sportFilter)),
  );

  const monthSlots = $derived.by(() => {
    if (!selectedYear) return [];
    const byKey = new Map(monthlyAll.map((b) => [b.key, b]));
    return MONTH_LABELS.map((label, idx) => {
      const key = `${selectedYear}-${String(idx + 1).padStart(2, "0")}`;
      return { label, key, bucket: byKey.get(key) };
    });
  });

  const monthMetric = $derived(
    monthSlots.reduce((sum, m) => sum + (m.bucket?.distance_m ?? 0), 0) > 0
      ? "distance_m"
      : "activity_count",
  );

  const monthMax = $derived(
    Math.max(
      1,
      ...monthSlots.map((m) => (m.bucket ? m.bucket[monthMetric] : 0)),
    ),
  );

  const sportMax = $derived(
    Math.max(1, ...summary.by_sport.map((s) => s.activity_count)),
  );

  // --- Routes & cities ---
  const selectedYearRoutes = $derived(
    selectedYear
      ? routes.features.filter((f) => f.properties.year === selectedYear)
      : routes.features,
  );
  const filteredRoutes = $derived(
    sportFilter
      ? selectedYearRoutes.filter(
          (f) => f.properties.sport_type === sportFilter,
        )
      : selectedYearRoutes,
  );
  const cityGroups = $derived(cityGroupsForRoutes(filteredRoutes));
  const maxRunCount = $derived(
    Math.max(1, ...cityGroups.map((g) => g.runCount)),
  );
  const hasPendingLocations = $derived(
    cityGroups.some((g) => g.status === "pending"),
  );
  const mappedRoutes = $derived(
    cityFilter
      ? filteredRoutes.filter(
          (f) => (f.properties.city || "Unknown") === cityFilter,
        )
      : filteredRoutes,
  );

  // --- Activities table (TanStack table-core: sorting + pagination over the
  // already-loaded, year/sport-filtered array -- no server round trip). ---
  const filteredActivities = $derived(
    filterByYearAndSport(activities, selectedYear, sportFilter),
  );

  let sorting = $state<SortingState>([{ id: "started_at", desc: true }]);

  const columns: ColumnDef<Activity>[] = [
    { accessorKey: "started_at", id: "started_at", header: "Date" },
    { accessorKey: "sport_type", id: "sport_type", header: "Activity" },
    {
      accessorKey: "distance_m",
      id: "distance_m",
      header: "Distance / Duration",
    },
    {
      accessorKey: "avg_pace_s_per_km",
      id: "avg_pace_s_per_km",
      header: "Pace / HR",
    },
  ];

  const table = createSvelteTable({
    get data() {
      return filteredActivities;
    },
    columns,
    state: {
      get sorting() {
        return sorting;
      },
    },
    onSortingChange: (updater) => {
      sorting = typeof updater === "function" ? updater(sorting) : updater;
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: 20 } },
  });
</script>

<svelte:head>
  <title>iroha · public archive</title>
</svelte:head>

<header class="hero tile">
  <p class="eyebrow">A window into the archive</p>
  <h1>The shape of the miles.</h1>
  <p class="muted">
    A calm, read-only view of the years, routes, and sessions made visible.
  </p>
  <p class="muted">Data as of {formatDateOnly(meta.generated_at)}</p>
</header>

<div class="stat-grid">
  <StatTile
    label="Distance"
    value={formatDistance(selectedYearTotals.distance_m)}
  />
  <StatTile
    label="Activities"
    value={selectedYearTotals.activity_count.toLocaleString()}
  />
  <StatTile
    label="Total time"
    value={formatDuration(
      selectedYearTotals.moving_time_s || selectedYearTotals.duration_s,
    )}
  />
</div>

{#if years.length > 0}
  <nav class="year-tabs" aria-label="Select year">
    {#each years as year (year)}
      <button
        type="button"
        class:active={selectedYear === year}
        onclick={() => selectYear(year)}
      >
        {year}
      </button>
    {/each}
  </nav>

  {#if selectedYear}
    <YearProgressChart
      byMonth={monthlyAll}
      year={selectedYear}
      sportName={sportFilter ? formatSport(sportFilter) : undefined}
    />
  {/if}

  <section class="tile month-chart">
    <div class="month-chart-head">
      Monthly {monthMetric === "distance_m" ? "distance" : "activities"} — {selectedYear}
    </div>
    <div class="month-bars">
      {#each monthSlots as slot (slot.key)}
        {@const value = slot.bucket ? slot.bucket[monthMetric] : 0}
        <div class="month-bar">
          <i style={`height: ${Math.max(2, (value / monthMax) * 100)}%`}></i>
          <strong>{slot.label}</strong>
        </div>
      {/each}
    </div>
  </section>
{/if}

{#if summary.by_sport.length > 0}
  <section class="tile by-sport">
    <div class="section-kicker">All-time by sport</div>
    {#each summary.by_sport as sport (sport.key)}
      <button
        type="button"
        class="sport-row"
        class:active={sportFilter === sport.key}
        onclick={() => toggleSport(sport.key)}
      >
        <SportBadge sport={sport.key} />
        <span class="bar">
          <i
            style={`width: ${Math.max(2, (sport.activity_count / sportMax) * 100)}%; background: ${sportColor(sport.key)}`}
          ></i>
        </span>
        <span class="count">{sport.activity_count}×</span>
      </button>
    {/each}
    {#if sportFilter}
      <button
        type="button"
        class="clear-filter"
        onclick={() => toggleSport(sportFilter!)}
      >
        Clear sport filter ({formatSport(sportFilter)})
      </button>
    {/if}
  </section>
{/if}

<section class="section-heading">
  <h2>Routes &amp; cities</h2>
  {#if hasPendingLocations}
    <p class="muted small">Some locations are waiting for geocoding.</p>
  {/if}
</section>

{#if routes.features.length === 0}
  <p class="muted">No routes recorded yet.</p>
{:else}
  <div class="routes-grid">
    <div class="map-wrap tile">
      <RoutesMap data={{ type: "FeatureCollection", features: mappedRoutes }} />
    </div>
    <div class="cities tile">
      <div class="cities-head">
        <h3>
          {sportFilter
            ? `${formatSport(sportFilter)} cities`
            : "Cities visited"}
        </h3>
        {#if cityFilter}
          <button
            type="button"
            class="clear-filter"
            onclick={() => (cityFilter = null)}
          >
            Show all
          </button>
        {/if}
      </div>
      {#if cityGroups.length === 0}
        <p class="muted small">No route coordinates for this selection.</p>
      {:else}
        <div class="city-grid">
          {#each cityGroups as group (group.city)}
            {@const intensity = group.runCount / maxRunCount}
            <button
              type="button"
              class="city-card"
              class:selected={cityFilter === group.city}
              style={`--intensity: ${intensity}`}
              onclick={() => toggleCity(group.city)}
            >
              <div class="city-name">{cityLabel(group.city, group.status)}</div>
              <div class="city-sports">
                {Array.from(group.sports)
                  .map((s) => formatSport(s))
                  .join(", ")}
              </div>
              <div class="city-count">
                {group.runCount}
                {group.runCount === 1 ? "run" : "runs"}
                {#if group.count > group.runCount}
                  <span class="muted small"
                    >(+{group.count - group.runCount} other)</span
                  >
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}

<section class="section-heading">
  <h2>Activities</h2>
</section>

{#if filteredActivities.length === 0}
  <p class="muted">No activities for this selection.</p>
{:else}
  <div class="table-wrap tile">
    <table>
      <thead>
        {#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
          <tr>
            {#each headerGroup.headers as header (header.id)}
              <th>
                <button
                  type="button"
                  class="sort-header"
                  onclick={header.column.getToggleSortingHandler()}
                >
                  {String(header.column.columnDef.header)}
                  {#if header.column.getIsSorted() === "asc"}▲{:else if header.column.getIsSorted() === "desc"}▼{/if}
                </button>
              </th>
            {/each}
          </tr>
        {/each}
      </thead>
      <tbody>
        {#each table.getRowModel().rows as row (row.id)}
          {@const activity = row.original}
          <tr>
            <td class="nowrap"
              >{formatDate(activity.started_at, activity.timezone)}</td
            >
            <td>
              <div class="activity-cell">
                <SportBadge sport={activity.sport_type} />
                {#if activity.title && formatSport(activity.title) !== formatSport(activity.sport_type)}
                  <span class="activity-title">{activity.title}</span>
                {/if}
              </div>
            </td>
            <td>
              {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                {formatDuration(activity.duration_s ?? activity.moving_time_s)}
              {:else}
                {formatDistance(activity.distance_m)}
              {/if}
            </td>
            <td>
              {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                {#if activity.avg_hr}Avg HR: {formatHr(
                    activity.avg_hr,
                  )}{:else if activity.max_hr}Max HR: {formatHr(
                    activity.max_hr,
                  )}{:else}—{/if}
              {:else if isCycling(activity.sport_type)}
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
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
    {#if table.getPageCount() > 1}
      <div class="pager">
        <button
          type="button"
          disabled={!table.getCanPreviousPage()}
          onclick={() => table.previousPage()}
        >
          Previous
        </button>
        <span class="muted small">
          Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}
        </span>
        <button
          type="button"
          disabled={!table.getCanNextPage()}
          onclick={() => table.nextPage()}
        >
          Next
        </button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .hero {
    padding: 2rem;
    margin-bottom: 1.25rem;
  }
  .eyebrow {
    margin: 0 0 0.4rem;
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .hero h1 {
    margin: 0;
    font-size: clamp(2rem, 5vw, 3.2rem);
    letter-spacing: -0.03em;
  }
  .small {
    font-size: 0.78rem;
  }
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }
  .year-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 1rem;
  }
  .year-tabs button {
    padding: 0.4rem 0.85rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .year-tabs button.active {
    border-color: var(--accent);
    color: var(--accent);
  }
  .month-chart {
    margin: 1rem 0;
    padding: 1rem;
  }
  .month-chart-head {
    margin-bottom: 0.75rem;
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  .month-bars {
    display: grid;
    grid-template-columns: repeat(12, 1fr);
    align-items: end;
    gap: 0.4rem;
    height: 10rem;
  }
  .month-bar {
    display: grid;
    grid-template-rows: 1fr auto;
    height: 100%;
  }
  .month-bar i {
    display: block;
    width: 65%;
    min-height: 2px;
    margin: 0 auto;
    background: var(--accent);
  }
  .month-bar strong {
    margin-top: 0.35rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    font-weight: 500;
    text-align: center;
  }
  .by-sport {
    margin-bottom: 1rem;
    padding: 1rem;
  }
  .section-kicker {
    margin-bottom: 0.65rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .sport-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.4rem 0;
    border: none;
    background: none;
    cursor: pointer;
  }
  .sport-row .bar {
    flex: 1;
    height: 0.5rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }
  .sport-row .bar i {
    display: block;
    height: 100%;
    border-radius: 999px;
  }
  .sport-row .count {
    min-width: 3rem;
    text-align: right;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .sport-row.active .count {
    color: var(--accent);
  }
  .clear-filter {
    margin-top: 0.4rem;
    border: none;
    background: none;
    color: var(--accent);
    font-size: 0.78rem;
    cursor: pointer;
    text-decoration: underline;
  }
  .section-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    margin: 1.5rem 0 0.75rem;
  }
  .section-heading h2 {
    margin: 0;
    font-size: 1.4rem;
    letter-spacing: -0.02em;
  }
  .routes-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 1rem;
  }
  .map-wrap {
    min-height: 20rem;
    padding: 0.5rem;
  }
  .cities {
    padding: 1rem;
    max-height: 24rem;
    overflow-y: auto;
  }
  .cities-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  .cities-head h3 {
    margin: 0;
    font-size: 0.9rem;
  }
  .city-grid {
    display: grid;
    gap: 0.5rem;
  }
  .city-card {
    text-align: left;
    padding: 0.6rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: color-mix(
      in srgb,
      var(--accent) calc(var(--intensity) * 25%),
      var(--surface-2)
    );
    color: var(--text);
    cursor: pointer;
  }
  .city-card.selected {
    outline: 2px solid var(--accent);
  }
  .city-name {
    font-weight: 700;
    font-size: 0.85rem;
  }
  .city-sports,
  .city-count {
    color: var(--text-muted);
    font-size: 0.72rem;
    margin-top: 0.2rem;
  }
  .table-wrap {
    overflow-x: auto;
    padding: 0.25rem;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.85rem;
  }
  th,
  td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  .sort-header {
    border: none;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-weight: 700;
    cursor: pointer;
  }
  .nowrap {
    white-space: nowrap;
  }
  .activity-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .activity-title {
    font-weight: 600;
  }
  .pager {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    padding: 0.75rem;
  }
  .pager button {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: transparent;
    color: var(--text);
    cursor: pointer;
  }
  .pager button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  @media (max-width: 720px) {
    .stat-grid {
      grid-template-columns: 1fr;
    }
    .routes-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
