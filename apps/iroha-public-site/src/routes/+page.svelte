<script lang="ts">
  import { browser } from "$app/environment";
  import { base } from "$app/paths";
  import { page } from "$app/state";
  import { onMount, untrack } from "svelte";
  import {
    activeDaysFromActivities,
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
  import { isDistanceSport, sportColor } from "@iroha/shared/domain/sport";
  import type {
    Activity,
    ActivityDetail as ActivityDetailData,
    RouteFeatureCollection,
  } from "$lib/types";
  import ActivityHeatmap from "@iroha/shared/theme-ui/components/ActivityHeatmap.svelte";
  import MonthlyBarChart from "@iroha/shared/components/MonthlyBarChart.svelte";
  import YearProgressChart from "@iroha/shared/components/YearProgressChart.svelte";
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";
  import StatTile from "@iroha/shared/components/StatTile.svelte";
  import RouteHeader from "@iroha/shared/components/RouteHeader.svelte";
  import PanelFrame from "@iroha/shared/components/PanelFrame.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import type { PageProps } from "./$types";

  let { data }: PageProps = $props();
  const activities = $derived(data.activities);
  let routes = $state<RouteFeatureCollection>({
    type: "FeatureCollection",
    features: [],
  });
  let routesLoading = $state(true);
  const meta = $derived(data.meta);
  const activeDays = $derived(activeDaysFromActivities(activities));
  const latestDay = $derived(
    activities[0]?.started_at.slice(0, 10) ??
      new Date().toISOString().slice(0, 10),
  );

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
  let recordsScroll = $state<HTMLDivElement>();
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
  // Issue #116 review fix: list absence alone is not proof of not-found.
  // The notice is only confirmed by an actual 404 detail response for the
  // currently selected id; pending or failing reads keep the archive
  // fallback without a not-found claim, and the state is reset per selection
  // so a stale 404 can never carry into a new id.
  let confirmedMissingActivityId = $state<string | null>(null);
  $effect(() => {
    const id = selectedActivityId;
    selectedActivityDetail = undefined;
    confirmedMissingActivityId = null;
    if (!id) return;
    let cancelled = false;
    void fetch(`/public/v1/activities/${encodeURIComponent(id)}`)
      .then((res) => {
        if (cancelled) return undefined;
        if (res.status === 404) confirmedMissingActivityId = id;
        return res.ok ? (res.json() as Promise<ActivityDetailData>) : undefined;
      })
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
      (acc, activity) => {
        const hasDistance = activity.distance_m != null;
        const isUnknownDistance =
          !hasDistance && isDistanceSport(activity.sport_type);
        return {
          activity_count: acc.activity_count + 1,
          distance_m: acc.distance_m + (activity.distance_m ?? 0),
          distance_known_count:
            acc.distance_known_count + (hasDistance ? 1 : 0),
          distance_unknown_count:
            acc.distance_unknown_count + (isUnknownDistance ? 1 : 0),
          duration_s: acc.duration_s + (activity.duration_s ?? 0),
          moving_time_s: acc.moving_time_s + (activity.moving_time_s ?? 0),
        };
      },
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

  let routesSectionEl = $state<HTMLElement>();
  let routesRequested = false;

  function ensureRoutesLoaded() {
    if (routesRequested) return;
    routesRequested = true;
    data
      .loadRoutes()
      .then((r) => {
        routes = r;
        routesLoading = false;
      })
      .catch(() => {
        routesLoading = false;
      });
  }

  $effect(() => {
    if (selectedActivityId) {
      ensureRoutesLoaded();
    }
  });

  $effect(() => {
    const el = routesSectionEl;
    if (!el || !browser) return;
    if (typeof IntersectionObserver === "undefined") {
      ensureRoutesLoaded();
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          ensureRoutesLoaded();
          observer.disconnect();
        }
      },
      { rootMargin: "400px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  });

  onMount(() => {
    ensureRoutesLoaded();
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
    title={`${site.name} ${site.byline}`}
    context={`Observed year: ${selectedYear || "No records"} · ${sportFilter ? formatSport(sportFilter) : "All sports"}`}
  >
    {#snippet titleSnippet()}
      <a class="brand" href={`${base}/`} aria-label={`${site.name} home`}>
        <img src={`${base}/favicon.svg`} alt="" width="30" height="30" />
        <h1 class="brand-heading">
          <span class="brand-title">{site.name}</span>
          <span class="brand-byline">{site.byline}</span>
        </h1>
      </a>
    {/snippet}
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
        <ThemeToggle />
        <div class="hero-meta" aria-label="Archive metadata">
          <span>Live</span><span
            >Updated {formatDateOnly(meta.generated_at)}</span
          ><span>iroha v{site.version}</span>
        </div>
      </div>
    {/snippet}
  </RouteHeader>
{/if}

{#snippet selectedRecord(activity: Activity)}
  <a href={page.url.pathname}>← Back to archive</a>
  <h2>{activity.title || formatSport(activity.sport_type)}</h2>
  <p>
    Distance: {formatDistance(activity.distance_m)} · Duration:
    {formatDuration(activity.duration_s)}
  </p>
{/snippet}

{#if selectedActivity}
  {#await import("$lib/components/ActivityDetailView.svelte")}
    <section class="tile" aria-busy="true">
      {@render selectedRecord(selectedActivity)}
      <p role="status">Loading activity details…</p>
    </section>
  {:then { default: DetailView }}
    <DetailView
      activity={selectedActivity}
      detail={selectedActivityDetail}
      {routes}
      backHref={page.url.pathname}
    />
  {:catch}
    <section class="tile" aria-busy="false">
      {@render selectedRecord(selectedActivity)}
      <p role="alert">
        Activity detail view unavailable. Record summary remains.
      </p>
      <button type="button" onclick={() => window.location.reload()}
        >Reload page</button
      >
    </section>
  {/await}
{:else}
  {#if selectedActivityId && !selectedActivity && confirmedMissingActivityId === selectedActivityId}
    <!-- Issue #116: an unknown ?activity= link must be told apart from the
         ordinary archive view. The notice renders only after the detail read
         for this exact id confirmed a 404; a pending or failing read keeps
         the plain archive fallback. -->
    <section
      class="tile notice-tile"
      role="status"
      aria-label="Activity not found"
    >
      <p>
        This activity is not in the public archive. Showing the full archive
        instead.
      </p>
      <a href={page.url.pathname}>View the full archive</a>
    </section>
  {/if}
  <div class="dashboard">
    <div class="stat-grid">
      <div class="stat-tile-wrap" style="--accent: #00e5bf;">
        <StatTile
          label="Distance"
          value={distanceValue}
          sub={distanceSub}
          context={summaryContext}
        />
      </div>
      <div class="stat-tile-wrap" style="--accent: #f59e0b;">
        <StatTile
          label="Activities"
          context={summaryContext}
          value={formatMetricValue(selectedYearTotals.activity_count, "count")}
        />
      </div>
      <div class="stat-tile-wrap" style="--accent: #38bdf8;">
        <StatTile
          label="Running count"
          value={formatMetricValue(selectedYearRunningCount, "count")}
          context={selectedYear
            ? `${selectedYear} · All running records`
            : "No records"}
        />
      </div>
      <div class="stat-tile-wrap" style="--accent: #f43f5e;">
        <StatTile
          label="Total time"
          context={summaryContext}
          value={formatHumanDuration(
            selectedYearTotals.moving_time_s || selectedYearTotals.duration_s,
          )}
        />
      </div>
    </div>

    <ActivityHeatmap days={activeDays} endDay={latestDay} embedded={false} />

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

    <section class="section-heading" bind:this={routesSectionEl}>
      <h2>Routes &amp; cities</h2>
      {#if hasPendingLocations}
        <p class="muted small">Some locations are waiting for geocoding.</p>
      {/if}
    </section>

    {#if routesLoading}
      <div class="routes-grid" aria-busy="true">
        <PanelFrame label="Route map">
          <div class="map-wrap map-loading">
            <p class="muted" role="status">Loading routes map…</p>
          </div>
        </PanelFrame>
        <PanelFrame label="Geography selection">
          <div class="cities cities-loading">
            <p class="muted" role="status">Loading geography data…</p>
          </div>
        </PanelFrame>
      </div>
    {:else if filteredRoutes.length === 0}
      <p class="muted">No routes recorded yet.</p>
    {:else}
      <div class="routes-grid">
        <PanelFrame label="Route map"
          ><div class="map-wrap">
            {#await import("$lib/components/RoutesMap.svelte") then { default: RoutesMap }}
              <RoutesMap
                data={{ type: "FeatureCollection", features: mappedRoutes }}
              />
            {/await}
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
      <PanelFrame label="Activity records">
        <p class="muted small">
          All record columns are preserved. Swipe horizontally, or focus the
          table region and use arrow keys, to reach later columns.
        </p>
        <button
          class="scroll-hint"
          type="button"
          aria-controls="public-records-scroll"
          onclick={() => recordsScroll?.focus({ preventScroll: true })}
          >Focus public records to scroll all columns</button
        >
        <div
          class="table-wrap"
          id="public-records-scroll"
          bind:this={recordsScroll}
          role="region"
          aria-label="Public activity records — horizontally scrollable"
          tabindex="-1"
        >
          <table aria-label="Public activity records">
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
  .notice-tile {
    margin-bottom: 1.25rem;
  }
  .notice-tile a {
    color: var(--accent);
  }
  .hero {
    position: relative;
    overflow: hidden;
    padding: 2.25rem 1.75rem;
    margin-bottom: 1.25rem;
    background:
      radial-gradient(
        ellipse 70% 60% at 85% 15%,
        color-mix(in srgb, var(--accent) 14%, transparent),
        transparent 70%
      ),
      radial-gradient(
        ellipse 50% 50% at 15% 85%,
        color-mix(in srgb, #38bdf8 10%, transparent),
        transparent 70%
      ),
      var(--tile-surface);
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
    font-weight: 800;
    background: linear-gradient(135deg, var(--text) 40%, var(--accent) 100%);
    -webkit-background-clip: text;
    background-clip: text;
    -webkit-text-fill-color: transparent;
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
  .stat-tile-wrap {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .stat-tile-wrap :global(.stat-tile) {
    height: 100%;
    border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
    background: linear-gradient(
      180deg,
      color-mix(in srgb, var(--accent) 9%, var(--surface)) 0%,
      var(--surface) 100%
    );
    transition:
      transform 0.15s ease,
      box-shadow 0.15s ease;
  }
  .stat-tile-wrap :global(.stat-tile:hover) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px -3px
      color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .stat-tile-wrap :global(.stat-label) {
    color: var(--accent);
    font-weight: 750;
    letter-spacing: 0.09em;
  }
  .brand-heading {
    margin: 0;
    display: inline-flex;
    align-items: baseline;
    gap: 0.45rem;
    font-size: var(--type-title);
    line-height: 1.25;
  }
  .brand-title {
    font-weight: 800;
    letter-spacing: -0.02em;
    color: var(--text);
  }
  .brand-byline {
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: var(--type-caption);
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .public-actions {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.35rem;
  }
  .public-actions .hero-meta {
    letter-spacing: normal;
    text-transform: none;
  }
  .public-actions .hero-meta span + span {
    padding-left: 0;
    border-left: 0;
  }
  @media (max-width: 640px) {
    .public-actions {
      align-items: flex-start;
    }
  }
  .year-tabs {
    display: flex;
    align-items: center;
    flex-wrap: nowrap;
    gap: 0.45rem;
    max-width: 100%;
    overflow-x: auto;
    overscroll-behavior-x: contain;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    padding: 0.25rem 0;
    margin-bottom: 0;
  }
  .year-tabs::-webkit-scrollbar {
    display: none;
  }
  .year-tabs button {
    flex-shrink: 0;
    padding: 0.42rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text-muted);
    font-size: var(--type-label);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .year-tabs button:hover {
    border-color: var(--text);
    color: var(--text);
  }
  .year-tabs button.active {
    border-color: var(--tab-active-border);
    background: var(--tab-active-bg);
    color: var(--tab-active-text);
    font-weight: 750;
    box-shadow: var(--tab-active-shadow);
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
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 750;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .sport-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.45rem 0.6rem;
    border-radius: 8px;
    border: none;
    background: none;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .sport-row:hover {
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
  }
  .sport-row .bar {
    flex: 1;
    height: 0.6rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }
  .sport-row .bar i {
    display: block;
    height: 100%;
    border-radius: 999px;
    box-shadow: 0 0 10px currentColor;
  }
  .sport-row .count {
    min-width: 3rem;
    text-align: right;
    color: var(--text-muted);
    font-size: var(--type-label);
  }
  .sport-row.active .count {
    color: var(--accent);
    font-weight: 700;
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
  .map-loading,
  .cities-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 18rem;
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
  .scroll-hint {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .scroll-hint:focus,
  .scroll-hint:focus-visible {
    position: static;
    width: auto;
    height: auto;
    min-height: 44px;
    padding: 0.5rem 1rem;
    margin: 0.5rem 0;
    overflow: visible;
    clip: auto;
    white-space: normal;
    background: var(--surface, #1e222b);
    color: var(--text, #fff);
    border: 1px solid var(--border, #333);
    border-radius: 6px;
    font-size: var(--type-label);
    font-weight: 500;
    cursor: pointer;
    display: inline-block;
    outline: 2px solid var(--color-focus, var(--accent));
    outline-offset: 2px;
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
  tbody tr {
    transition: background 0.15s ease;
  }
  tbody tr:hover {
    background: color-mix(in srgb, var(--accent) 7%, transparent);
  }
  .sort-header {
    min-height: 44px;
    min-width: 44px;
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
    color: var(--accent);
    font-weight: 700;
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
