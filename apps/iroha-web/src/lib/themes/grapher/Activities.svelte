<script lang="ts">
  import type { Activity, MetricSeriesResponse } from "$lib/api";
  import ActivityMetricChart from "$lib/components/ActivityMetricChart.svelte";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "$lib/format";
  import { sportLabel } from "$lib/sport";

  type DisplaySummary = {
    activity_count: number;
    distance_m: number;
    duration_s: number;
  };
  let {
    activities,
    displaySummary,
    sportType,
    sportOptions,
    loading,
    error,
    hasMore,
    loadingMore,
    onSportType,
    onLoadMore,
    activitySeries = null,
    activityDurationSeries = null,
    activitySeriesLoading = false,
    activitySeriesError = null,
    activitySeriesScope = "",
  }: {
    activities: Activity[];
    displaySummary: DisplaySummary;
    sportType: string;
    sportOptions: string[];
    loading: boolean;
    error: string | null;
    hasMore: boolean;
    loadingMore: boolean;
    onSportType: (value: string) => void;
    onLoadMore: () => void;
    activitySeries?: MetricSeriesResponse | null;
    activityDurationSeries?: MetricSeriesResponse | null;
    activitySeriesLoading?: boolean;
    activitySeriesError?: string | null;
    activitySeriesScope?: string;
  } = $props();
</script>

<section class="grapher-activities" aria-labelledby="activity-data-title">
  <header class="page-intro">
    <p class="kicker">Activity data / public-style table</p>
    <h1 id="activity-data-title">The movement record.</h1>
    <p>
      Filter the imported sessions, then compare the same fields row by row.
    </p>
  </header>

  <div class="filters" aria-label="Activity filters">
    <label
      >Sport<select
        value={sportType}
        onchange={(event) =>
          onSportType((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All sports</option
        >{#each sportOptions as sport}<option value={sport}
            >{sportLabel(sport)}</option
          >{/each}</select
      ></label
    >
  </div>

  <ActivityMetricChart
    series={activitySeries}
    durationSeries={activityDurationSeries}
    loading={activitySeriesLoading}
    error={activitySeriesError}
    scope={activitySeriesScope}
  />

  <div class="summary-row" aria-label="Filtered activity summary">
    <div>
      <span>Sessions</span><strong
        >{displaySummary.activity_count.toLocaleString()}</strong
      >
    </div>
    <div>
      <span>Distance</span><strong
        >{formatDistance(displaySummary.distance_m)}</strong
      >
    </div>
    <div>
      <span>Moving time</span><strong
        >{formatDuration(displaySummary.duration_s)}</strong
      >
    </div>
  </div>

  {#if loading}
    <p class="muted">Loading activity data…</p>
  {:else if error}
    <p class="error">Could not load activity data: {error}</p>
  {:else if activities.length === 0}
    <p class="muted">No activity sessions match this selection.</p>
  {:else}
    <div class="table-frame">
      <table>
        <thead
          ><tr
            ><th>Date</th><th>Activity</th><th>Distance</th><th>Duration</th><th
              >Pace</th
            ></tr
          ></thead
        >
        <tbody>
          {#each activities as activity (activity.id)}
            <tr
              class="activity-row"
              ondblclick={() =>
                (window.location.href = `/motion/${activity.id}`)}
            >
              <td>{formatDateOnly(activity.started_at, activity.timezone)}</td>
              <td
                ><a href={`/motion/${activity.id}`}
                  >{activity.title || sportLabel(activity.sport_type)}</a
                ><small>{sportLabel(activity.sport_type)}</small></td
              >
              <td>{formatDistance(activity.distance_m)}</td>
              <td
                >{formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</td
              >
              <td>{formatPace(activity.avg_pace_s_per_km)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if hasMore}<button
        class="load-more"
        onclick={onLoadMore}
        disabled={loadingMore}
        >{loadingMore ? "Loading…" : "Load more rows"}</button
      >{/if}
  {/if}
</section>

<style>
  .grapher-activities {
    display: grid;
    gap: 1rem;
  }
  .page-intro {
    max-width: 48rem;
    padding-bottom: 2rem;
    border-bottom: 3px solid var(--text);
  }
  .kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1 {
    margin: 0;
    font-size: clamp(2.8rem, 7vw, 6.5rem);
    letter-spacing: -0.1em;
    line-height: 0.88;
  }
  .page-intro p:last-child {
    margin: 1rem 0 0;
    color: var(--text-muted);
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
  }
  .filters label {
    display: grid;
    gap: 0.35rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    text-transform: uppercase;
  }
  .filters select {
    min-width: 9rem;
    border-radius: 0;
    font-size: 0.78rem;
  }
  .summary-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border-block: 1px solid var(--border);
  }
  .summary-row div {
    display: grid;
    gap: 0.4rem;
    padding: 1rem;
    border-right: 1px solid var(--border);
  }
  .summary-row div:last-child {
    border: 0;
  }
  .summary-row span {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .summary-row strong {
    font-size: clamp(1.35rem, 3vw, 2.4rem);
    letter-spacing: -0.08em;
  }
  .table-frame {
    overflow-x: auto;
    border-top: 2px solid var(--text);
  }
  table {
    width: 100%;
    min-width: 44rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th,
  td {
    padding: 0.75rem 0.45rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  td a {
    color: var(--text);
    font-weight: 700;
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
  }
  td small {
    display: block;
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .load-more {
    padding: 0.7rem 1rem;
    border: 1px solid var(--border);
    border-radius: 0;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }
  .muted {
    color: var(--text-muted);
  }
  .error {
    color: var(--danger);
  }
  @media (max-width: 600px) {
    .summary-row strong {
      font-size: 1.2rem;
    }
    .summary-row div {
      padding: 0.75rem 0.4rem;
    }
  }
</style>
