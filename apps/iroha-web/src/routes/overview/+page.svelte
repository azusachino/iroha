<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { IROHA_TIMEZONE } from "$lib/config";
  import { todayInTimezone } from "@iroha/shared/format/date";
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
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RouteFootprint from "$lib/components/RouteFootprint.svelte";
  import GrapherDashboard from "@iroha/shared/theme-ui/grapher/Dashboard.svelte";
  import { createAsyncResource } from "$lib/asyncResource.svelte";

  const RECENT_ACTIVITY_LIMIT = 5;

  const overviewResource = createAsyncResource<{
    summary: Summary;
    recent: Activity[];
    active_days: ActivityActiveDay[];
    current_streak: number;
  }>();
  const routesResource = createAsyncResource<RouteFeatureCollection>();

  // Recent-nights window for the Overview's sleep tile -- same averaging
  // approach as the Night page, just a smaller recent slice since this is a
  // summary tile, not the full history.
  const SLEEP_SWEEP_LIMIT = 30;
  const sleepResource = createAsyncResource<SleepOverview>();
  const mediaResource = createAsyncResource<MediaAggregates>();

  const summary = $derived(overviewResource.data?.summary ?? null);
  const activities = $derived(overviewResource.data?.recent ?? []);
  const activeDays = $derived(overviewResource.data?.active_days ?? []);
  const currentStreak = $derived(overviewResource.data?.current_streak ?? 0);
  const routes = $derived(routesResource.data ?? null);
  const sleepOverview = $derived(sleepResource.data ?? null);
  const mediaAggregates = $derived(mediaResource.data ?? null);

  const heatmapEndDay = todayInTimezone(new Date(), IROHA_TIMEZONE);

  const sleepSummary = $derived({
    averageAsleepS: sleepOverview?.average_asleep_s ?? 0,
    averageEfficiency: sleepOverview?.average_efficiency ?? 0,
    nightCount: sleepOverview?.main_sleep_count ?? 0,
  });
  const streakValue = $derived(
    currentStreak === 1 ? "1 day" : `${currentStreak} days`,
  );

  async function loadActivityOverview() {
    await overviewResource.run(() =>
      getActivityOverview({ recent: RECENT_ACTIVITY_LIMIT }),
    );
  }

  async function loadRoutes() {
    await routesResource.run(() => getActivityRoutes());
  }

  async function loadSleep() {
    // The Overview's sleep tile is supplemental -- a failure here shouldn't
    // block the rest of the dashboard from rendering (sleepResource.error
    // isn't surfaced anywhere on this page).
    await sleepResource.run(() =>
      getSleepOverview({ recent: SLEEP_SWEEP_LIMIT }),
    );
  }

  async function loadMedia() {
    await mediaResource.run(() => getMediaAggregates({}));
  }

  async function reloadDashboard() {
    await Promise.all([
      loadActivityOverview(),
      loadRoutes(),
      loadSleep(),
      loadMedia(),
    ]);
  }

  onMount(() => {
    void loadActivityOverview();
    void loadRoutes();
    void loadSleep();
    void loadMedia();
  });
</script>

<svelte:head>
  <title>iroha · Overview</title>
</svelte:head>

<section class="dashboard-shell">
  <LoadingBoundary
    resource={overviewResource}
    preserveLayout
    label="Loading overview…"
  >
    <GrapherDashboard
      theme="grapher"
      {summary}
      {activeDays}
      {heatmapEndDay}
      {activities}
      {routes}
      streak={streakValue}
      loading={overviewResource.loading}
      error={overviewResource.error}
      {sleepSummary}
      sleepLoading={sleepResource.loading}
      sleepError={sleepResource.error}
      {mediaAggregates}
      mediaLoading={mediaResource.loading}
      mediaError={mediaResource.error}
      onRetry={() => void reloadDashboard()}
      onOpenActivity={(id: string) => void goto(`/motion/${id}`)}
      onOpenSport={(sport: string) =>
        void goto(`/motion?sport=${encodeURIComponent(sport)}`)}
    >
      {#snippet children()}
        <RouteFootprint
          {routes}
          loading={routesResource.loading}
          error={routesResource.error}
          onLoad={() => void loadRoutes()}
        />
      {/snippet}
    </GrapherDashboard>
  </LoadingBoundary>
</section>

<style>
  .dashboard-shell {
    display: grid;
    gap: 1.25rem;
  }
</style>
