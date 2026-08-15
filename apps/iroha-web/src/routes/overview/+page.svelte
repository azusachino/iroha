<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import {
    getActivityRoutes,
    getActivityOverview,
    getMediaAggregates,
    getSleepOverview,
    type Activity,
    type ActivityActiveDay,
    type MediaAggregates,
    type RouteFeatureCollection,
    type SleepOverview,
    type Summary,
  } from "$lib/api";
  import ArchiveTotals from "$lib/components/ArchiveTotals.svelte";
  import DomainTile from "$lib/components/DomainTile.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import Heatmap from "$lib/components/Heatmap.svelte";
  import RouteFootprint from "$lib/components/RouteFootprint.svelte";
  import SportBadge from "@iroha/shared/SportBadge.svelte";
  import StatTile from "@iroha/shared/StatTile.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
  } from "$lib/format";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  const RECENT_ACTIVITY_LIMIT = 5;

  let summary = $state<Summary | null>(null);
  let summaryError = $state<string | null>(null);
  let summaryLoading = $state(true);

  let activities = $state<Activity[]>([]);
  let activeDays = $state<ActivityActiveDay[]>([]);
  let currentStreak = $state(0);
  let activitiesError = $state<string | null>(null);
  let activitiesLoading = $state(true);

  let routes = $state<RouteFeatureCollection | null>(null);
  let routesError = $state<string | null>(null);
  let routesLoading = $state(true);

  // Recent-nights window for the Overview's sleep tile -- same averaging
  // approach as the Night page, just a smaller recent slice since this is a
  // summary tile, not the full history.
  const SLEEP_SWEEP_LIMIT = 30;
  let sleepOverview = $state<SleepOverview | null>(null);
  let sleepSessionCount = $state<number | null>(null);
  let sleepLoading = $state(true);

  let mediaAggregates = $state<MediaAggregates | null>(null);
  let mediaLoading = $state(true);

  const theme = useTheme();

  const recentActivities = $derived(activities.slice(0, RECENT_ACTIVITY_LIMIT));

  const sleepSummary = $derived({
    averageAsleepS: sleepOverview?.average_asleep_s ?? 0,
    averageEfficiency: sleepOverview?.average_efficiency ?? 0,
    nightCount: sleepOverview?.main_sleep_count ?? 0,
  });
  const heatmapDates = $derived(activeDays.map((day) => day.day));
  const streak = $derived(currentStreak);
  const activityCount = $derived(summary?.totals.activity_count ?? 0);
  const archiveRecordCount = $derived(
    activityCount +
      (sleepSessionCount ?? 0) +
      (mediaAggregates?.totals.item_count ?? 0),
  );
  const totalDistance = $derived(formatDistance(summary?.totals.distance_m));
  const totalDuration = $derived(
    formatDuration(summary?.totals.moving_time_s || summary?.totals.duration_s),
  );
  const streakValue = $derived(streak === 1 ? "1 day" : `${streak} days`);

  function errorMessage(error: unknown): string {
    return error instanceof Error ? error.message : String(error);
  }

  async function loadActivityOverview() {
    summaryLoading = true;
    activitiesLoading = true;
    summaryError = null;
    activitiesError = null;
    try {
      const overview = await getActivityOverview({
        recent: RECENT_ACTIVITY_LIMIT,
      });
      summary = overview.summary;
      activities = overview.recent;
      activeDays = overview.active_days;
      currentStreak = overview.current_streak;
    } catch (error) {
      const message = errorMessage(error);
      summaryError = message;
      activitiesError = message;
    } finally {
      summaryLoading = false;
      activitiesLoading = false;
    }
  }

  async function loadRoutes() {
    routesLoading = true;
    routesError = null;
    try {
      routes = await getActivityRoutes();
    } catch (error) {
      routesError = errorMessage(error);
    } finally {
      routesLoading = false;
    }
  }

  async function loadSleep() {
    sleepLoading = true;
    try {
      sleepOverview = await getSleepOverview({ recent: SLEEP_SWEEP_LIMIT });
      sleepSessionCount = sleepOverview.session_count;
    } catch {
      // The Overview's sleep tile is supplemental -- a failure here
      // shouldn't block the rest of the dashboard from rendering.
      sleepOverview = null;
      sleepSessionCount = null;
    } finally {
      sleepLoading = false;
    }
  }

  async function loadMedia() {
    mediaLoading = true;
    try {
      mediaAggregates = await getMediaAggregates({});
    } catch {
      mediaAggregates = null;
    } finally {
      mediaLoading = false;
    }
  }

  async function reloadDashboard() {
    await Promise.all([
      loadActivityOverview(),
      loadRoutes(),
      loadSleep(),
      loadMedia(),
    ]);
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
    const units100m = distanceM / 100;
    const paceSPer100m = durationS / units100m;
    const mins = Math.floor(paceSPer100m / 60);
    const secs = Math.round(paceSPer100m % 60);
    return `${mins}:${String(secs).padStart(2, "0")}/100m`;
  }

  onMount(() => {
    void loadActivityOverview();
    void loadRoutes();
    void loadSleep();
    void loadMedia();
  });
</script>

<svelte:head>
  <title>Overview · iroha</title>
</svelte:head>

<section class="dashboard-shell">
  <ArchiveTotals
    activityCount={summaryLoading || summaryError ? null : activityCount}
    nightCount={sleepLoading ? null : sleepSessionCount}
    mediaCount={mediaLoading || !mediaAggregates
      ? null
      : mediaAggregates.totals.item_count}
  />
  {#if hasThemeRoute(theme.definition(), "dashboard")}
    <LoadingBoundary
      loading={summaryLoading || activitiesLoading}
      ready={!summaryLoading && !activitiesLoading}
      preserveLayout
      label="Loading overview…"
    >
      <ThemeRouteRenderer
        route="dashboard"
        props={{
          summary,
          activities,
          routes,
          streak: streakValue,
          loading: summaryLoading || activitiesLoading,
          error: summaryError || activitiesError,
          routesLoading,
          routesError,
          sleepSummary,
          sleepLoading,
          mediaAggregates,
          mediaLoading,
          onLoadRoutes: () => void loadRoutes(),
          onRetry: () => void reloadDashboard(),
          onOpenActivity: (id: string) => void goto(`/motion/${id}`),
          onOpenSport: (sport: string) =>
            void goto(`/motion?sport=${encodeURIComponent(sport)}`),
        }}
      >
        {#snippet children()}
          <RouteFootprint
            {routes}
            loading={routesLoading}
            error={routesError}
            onLoad={() => void loadRoutes()}
          />
        {/snippet}
      </ThemeRouteRenderer>
    </LoadingBoundary>
  {:else}
    <RouteIntro
      eyebrow="Observatory / long view"
      title="Your history, in view."
      description="A long view of the movement archive: accumulated distance, recent sessions, route footprint, and the data domains available to explore."
    />

    <div class="stats-grid" aria-label="Activity totals">
      <StatTile
        label="Total distance"
        value={summaryLoading || summaryError ? "—" : totalDistance}
        sub={summaryLoading
          ? "Loading totals…"
          : summaryError
            ? "Summary unavailable"
            : undefined}
      />
      <StatTile
        label="Archive records"
        value={summaryLoading || summaryError
          ? "—"
          : archiveRecordCount.toLocaleString()}
        sub={summaryLoading
          ? "Loading totals…"
          : summaryError
            ? "Summary unavailable"
            : "Activities, nights, and media items"}
      />
      <StatTile
        label="Total time"
        value={summaryLoading || summaryError ? "—" : totalDuration}
        sub={summaryLoading
          ? "Loading totals…"
          : summaryError
            ? "Summary unavailable"
            : undefined}
      />
      <StatTile
        label="Current streak"
        value={activitiesLoading || activitiesError ? "—" : streakValue}
        sub={activitiesLoading
          ? "Loading activity days…"
          : activitiesError
            ? "Activity history unavailable"
            : "Consecutive days ending today"}
      />
    </div>

    <div class="bento-grid">
      {#if activitiesLoading}
        <section class="status-tile tile heatmap-tile">
          <p>Loading activity history…</p>
        </section>
      {:else if activitiesError}
        <section class="status-tile tile heatmap-tile">
          <p>Activity history could not be loaded.</p>
        </section>
      {:else}
        <Heatmap dates={heatmapDates} title="Activity history" />
      {/if}

      <section class="recent-tile tile">
        <header class="tile-heading">
          <div>
            <h2>Recent activity</h2>
            <p>Start where you last left off.</p>
          </div>
          <a href="/motion">View all</a>
        </header>
        {#if activitiesLoading}
          <p class="muted">Loading activities…</p>
        {:else if activitiesError}
          <p class="error">Recent activity could not be loaded.</p>
        {:else if recentActivities.length === 0}
          <p class="muted">No activities imported yet.</p>
        {:else}
          <ul class="recent-list">
            {#each recentActivities as activity (activity.id)}
              <li>
                <a class="recent-row" href={`/motion/${activity.id}`}>
                  <SportBadge sport={activity.sport_type} />
                  <span class="recent-title"
                    >{activity.title || "Untitled activity"}</span
                  >
                  <span class="recent-metrics">
                    {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                      {formatDuration(
                        activity.duration_s ?? activity.moving_time_s,
                      )}
                      {#if activity.avg_hr}
                        · Avg HR: {formatHr(activity.avg_hr)}
                      {/if}
                    {:else}
                      {formatDistance(activity.distance_m)}
                      · {formatDuration(
                        activity.duration_s ?? activity.moving_time_s,
                      )}
                      · {#if isCycling(activity.sport_type)}
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
                    {/if}
                  </span>
                  <span class="recent-date"
                    >{formatDate(activity.started_at, activity.timezone)}</span
                  >
                </a>
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <section class="routes-tile tile">
        <header class="tile-heading">
          <div>
            <h2>All routes</h2>
            <p>Your privacy-trimmed route footprint.</p>
          </div>
        </header>
        <div class="dashboard-map">
          <RouteFootprint
            {routes}
            loading={routesLoading}
            error={routesError}
            onLoad={() => void loadRoutes()}
          />
        </div>
      </section>

      <section class="domains-tile">
        <header class="domains-heading">
          <h2>Data domains</h2>
          <p>Expand your cockpit as more of your life arrives here.</p>
        </header>
        <div class="domain-grid">
          <DomainTile
            name="Activity"
            stat={summaryLoading || summaryError
              ? "Loading activity count…"
              : `${activityCount.toLocaleString()} sessions`}
            href="/motion"
            state="active"
          />
          <DomainTile
            name="Sleep"
            stat={sleepLoading || sleepSessionCount == null
              ? "Loading night count…"
              : `${sleepSessionCount.toLocaleString()} sessions`}
            href="/night"
            state="active"
          />
          <DomainTile
            name="Media"
            stat={mediaLoading || !mediaAggregates
              ? "Loading media count…"
              : `${mediaAggregates.totals.item_count.toLocaleString()} items`}
            href="/library"
            state="active"
          />
        </div>
      </section>
    </div>
  {/if}
</section>

<style>
  .dashboard-shell {
    display: grid;
    gap: 1.25rem;
  }

  .tile-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .tile-heading h2,
  .domains-heading h2 {
    margin: 0;
  }

  .tile-heading p,
  .domains-heading p {
    margin: 0.35rem 0 0;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .bento-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .bento-grid :global(.heatmap) {
    grid-column: 1 / -1;
  }

  .status-tile,
  .recent-tile,
  .routes-tile {
    padding: 1rem;
  }

  .status-tile {
    grid-column: 1 / -1;
    min-height: 16rem;
    display: grid;
    place-items: center;
    color: var(--text-muted);
  }

  .tile-heading h2,
  .domains-heading h2 {
    font-size: 1rem;
  }

  .tile-heading p,
  .domains-heading p {
    color: var(--text-muted);
    font-size: 0.84rem;
  }

  .tile-heading a {
    font-size: 0.84rem;
    white-space: nowrap;
  }

  .recent-list {
    list-style: none;
    margin: 1rem 0 0;
    padding: 0;
  }

  .recent-list li + li {
    border-top: 1px solid var(--border);
  }

  .recent-row {
    display: grid;
    grid-template-columns: minmax(7rem, 0.8fr) minmax(0, 1.5fr) auto;
    gap: 0.35rem 0.75rem;
    padding: 0.8rem 0;
    color: var(--text);
    text-decoration: none;
  }

  .recent-row:hover {
    color: var(--accent);
    text-decoration: none;
  }

  .recent-title {
    min-width: 0;
    overflow: hidden;
    font-weight: 650;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .recent-metrics,
  .recent-date {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  /* Span the full width under the title so the distance · duration · pace
	   trio stays on one line instead of wrapping inside a narrow column. */
  .recent-metrics {
    grid-column: 1 / 3;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-date {
    grid-column: 3;
    grid-row: 1 / span 2;
    align-self: center;
    text-align: right;
  }

  .routes-tile {
    min-height: 21rem;
  }

  .dashboard-map {
    margin-top: 1rem;
  }

  .dashboard-map :global(.map) {
    height: 18rem;
  }

  .domains-tile {
    grid-column: 1 / -1;
  }

  .domains-heading {
    margin-bottom: 0.75rem;
  }

  .domain-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }

  @media (max-width: 1024px) {
    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .bento-grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 640px) {
    .tile-heading {
      flex-direction: column;
    }

    .recent-row {
      grid-template-columns: minmax(0, 1fr) auto;
    }

    .recent-title,
    .recent-metrics {
      grid-column: 1;
    }

    /* On narrow phones prefer wrapping over truncating the run metrics. */
    .recent-metrics {
      white-space: normal;
      overflow: visible;
    }

    .recent-date {
      grid-column: 2;
    }

    .domain-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
