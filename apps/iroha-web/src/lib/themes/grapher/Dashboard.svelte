<script lang="ts">
  import type {
    Activity,
    MediaAggregates,
    RouteFeatureCollection,
    Summary,
  } from "$lib/api";
  import BarChart from "@iroha/shared/theme-ui/components/BarChart.svelte";
  import RouteFootprint from "$lib/components/RouteFootprint.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatMonth,
  } from "$lib/format";

  let {
    summary,
    activities,
    routes,
    streak,
    loading,
    error,
    routesLoading,
    routesError,
    onLoadRoutes,
    sleepSummary,
    mediaAggregates,
  }: {
    summary: Summary | null;
    activities: Activity[];
    routes: RouteFeatureCollection | null;
    streak: string;
    loading: boolean;
    error: string | null;
    routesLoading: boolean;
    routesError: string | null;
    onLoadRoutes: () => void;
    sleepSummary: {
      averageAsleepS: number;
      averageEfficiency: number;
      nightCount: number;
    };
    mediaAggregates: MediaAggregates | null;
  } = $props();

  const monthly = $derived(
    [...(summary?.by_month ?? [])]
      .sort((left, right) => left.key.localeCompare(right.key))
      .slice(-24),
  );
  const incline = $derived.by(() => {
    if (monthly.length < 2) return null;
    const current = monthly[monthly.length - 1].distance_m;
    const previous = monthly[monthly.length - 2].distance_m;
    return {
      diff: current - previous,
      previous: monthly[monthly.length - 2].key,
    };
  });
</script>

<section class="grapher-dashboard" aria-labelledby="grapher-dashboard-title">
  <header class="dashboard-header">
    <div>
      <p class="kicker">Overview / comparative record</p>
      <h1 id="grapher-dashboard-title">Your data, plotted.</h1>
      <p>
        A chart-led index of movement, rest, routes, and the records around
        them. Every comparison keeps its unit and period explicit.
      </p>
    </div>
    <div class="readout"><strong>{streak}</strong><span>day streak</span></div>
  </header>

  {#if loading}
    <p class="status">Loading the canonical overview…</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else}
    <section class="chart-panel" aria-labelledby="distance-trend-title">
      <header class="panel-header">
        <div>
          <p class="kicker">Primary comparison</p>
          <h2 id="distance-trend-title">Monthly distance</h2>
        </div>
        {#if incline}
          <span class:down={incline.diff < 0} class="incline">
            {incline.diff >= 0 ? "▲" : "▼"}
            {formatDistance(Math.abs(incline.diff))}
            <small>vs {formatMonth(incline.previous)}</small>
          </span>
        {/if}
      </header>
      {#if monthly.length}
        <BarChart
          categories={monthly.map((bucket) => formatMonth(bucket.key))}
          primary={{
            name: "Distance",
            values: monthly.map((bucket) => bucket.distance_m / 1000),
            color: "var(--accent)",
            formatter: (value) => formatDistance(value * 1000),
          }}
          primaryType="line"
          height={300}
        />
        <p class="chart-note">
          Server summary · {monthly[0].key}–{monthly[monthly.length - 1].key} · gaps
          mean no recorded movement.
        </p>
      {:else}
        <p class="status">No canonical movement periods yet.</p>
      {/if}
    </section>

    <div class="stat-grid" aria-label="Canonical overview totals">
      <div>
        <span>Distance</span><strong
          >{formatDistance(summary?.totals.distance_m)}</strong
        >
      </div>
      <div>
        <span>Activities</span><strong
          >{summary?.totals.activity_count ?? "—"}</strong
        >
      </div>
      <div>
        <span>Total time</span><strong
          >{formatDuration(
            summary?.totals.moving_time_s || summary?.totals.duration_s,
          )}</strong
        >
      </div>
      <div>
        <span>Main sleep</span><strong
          >{sleepSummary.nightCount
            ? formatDuration(sleepSummary.averageAsleepS)
            : "—"}</strong
        >
      </div>
      <div>
        <span>Library</span><strong
          >{mediaAggregates?.totals.item_count ?? "—"}</strong
        >
      </div>
    </div>

    <div class="dashboard-grid">
      <section class="table-panel">
        <header class="panel-header">
          <div>
            <p class="kicker">Latest rows</p>
            <h2>Recent movement</h2>
          </div>
          <span>{Math.min(8, activities.length)} shown</span>
        </header>
        <div class="table-wrap">
          <table>
            <thead
              ><tr
                ><th>Date</th><th>Activity</th><th>Distance</th><th>Duration</th
                ></tr
              ></thead
            >
            <tbody>
              {#each activities.slice(0, 8) as activity (activity.id)}
                <tr>
                  <td>{formatDate(activity.started_at, activity.timezone)}</td>
                  <td
                    ><a href={`/motion/${activity.id}`}
                      >{activity.title || activity.sport_type}</a
                    ></td
                  >
                  <td>{formatDistance(activity.distance_m)}</td>
                  <td
                    >{formatDuration(
                      activity.duration_s ?? activity.moving_time_s,
                    )}</td
                  >
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
      <section class="route-panel">
        <header class="panel-header">
          <div>
            <p class="kicker">Geography</p>
            <h2>Route footprint</h2>
          </div>
          <span>{routes?.features.length ?? "—"} traces</span>
        </header>
        <RouteFootprint
          {routes}
          loading={routesLoading}
          error={routesError}
          onLoad={onLoadRoutes}
        />
      </section>
    </div>
  {/if}
  <footer>
    Source: canonical activity summary · comparison is month-over-month and
    unit-preserving.
  </footer>
</section>

<style>
  .grapher-dashboard {
    display: grid;
    gap: 1rem;
    font-family: var(--font-mono);
  }
  .dashboard-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 3px solid var(--text);
    padding-bottom: 1.5rem;
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    max-width: 11ch;
    font-family: var(--font-sans);
    font-size: clamp(3rem, 8vw, 7rem);
    letter-spacing: -0.12em;
    line-height: 0.82;
  }
  h2 {
    font-family: var(--font-sans);
    font-size: 1.15rem;
    letter-spacing: -0.04em;
  }
  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .dashboard-header > div:first-child > p:last-child {
    max-width: 42rem;
    margin-top: 1rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
    line-height: 1.55;
  }
  .readout {
    display: grid;
    justify-items: end;
    color: var(--accent);
    white-space: nowrap;
  }
  .readout strong {
    font-family: var(--font-sans);
    font-size: 3rem;
    letter-spacing: -0.1em;
  }
  .readout span,
  .panel-header > span,
  footer,
  .chart-note {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .chart-panel,
  .table-panel,
  .route-panel {
    border: 1px solid var(--border);
    background: var(--surface);
    padding: 1rem;
  }
  .chart-panel {
    border-top: 4px solid var(--accent);
  }
  .panel-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    margin-bottom: 0.75rem;
  }
  .incline {
    color: var(--accent);
    font-size: 0.78rem;
    white-space: nowrap;
  }
  .incline.down {
    color: var(--mark-coral, #d96b5f);
  }
  .incline small {
    display: block;
    color: var(--text-muted);
    font-size: 0.62rem;
    text-align: right;
  }
  .chart-note {
    border-top: 1px solid var(--border);
    padding-top: 0.6rem;
  }
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    border-block: 1px solid var(--border);
  }
  .stat-grid div {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.85rem;
    border-right: 1px solid var(--border);
  }
  .stat-grid div:last-child {
    border-right: 0;
  }
  .stat-grid span {
    color: var(--text-muted);
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .stat-grid strong {
    font-family: var(--font-sans);
    font-size: 1.35rem;
    letter-spacing: -0.06em;
  }
  .dashboard-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(18rem, 0.85fr);
    gap: 1rem;
  }
  .table-wrap {
    overflow-x: auto;
  }
  table {
    width: 100%;
    min-width: 34rem;
    border-collapse: collapse;
    font-size: 0.72rem;
  }
  th,
  td {
    padding: 0.7rem 0.35rem;
    border-top: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.08em;
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
  .status {
    border: 1px dashed var(--border);
    padding: 1.5rem;
    color: var(--text-muted);
  }
  .status.error {
    color: var(--error, #d96b5f);
  }
  footer {
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
  }
  @media (max-width: 760px) {
    .dashboard-header,
    .dashboard-grid {
      grid-template-columns: 1fr;
      display: grid;
    }
    .readout {
      justify-items: start;
    }
    .stat-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .stat-grid div:nth-child(even) {
      border-right: 0;
    }
    .stat-grid div:nth-child(n + 3) {
      border-top: 1px solid var(--border);
    }
  }
</style>
