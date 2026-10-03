<script lang="ts">
  import { browser } from "$app/environment";
  import { base } from "$app/paths";
  import { page } from "$app/state";
  import { onMount, untrack } from "svelte";
  import {
    cityGroupsForRoutes,
    filterByYearAndSport,
    monthlyBuckets,
    yearsFromActivities,
  } from "$lib/aggregate";
  import {
    formatDate,
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatHr,
    formatHumanDuration,
    formatMetricValue,
    formatPace,
    formatSport,
  } from "$lib/format";
  import { site } from "$lib/site";
  import { sportColor } from "@iroha/shared/domain/sport";
  import type {
    Activity,
    ActivityDetail as ActivityDetailData,
  } from "$lib/types";
  import ApprovedActivityDetail from "$lib/components/ApprovedActivityDetail.svelte";
  import RoutesMap from "$lib/components/RoutesMap.svelte";
  import ActivityDetail from "$lib/components/ActivityDetail.svelte";
  import MonthlyBarChart from "@iroha/shared/components/MonthlyBarChart.svelte";
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";
  import StatTile from "@iroha/shared/components/StatTile.svelte";
  import RouteHeader from "@iroha/shared/components/RouteHeader.svelte";
  import PanelFrame from "@iroha/shared/components/PanelFrame.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import YearProgressChart from "@iroha/shared/components/YearProgressChart.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();
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
  let selectedYear = $state<string>(
    untrack(() => yearsFromActivities(data.activities)[0] ?? ""),
  );
  let sportFilter = $state<string | null>(null);
  let cityFilter = $state<string | null>(null);
  let selectedActivityId = $state<string | null>(
    browser
      ? new URLSearchParams(window.location.search).get("activity")
      : null,
  );
  const activityQuery = $derived(page.url.search);
  const selectedActivity = $derived(
    activities.find((activity) => activity.id === selectedActivityId),
  );
  // Details are fetched per activity on selection instead of shipping every
  // activity's route and samplings up front.
  let selectedActivityDetail = $state<ActivityDetailData | undefined>();
  $effect(() => {
    const id = selectedActivityId;
    selectedActivityDetail = undefined;
    if (!id) return;
    let cancelled = false;
    void fetch(`/public/v1/activities/${encodeURIComponent(id)}`)
      .then((res) =>
        res.ok ? (res.json() as Promise<ActivityDetailData>) : undefined,
      )
      .then((detail) => {
        if (!cancelled) selectedActivityDetail = detail;
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  });

  $effect(() => {
    const activityId = new URLSearchParams(
      activityQuery || window.location.search,
    ).get("activity");
    if (activityId !== selectedActivityId) selectedActivityId = activityId;
  });

  function activityHref(id: string): string {
    const url = new URL(page.url);
    url.searchParams.set("activity", id);
    return `${url.pathname}${url.search}`;
  }

  function selectYear(year: string) {
    selectedYear = year;
    cityFilter = null;
    visibleCount = batchSize;
  }

  function toggleSport(sport: string) {
    sportFilter = sportFilter === sport ? null : sport;
    cityFilter = null;
    visibleCount = batchSize;
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

  function hasMeaningfulActivityTitle(activity: Activity): boolean {
    const title = formatSport(activity.title);
    const sport = formatSport(activity.sport_type);
    const genericTitles: Record<string, string[]> = {
      Run: ["Run", "Running"],
      Walk: ["Walk", "Walking"],
      Ride: ["Ride", "Riding", "Cycling"],
      Swim: ["Swim", "Swimming"],
    };
    return title !== "—" && !(genericTitles[sport] ?? [sport]).includes(title);
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
      (acc, activity) => ({
        activity_count: acc.activity_count + 1,
        distance_m: acc.distance_m + (activity.distance_m ?? 0),
        distance_known_count:
          acc.distance_known_count + (activity.distance_m == null ? 0 : 1),
        distance_unknown_count:
          acc.distance_unknown_count + (activity.distance_m == null ? 1 : 0),
        duration_s: acc.duration_s + (activity.duration_s ?? 0),
        moving_time_s: acc.moving_time_s + (activity.moving_time_s ?? 0),
      }),
      {
        activity_count: 0,
        distance_m: 0,
        distance_known_count: 0,
        distance_unknown_count: 0,
        duration_s: 0,
        moving_time_s: 0,
      },
    );
  });
  const distanceSub = $derived.by(() => {
    const { distance_known_count, distance_unknown_count } = selectedYearTotals;
    if (distance_unknown_count === 0) return undefined;
    if (distance_known_count === 0)
      return `Distance unavailable for ${distance_unknown_count} ${distance_unknown_count === 1 ? "activity" : "activities"}`;
    return `Known distance for ${distance_known_count} of ${selectedYearTotals.activity_count} activities; ${distance_unknown_count} unavailable`;
  });
  const summaryContext = $derived(
    `${selectedYear || "No records"} · ${sportFilter ? formatSport(sportFilter) : "All sports"}`,
  );
  const distanceValue = $derived(
    selectedYearTotals.distance_unknown_count > 0 &&
      selectedYearTotals.distance_known_count === 0
      ? "—"
      : formatDistance(selectedYearTotals.distance_m),
  );
  const selectedYearRunningCount = $derived(
    activities.filter(
      (activity) =>
        activity.started_at.slice(0, 4) === selectedYear &&
        activity.sport_type.toLowerCase().includes("run"),
    ).length,
  );

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
    monthSlots.some((slot) => (slot.bucket?.distance_unknown_count ?? 0) > 0) ||
      !monthSlots.some((slot) => (slot.bucket?.distance_known_count ?? 0) > 0)
      ? "activity_count"
      : "distance_m",
  );

  const sportBuckets = $derived.by(() => {
    const buckets = new Map<
      string,
      {
        key: string;
        activity_count: number;
        distance_m: number;
        duration_s: number;
        moving_time_s: number;
      }
    >();
    for (const activity of filterByYearAndSport(
      activities,
      selectedYear,
      null,
    )) {
      const bucket = buckets.get(activity.sport_type) ?? {
        key: activity.sport_type,
        activity_count: 0,
        distance_m: 0,
        duration_s: 0,
        moving_time_s: 0,
      };
      bucket.activity_count += 1;
      bucket.distance_m += activity.distance_m ?? 0;
      bucket.duration_s += activity.duration_s ?? 0;
      bucket.moving_time_s += activity.moving_time_s ?? 0;
      buckets.set(activity.sport_type, bucket);
    }
    return Array.from(buckets.values()).sort(
      (a, b) => b.activity_count - a.activity_count,
    );
  });

  const sportMax = $derived(
    Math.max(1, ...sportBuckets.map((s) => s.activity_count)),
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

  // --- Activities table (sorting + progressive rendering over the already-
  // loaded, year/sport-filtered array -- no server round trip). ---
  const filteredActivities = $derived(
    filterByYearAndSport(activities, selectedYear, sportFilter),
  );

  type SortKey =
    "started_at" | "sport_type" | "distance_m" | "avg_pace_s_per_km";
  let sorting = $state<{ id: SortKey; desc: boolean }[]>([
    { id: "started_at", desc: true },
  ]);
  const batchSize = 50;
  let visibleCount = $state(batchSize);
  const columns: { id: SortKey; header: string }[] = [
    { id: "started_at", header: "Date" },
    { id: "sport_type", header: "Activity" },
    { id: "distance_m", header: "Distance / Duration" },
    { id: "avg_pace_s_per_km", header: "Pace / HR" },
  ];

  // Preserve the former TanStack v8 toggle defaults, including Shift state.
  // The existing row ordering below deliberately uses only the leading key.
  function toggleSort(id: SortKey, event: MouseEvent) {
    const current = sorting.find((sort) => sort.id === id);
    const firstDesc = typeof filteredActivities[0]?.[id] !== "string";
    const remove = current && current.desc !== firstDesc;
    const next = {
      id,
      desc: remove ? false : current ? !current.desc : firstDesc,
    };
    if (event.shiftKey && sorting.length) {
      sorting = remove
        ? sorting.filter((sort) => sort.id !== id)
        : current
          ? sorting.map((sort) => (sort.id === id ? next : sort))
          : [...sorting, next];
    } else {
      sorting = remove && sorting.at(-1)?.id === id ? [] : [next];
    }
  }

  const sortedActivities = $derived.by(() => {
    const rows = [...filteredActivities];
    const sort = sorting[0];
    if (!sort) return rows;
    const numeric = ["distance_m", "avg_pace_s_per_km"].includes(sort.id);
    const key = sort.id as keyof Activity;
    rows.sort((a, b) => {
      const left = numeric ? Number(a[key] ?? -Infinity) : String(a[key] ?? "");
      const right = numeric
        ? Number(b[key] ?? -Infinity)
        : String(b[key] ?? "");
      const result = left < right ? -1 : left > right ? 1 : 0;
      return sort.desc ? -result : result;
    });
    return rows;
  });

  const visibleActivities = $derived(sortedActivities.slice(0, visibleCount));
  const hasMoreActivities = $derived(visibleCount < sortedActivities.length);

  function loadMore() {
    visibleCount = Math.min(visibleCount + batchSize, sortedActivities.length);
  }

  function loadMoreNearBottom() {
    if (!hasMoreActivities) return;
    const remaining =
      document.documentElement.scrollHeight -
      (window.scrollY + window.innerHeight);
    if (remaining < 600) loadMore();
  }

  onMount(() => {
    window.addEventListener("scroll", loadMoreNearBottom, { passive: true });
    return () => window.removeEventListener("scroll", loadMoreNearBottom);
  });
</script>

<svelte:head>
  <title>{site.name} {site.byline} · public archive</title>
</svelte:head>

{#if selectedActivity}
  <header class="hero tile">
    <div class="hero-topline">
      <a class="brand" href={`${base}/`} aria-label={`${site.name} home`}>
        <img src={`${base}/favicon.svg`} alt="" width="28" height="28" />
        <span class="eyebrow">{site.name} {site.byline}</span>
      </a>
      <ThemeToggle />
    </div>
    <h1>The shape of the miles.</h1>
    <p class="hero-summary">
      A public field guide to the routes and rhythms made visible.
    </p>
    <div class="hero-meta" aria-label="Archive metadata">
      <span>Live</span>
      <span>Updated {formatDateOnly(meta.generated_at)}</span>
      <span>iroha v{site.version}</span>
    </div>
  </header>
{:else}
  <RouteHeader
    title="Public archive"
    context={`Observed year: ${selectedYear || "No records"} · ${sportFilter ? formatSport(sportFilter) : "All sports"}`}
  >
    {#if years.length > 0}
      <nav class="year-tabs" aria-label="Select year">
        {#each years as year (year)}
          <button
            type="button"
            class:active={selectedYear === year}
            onclick={() => selectYear(year)}>{year}</button
          >
        {/each}
      </nav>
    {/if}
    {#snippet actions()}
      <div class="public-actions">
        <div class="hero-topline">
          <a class="brand" href={`${base}/`} aria-label={`${site.name} home`}>
            <img src={`${base}/favicon.svg`} alt="" width="28" height="28" />
            <span class="eyebrow">{site.name} {site.byline}</span>
          </a>
          <ThemeToggle />
        </div>
        <div class="hero-meta" aria-label="Archive metadata">
          <span>Live</span><span
            >Updated {formatDateOnly(meta.generated_at)}</span
          ><span>iroha v{site.version}</span>
        </div>
      </div>
    {/snippet}
  </RouteHeader>
{/if}

{#if selectedActivity && selectedActivityDetail}
  <ApprovedActivityDetail
    detail={selectedActivityDetail}
    backHref={page.url.pathname}
  />
{:else if selectedActivity}
  <ActivityDetail
    activity={selectedActivity}
    {routes}
    backHref={page.url.pathname}
  />
{:else}
  <div class="dashboard">
    <div class="stat-grid">
      <StatTile
        label="Distance"
        value={distanceValue}
        sub={distanceSub}
        context={summaryContext}
      />
      <StatTile
        label="Activities"
        context={summaryContext}
        value={formatMetricValue(selectedYearTotals.activity_count, "count")}
      />
      <StatTile
        label="Running count"
        value={formatMetricValue(selectedYearRunningCount, "count")}
        context={selectedYear
          ? `${selectedYear} · All running records`
          : "No records"}
        sub={selectedYear ? `runs in ${selectedYear}` : undefined}
      />
      <StatTile
        label="Total time"
        context={summaryContext}
        value={formatHumanDuration(
          selectedYearTotals.moving_time_s || selectedYearTotals.duration_s,
        )}
      />
    </div>

    {#if years.length > 0}
      {#if selectedYear}
        <div class="analytics-grid">
          <PanelFrame label="Cumulative distance">
            <YearProgressChart
              embedded
              byMonth={monthlyAll}
              year={selectedYear}
              sportName={sportFilter ? formatSport(sportFilter) : undefined}
            />
          </PanelFrame>
          <PanelFrame label="Monthly observations">
            <MonthlyBarChart
              embedded
              points={monthSlots.map((slot) => ({
                label: slot.label,
                value: slot.bucket?.[monthMetric] ?? 0,
              }))}
              metric={monthMetric}
              year={selectedYear}
            />
          </PanelFrame>
        </div>
      {/if}
    {/if}

    {#if sportBuckets.length > 0}
      <PanelFrame label="Activities by sport"
        ><section class="by-sport">
          <div class="section-kicker">
            {selectedYear} by sport
          </div>
          {#each sportBuckets as sport (sport.key)}
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
        </section></PanelFrame
      >
    {/if}

    <section class="section-heading">
      <h2>Routes &amp; cities</h2>
      {#if hasPendingLocations}
        <p class="muted small">Some locations are waiting for geocoding.</p>
      {/if}
    </section>

    {#if filteredRoutes.length === 0}
      <p class="muted">No routes recorded yet.</p>
    {:else}
      <div class="routes-grid">
        <PanelFrame label="Route map"
          ><div class="map-wrap">
            <RoutesMap
              data={{ type: "FeatureCollection", features: mappedRoutes }}
            />
          </div></PanelFrame
        >
        <PanelFrame label="Geography selection"
          ><div class="cities">
            <p class="muted small">
              {selectedYear} · {sportFilter
                ? formatSport(sportFilter)
                : "All sports"} · {cityFilter
                ? cityLabel(cityFilter)
                : "All cities"} (geography only)
            </p>
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
              <p class="muted small">
                No route coordinates for this selection.
              </p>
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
                    <div class="city-name">
                      {cityLabel(group.city, group.status)}
                    </div>
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
          </div></PanelFrame
        >
      </div>
    {/if}

    <section class="section-heading">
      <h2>Activities</h2>
    </section>

    {#if filteredActivities.length === 0}
      <p class="muted">No activities for this selection.</p>
    {:else}
      <PanelFrame label="Activity records"
        ><div class="table-wrap">
          <table>
            <thead>
              <tr>
                {#each columns as column (column.id)}
                  {@const activeSort = sorting.find(
                    (sort) => sort.id === column.id,
                  )}
                  <th
                    aria-sort={sorting[0]?.id === column.id
                      ? sorting[0].desc
                        ? "descending"
                        : "ascending"
                      : "none"}
                  >
                    <button
                      type="button"
                      class="sort-header"
                      onclick={(event) => toggleSort(column.id, event)}
                    >
                      {column.header}
                      {#if activeSort}{activeSort.desc ? "▼" : "▲"}{/if}
                    </button>
                  </th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each visibleActivities as activity (activity.id)}
                <tr>
                  <td class="nowrap">
                    <a class="activity-link" href={activityHref(activity.id)}>
                      {formatDate(activity.started_at, activity.timezone)}
                    </a>
                  </td>
                  <td>
                    <div class="activity-cell">
                      <SportBadge sport={activity.sport_type} />
                      {#if hasMeaningfulActivityTitle(activity)}
                        <span class="activity-title">{activity.title}</span>
                      {/if}
                    </div>
                  </td>
                  <td>
                    {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                      {formatDuration(
                        activity.duration_s ?? activity.moving_time_s,
                      )}
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
                      {#if activity.avg_hr}· {formatHr(activity.avg_hr)}{/if}
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
          {#if hasMoreActivities}
            <div class="load-more">
              <button type="button" onclick={loadMore}>Load more</button>
              <span class="muted small">
                Showing {visibleActivities.length} of {sortedActivities.length}
              </span>
            </div>
          {/if}
        </div></PanelFrame
      >
    {/if}
  </div>
{/if}

<style>
  .hero {
    padding: 2rem;
    margin-bottom: 1.25rem;
  }
  .eyebrow {
    margin: 0;
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .brand {
    display: inline-flex;
    min-height: 2.75rem;
    align-items: center;
    gap: 0.55rem;
    border-radius: 8px;
    color: inherit;
    text-decoration: none;
  }
  .brand img {
    flex: none;
    border-radius: 7px;
  }
  .brand:hover .eyebrow {
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .brand:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 3px;
  }
  .hero-topline {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.4rem;
  }
  .hero h1 {
    margin: 0;
    font-size: clamp(var(--type-title), 5vw, var(--type-display));
    letter-spacing: -0.03em;
  }
  .hero-summary {
    max-width: 34rem;
    margin: 0.7rem 0 1.15rem;
    color: var(--text-muted);
    font-size: var(--type-body);
    line-height: 1.5;
  }
  .hero-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem 1rem;
    color: var(--text-muted);
    font-size: var(--type-caption);
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }
  .hero-meta span + span {
    padding-left: 1rem;
    border-left: 1px solid var(--border);
  }
  @media (max-width: 640px) {
    .hero-meta span + span {
      padding-left: 0;
      border-left: 0;
    }
  }
  .small {
    font-size: var(--type-label);
  }
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }
  .public-actions {
    max-width: 23rem;
  }
  .public-actions .hero-meta {
    letter-spacing: normal;
    text-transform: none;
  }
  .public-actions .hero-meta span + span {
    padding-left: 0;
    border-left: 0;
  }
  .year-tabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0;
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
  .dashboard {
    display: grid;
    gap: 1rem;
  }
  .analytics-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
  .by-sport {
    min-width: 0;
  }
  .section-kicker {
    margin-bottom: 0.65rem;
    color: var(--text-muted);
    font-size: var(--type-caption);
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
    font-size: var(--type-label);
  }
  .sport-row.active .count {
    color: var(--accent);
  }
  .clear-filter {
    margin-top: 0.4rem;
    border: none;
    background: none;
    color: var(--accent);
    font-size: var(--type-label);
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
    font-size: var(--type-title);
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
    min-width: 0;
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
    font-size: var(--type-label);
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
    font-size: var(--type-label);
  }
  .city-sports,
  .city-count {
    color: var(--text-muted);
    font-size: var(--type-caption);
    margin-top: 0.2rem;
  }
  .table-wrap {
    overflow-x: auto;
    padding: 0.25rem;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--type-label);
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
  .activity-link {
    font-weight: 650;
  }
  .activity-title {
    font-weight: 600;
  }
  .load-more {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    padding: 0.75rem;
  }
  .load-more button {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: transparent;
    color: var(--text);
    cursor: pointer;
  }
  @media (max-width: 768px) {
    .stat-grid {
      grid-template-columns: 1fr;
    }
    .routes-grid {
      grid-template-columns: 1fr;
    }
    .analytics-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
