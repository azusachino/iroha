<script lang="ts">
  import type { DashboardThemeProps } from "../../view-contracts/dashboard-view";
  import ActivityHeatmap from "../components/ActivityHeatmap.svelte";
  import RouteHeader from "../../components/RouteHeader.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import StatTile from "../../components/StatTile.svelte";
  import BarChart from "../components/BarChart.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHumanDuration,
    formatMetricValue,
  } from "../../format/format";
  import { formatMonth } from "../../format/month";
  import { buildMonthlyDistanceSeries } from "./monthly-distance";

  type Period = 6 | 12 | 24;
  const periods: Period[] = [6, 12, 24];

  let {
    summary,
    activeDays,
    heatmapEndDay,
    activities,
    routes,
    streak,
    loading,
    error,
    onRetry,
    onOpenActivity,
    onOpenSport,
    sleepSummary,
    sleepLoading,
    sleepError,
    mediaAggregates,
    mediaLoading,
    mediaError,
    theme,
    children,
  }: DashboardThemeProps = $props();

  let period = $state<Period>(12);
  let recentMovementScroll = $state<HTMLDivElement>();

  function focusRecentMovementTable() {
    recentMovementScroll?.focus({ preventScroll: true });
  }

  const monthly = $derived(
    buildMonthlyDistanceSeries(
      summary?.by_month ?? [],
      heatmapEndDay.slice(0, 7),
      period,
    ),
  );
  const sports = $derived(
    [...(summary?.by_sport ?? [])]
      .sort((left, right) => right.activity_count - left.activity_count)
      .slice(0, 5),
  );
  const comparison = $derived.by(() => {
    if (monthly.length < 2) return null;
    const current = monthly[monthly.length - 1];
    const previous = monthly[monthly.length - 2];
    return { difference: current.distance_m - previous.distance_m, previous };
  });
</script>

<section class="grapher-dashboard" data-theme={theme} aria-label="Overview">
  <RouteHeader
    title="Overview"
    context="Mixed windows · movement and library: all time · sleep and activity: recent records"
  >
    {#snippet actions()}
      <a class="archive-link" href="/motion"
        >Open movement archive <span aria-hidden="true">→</span></a
      >
    {/snippet}
  </RouteHeader>

  {#if loading && !summary}
    <p class="status" role="status">Loading your long view…</p>
  {:else if error}
    <section class="status error" role="alert">
      <p>{error}</p>
      <button type="button" onclick={onRetry}>Retry overview</button>
    </section>
  {:else if !summary}
    <p class="status">No canonical overview is available yet.</p>
  {:else}
    {#if loading}
      <p class="updating" role="status">Updating overview…</p>
    {/if}

    <div class="heatmap-panel">
      <PanelFrame label="Activity calendar">
        <ActivityHeatmap
          days={activeDays}
          endDay={heatmapEndDay}
          title="Activity through the year"
          embedded
        />
      </PanelFrame>
    </div>
    <div class="stat-grid">
      <StatTile
        compact
        label="Distance · all time"
        context="All time"
        value={summary.totals.distance_unknown_count > 0 &&
        summary.totals.distance_known_count === 0
          ? "—"
          : formatDistance(summary.totals.distance_m)}
        sub={summary.totals.distance_unknown_count > 0
          ? `Known distance for ${summary.totals.distance_known_count} of ${summary.totals.activity_count} activities; ${summary.totals.distance_unknown_count} unavailable`
          : "Across recorded movement"}
      />
      <StatTile
        compact
        label="Activity records"
        value={formatMetricValue(summary.totals.activity_count, "count")}
        context="All time"
        sub="Imported movement sessions"
      />
      <StatTile
        compact
        label="Total movement time"
        value={formatHumanDuration(
          summary.totals.moving_time_s || summary.totals.duration_s,
        )}
        context="All time"
        sub="Recorded duration"
      />
      <StatTile
        compact
        label="Main sleep · recent"
        context="Recent records (limit 30)"
        value={sleepLoading || sleepError || sleepSummary.nightCount === 0
          ? "—"
          : formatHumanDuration(sleepSummary.averageAsleepS)}
        sub={sleepLoading
          ? "Loading recent nights"
          : sleepError
            ? "Could not load sleep data"
            : sleepSummary.nightCount
              ? `Average across ${sleepSummary.nightCount} main nights`
              : "No main sleep records in this window"}
      >
        <a href="/night">Explore sleep <span aria-hidden="true">→</span></a>
      </StatTile>
      <StatTile
        compact
        label="Library items"
        context="All time"
        value={mediaLoading || mediaError
          ? "—"
          : formatMetricValue(mediaAggregates?.totals.item_count ?? 0, "count")}
        sub={mediaLoading
          ? "Loading the library"
          : mediaError
            ? "Could not load library data"
            : "Collected and tracked items"}
      >
        <a href="/library">Explore library <span aria-hidden="true">→</span></a>
      </StatTile>
      <StatTile
        compact
        label="Current activity streak"
        value={streak}
        context="Consecutive days ending today"
      />
    </div>

    <div class="dashboard-grid">
      <div class="chart-panel">
        <PanelFrame label="Monthly distance">
          <header class="panel-header">
            <div>
              <p class="kicker">Movement / distance</p>
              <h2 id="distance-trend-title">Monthly distance</h2>
              <p class="panel-caption">
                Each point represents one calendar month.
              </p>
            </div>
            <div class="chart-tools">
              {#if comparison}
                <span class="comparison">
                  {comparison.difference > 0
                    ? "▲"
                    : comparison.difference < 0
                      ? "▼"
                      : "→"}
                  {formatDistance(Math.abs(comparison.difference))}
                  <small>vs {formatMonth(comparison.previous.key)}</small>
                </span>
              {/if}
              <div
                class="period-control"
                role="group"
                aria-label="Distance chart range"
              >
                {#each periods as months (months)}
                  <button
                    type="button"
                    aria-label={`${months}M: Last ${months} months`}
                    aria-pressed={period === months}
                    class:selected={period === months}
                    onclick={() => (period = months)}>{months}M</button
                  >
                {/each}
              </div>
            </div>
          </header>
          {#if monthly.length}
            <BarChart
              categories={monthly.map((bucket) => formatMonth(bucket.key))}
              primary={{
                name: "Distance",
                values: monthly.map((bucket) => bucket.distance_m / 1000),
                color: "var(--accent)",
                axis: { unit: "km" },
                formatter: (value) => formatDistance(value * 1000),
              }}
              primaryType="line"
              height={270}
            />
            <p class="chart-note">
              {monthly[0].key}–{monthly[monthly.length - 1].key} · Months without
              recorded distance show 0 km; the current month is partial.
            </p>
          {:else}
            <p class="empty-note">No canonical movement periods yet.</p>
          {/if}
        </PanelFrame>
      </div>

      <div class="sport-panel">
        <PanelFrame label="By activity">
          <header class="panel-header">
            <div>
              <p class="kicker">Movement / composition</p>
              <h2 id="sport-breakdown-title">By activity</h2>
            </div>
            <a href="/motion">All records <span aria-hidden="true">→</span></a>
          </header>
          {#if sports.length}
            <ul class="sport-list">
              {#each sports as sport (sport.key)}
                <li>
                  <button
                    class="sport-row"
                    type="button"
                    aria-label={`${sport.key.replaceAll("_", " ")}: ${sport.activity_count} activity records`}
                    onclick={() => onOpenSport(sport.key)}
                  >
                    <span class="sport-name"
                      >{sport.key.replaceAll("_", " ")}</span
                    >
                    <span class="sport-count">
                      {formatMetricValue(sport.activity_count, "count")}
                    </span>
                  </button>
                </li>
              {/each}
            </ul>
            <p class="chart-note">
              Select a category to filter the movement archive.
            </p>
          {:else}
            <p class="empty-note">No activity categories to compare yet.</p>
          {/if}
        </PanelFrame>
      </div>

      <div class="route-panel">
        <PanelFrame label="Route footprint">
          <header class="panel-header">
            <div>
              <p class="kicker">Geography / privacy-trimmed</p>
              <h2 id="route-footprint-title">Route footprint</h2>
            </div>
            <span>{routes?.features.length ?? "—"} traces</span>
          </header>
          {@render children?.()}
        </PanelFrame>
      </div>

      <div class="table-panel">
        <PanelFrame label="Recent movement">
          <header class="panel-header">
            <div>
              <p class="kicker">Latest records · up to 5</p>
              <h2 id="recent-movement-title">Recent movement</h2>
            </div>
            <a href="/motion"
              >Browse archive <span aria-hidden="true">→</span></a
            >
          </header>
          {#if activities.length}
            <button
              class="table-scroll-hint"
              type="button"
              aria-controls="recent-movement-table-scroll"
              onclick={focusRecentMovementTable}
            >
              Focus the recent movement table.
            </button>
            <div
              id="recent-movement-table-scroll"
              bind:this={recentMovementScroll}
              class="table-wrap"
              role="region"
              tabindex="-1"
              aria-label="Recent movement table; scroll horizontally to view all columns"
            >
              <table>
                <caption class="visually-hidden">
                  Recent movement records with distance and duration
                </caption>
                <thead>
                  <tr>
                    <th>Date</th>
                    <th>Activity</th>
                    <th>Distance</th>
                    <th>Duration</th>
                  </tr>
                </thead>
                <tbody>
                  {#each activities.slice(0, 8) as activity (activity.id)}
                    <tr>
                      <td
                        >{formatDate(
                          activity.started_at,
                          activity.timezone,
                        )}</td
                      >
                      <td>
                        <button
                          class="activity-link"
                          type="button"
                          onclick={() => onOpenActivity(activity.id)}
                          >{activity.title || activity.sport_type}</button
                        >
                      </td>
                      <td>{formatDistance(activity.distance_m)}</td>
                      <td>
                        {formatDuration(
                          activity.duration_s ?? activity.moving_time_s,
                        )}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {:else}
            <p class="empty-note">No movement records to show yet.</p>
          {/if}
        </PanelFrame>
      </div>
    </div>
  {/if}

  <footer>
    Source: canonical activity summary · every chart keeps its period and unit
    visible.
  </footer>
</section>

<style>
  .grapher-dashboard {
    display: grid;
    gap: 1.1rem;
    min-width: 0;
  }

  .grapher-dashboard > * {
    min-width: 0;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-family: var(--font-sans);
    font-size: var(--type-title-small);
    letter-spacing: -0.04em;
  }

  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: var(--type-caption);
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .archive-link,
  .panel-header > a {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    width: fit-content;
    min-height: 24px;
    padding: 0 0.15rem;
    font-size: var(--type-label);
    font-weight: 650;
  }

  .stat-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 0.75rem;
    margin: 0;
  }

  .stat-grid > :global(.stat-tile:nth-child(1)) {
    --accent: var(--sport-walk, #00e5bf);
  }

  .stat-grid > :global(.stat-tile:nth-child(2)) {
    --accent: var(--mark-amber, #f59e0b);
  }

  .stat-grid > :global(.stat-tile:nth-child(3)) {
    --accent: var(--sport-run, #38bdf8);
  }

  .stat-grid > :global(.stat-tile:nth-child(4)) {
    --accent: var(--mark-violet, #818cf8);
  }

  .stat-grid > :global(.stat-tile:nth-child(5)) {
    --accent: var(--category-subscriptions, #ff5c8a);
  }

  .stat-grid > :global(.stat-tile:nth-child(6)) {
    --accent: var(--ring-move, #f07c78);
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    align-items: start;
    gap: 1rem;
  }

  .dashboard-grid > * {
    min-width: 0;
  }

  .heatmap-panel {
    grid-column: 1 / -1;
  }

  .chart-panel {
    grid-column: span 8;
  }

  .sport-panel {
    grid-column: span 4;
  }

  .table-panel {
    grid-column: span 7;
  }

  .route-panel {
    grid-column: span 5;
  }

  .panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.8rem;
    margin-bottom: 0.9rem;
  }

  .panel-caption,
  .chart-note,
  .empty-note,
  footer {
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  .panel-caption {
    margin-top: 0.28rem;
  }

  .chart-tools {
    display: grid;
    justify-items: end;
    gap: 0.5rem;
  }

  .comparison {
    color: var(--accent);
    font-size: var(--type-label);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .comparison small {
    display: block;
    color: var(--text-muted);
    font-size: var(--type-caption);
    text-align: right;
  }

  .period-control {
    display: inline-flex;
    gap: 0.1rem;
    padding: 0.18rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface-2);
  }

  .period-control button {
    min-width: 2.45rem;
    min-height: 2.1rem;
    padding: 0.25rem 0.45rem;
    border: 0;
    border-radius: 999px;
    background: transparent;
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    cursor: pointer;
    transition:
      background-color var(--motion-micro),
      color var(--motion-micro),
      transform var(--motion-micro);
  }

  .period-control button:hover {
    background: color-mix(in srgb, var(--accent) 14%, var(--surface-2));
    color: var(--text);
  }

  .period-control button:active {
    transform: scale(0.96);
  }

  .period-control button.selected {
    background: var(--accent);
    color: var(--bg);
  }

  .chart-note {
    border-top: 1px solid var(--border);
    padding-top: 0.6rem;
  }

  .sport-list {
    display: grid;
    gap: 0.2rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .sport-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.35rem 0.7rem;
    width: 100%;
    padding: 0.65rem 0.2rem;
    border: 0;
    border-bottom: 1px solid var(--border);
    background: transparent;
    color: var(--text);
    text-align: left;
    cursor: pointer;
  }

  .sport-row:hover .sport-name {
    color: var(--accent);
  }

  .sport-name {
    overflow: hidden;
    font-size: var(--type-label);
    font-weight: 650;
    text-overflow: ellipsis;
    text-transform: capitalize;
    white-space: nowrap;
  }

  .sport-count {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-variant-numeric: tabular-nums;
  }

  .table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .table-wrap tbody tr {
    transition: background-color var(--motion-micro);
  }

  .table-wrap tbody tr:hover {
    background: color-mix(in srgb, var(--accent) 8%, var(--surface));
  }

  .table-scroll-hint {
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    margin: 0 0 0.35rem;
    padding: 0 0.25rem;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: var(--type-caption);
    text-align: left;
    text-decoration: underline;
    text-underline-offset: 0.15em;
    cursor: pointer;
  }

  .table-scroll-hint:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  table {
    width: 100%;
    min-width: 34rem;
    border-collapse: collapse;
    font-size: var(--type-caption);
  }

  th,
  td {
    padding: 0.68rem 0.35rem;
    border-top: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }

  th {
    color: var(--text-muted);
    font-size: var(--type-caption);
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .activity-link {
    min-height: var(--control-target-min);
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-weight: 700;
    text-align: left;
    cursor: pointer;
  }

  .activity-link:hover {
    color: var(--accent);
  }

  .route-panel :global(.map) {
    height: 20rem;
  }

  .status {
    display: grid;
    gap: 0.75rem;
    padding: 1.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text-muted);
  }

  .status button {
    width: fit-content;
    min-height: 2.5rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: var(--type-caption);
    cursor: pointer;
  }

  .status.error {
    color: var(--danger);
  }

  .updating {
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  footer {
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
  }

  @media (max-width: 1024px) {
    .stat-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 768px) {
    .chart-panel,
    .sport-panel,
    .table-panel,
    .route-panel {
      grid-column: 1 / -1;
    }
  }

  @media (max-width: 640px) {
    .stat-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 0.5rem;
    }

    .dashboard-grid {
      gap: 0.75rem;
    }

    .panel-header {
      flex-direction: column;
    }

    .chart-tools {
      grid-template-columns: 1fr auto;
      align-items: center;
      justify-items: start;
      width: 100%;
    }

    .comparison {
      grid-column: 1;
    }

    .period-control {
      grid-column: 2;
      grid-row: 1;
    }

    .route-panel :global(.map) {
      height: 16rem;
    }
  }
</style>
