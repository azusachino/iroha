<script lang="ts">
  import type { DashboardThemeProps } from "../../view-contracts/dashboard-view";
  import ActivityHeatmap from "../components/ActivityHeatmap.svelte";
  import BarChart from "../components/BarChart.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
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

  function sportShare(activityCount: number): number {
    const total = summary?.totals.activity_count ?? 0;
    return total > 0 ? Math.min(100, (activityCount / total) * 100) : 0;
  }
</script>

<section
  class="grapher-dashboard"
  data-theme={theme}
  aria-labelledby="grapher-dashboard-title"
>
  <header class="dashboard-header">
    <div>
      <p class="kicker">Overview / living archive</p>
      <h1 id="grapher-dashboard-title">Your history, in perspective.</h1>
      <p>
        Movement, rest, routes and the things you collect, organized around the
        records behind them.
      </p>
    </div>
    <div class="streak-readout">
      <span>Current activity streak</span>
      <strong>{streak}</strong>
      <a href="/motion"
        >Open movement archive <span aria-hidden="true">→</span></a
      >
    </div>
  </header>

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

    <dl class="stat-grid">
      <div class="stat-card distance-card">
        <dt>Distance · all time</dt>
        <dd>
          <strong>{formatDistance(summary.totals.distance_m)}</strong>
          <small>Across recorded movement</small>
        </dd>
      </div>
      <div class="stat-card">
        <dt>Activity records</dt>
        <dd>
          <strong>{summary.totals.activity_count.toLocaleString()}</strong>
          <small>Imported movement sessions</small>
        </dd>
      </div>
      <div class="stat-card">
        <dt>Total movement time</dt>
        <dd>
          <strong>
            {formatDuration(
              summary.totals.moving_time_s || summary.totals.duration_s,
            )}
          </strong>
          <small>Recorded duration</small>
        </dd>
      </div>
      <div class="stat-card">
        <dt>Main sleep · recent</dt>
        <dd>
          <strong>
            {sleepLoading || sleepError || sleepSummary.nightCount === 0
              ? "—"
              : formatDuration(sleepSummary.averageAsleepS)}
          </strong>
          <small>
            {#if sleepLoading}
              Loading recent nights
            {:else if sleepError}
              Could not load sleep data
            {:else if sleepSummary.nightCount}
              Average across {sleepSummary.nightCount} main nights
            {:else}
              No main sleep records in this window
            {/if}
          </small>
          <a href="/night">Explore sleep <span aria-hidden="true">→</span></a>
        </dd>
      </div>
      <div class="stat-card">
        <dt>Library items</dt>
        <dd>
          <strong>
            {mediaLoading || mediaError
              ? "—"
              : (mediaAggregates?.totals.item_count ?? 0).toLocaleString()}
          </strong>
          <small>
            {#if mediaLoading}
              Loading the library
            {:else if mediaError}
              Could not load library data
            {:else}
              Collected and tracked items
            {/if}
          </small>
          <a href="/library"
            >Explore library <span aria-hidden="true">→</span></a
          >
        </dd>
      </div>
    </dl>

    <div class="dashboard-grid">
      <ActivityHeatmap
        days={activeDays}
        endDay={heatmapEndDay}
        title="Activity through the year"
      />

      <section class="panel chart-panel" aria-labelledby="distance-trend-title">
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
              formatter: (value) => formatDistance(value * 1000),
            }}
            primaryType="line"
            height={270}
          />
          <p class="chart-note">
            {monthly[0].key}–{monthly[monthly.length - 1].key} · Months without recorded
            distance show 0 km; the current month is partial.
          </p>
        {:else}
          <p class="empty-note">No canonical movement periods yet.</p>
        {/if}
      </section>

      <section
        class="panel sport-panel"
        aria-labelledby="sport-breakdown-title"
      >
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
                    {sport.activity_count.toLocaleString()}
                  </span>
                  <span
                    class="sport-bar"
                    style={`--activity-share: ${sportShare(sport.activity_count)}%`}
                    aria-hidden="true"
                  ></span>
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
      </section>

      <section
        class="panel table-panel"
        aria-labelledby="recent-movement-title"
      >
        <header class="panel-header">
          <div>
            <p class="kicker">Latest records</p>
            <h2 id="recent-movement-title">Recent movement</h2>
          </div>
          <a href="/motion">Browse archive <span aria-hidden="true">→</span></a>
        </header>
        {#if activities.length}
          <div
            class="table-wrap"
            role="region"
            tabindex="0"
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
                    <td>{formatDate(activity.started_at, activity.timezone)}</td
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
          <p class="table-scroll-hint">
            Scroll horizontally to view all columns.
          </p>
        {:else}
          <p class="empty-note">No movement records to show yet.</p>
        {/if}
      </section>

      <section
        class="panel route-panel"
        aria-labelledby="route-footprint-title"
      >
        <header class="panel-header">
          <div>
            <p class="kicker">Geography / privacy-trimmed</p>
            <h2 id="route-footprint-title">Route footprint</h2>
          </div>
          <span>{routes?.features.length ?? "—"} traces</span>
        </header>
        {@render children?.()}
      </section>
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

  h1,
  h2,
  p {
    margin: 0;
  }

  .dashboard-header {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(13rem, 0.34fr);
    align-items: end;
    gap: 2rem;
    padding: 0.75rem 0 1.4rem;
    border-bottom: 1px solid var(--border);
  }

  h1 {
    max-width: 18ch;
    font-family: var(--font-sans);
    font-size: var(--grapher-utility-title-size);
    letter-spacing: -0.055em;
    line-height: 0.98;
  }

  h2 {
    font-family: var(--font-sans);
    font-size: 1.14rem;
    letter-spacing: -0.04em;
  }

  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .dashboard-header > div:first-child > p:last-child {
    max-width: 46rem;
    margin-top: 0.9rem;
    color: var(--text-muted);
    font-family: var(--font-sans);
    line-height: 1.55;
  }

  .streak-readout {
    display: grid;
    gap: 0.45rem;
    padding: 0.75rem 1rem;
    border-left: 3px solid var(--accent);
    background: color-mix(in srgb, var(--accent) 7%, var(--surface));
  }

  .streak-readout > span,
  .stat-card dt {
    color: var(--text-muted);
    font-size: 0.64rem;
    font-weight: 650;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .streak-readout strong {
    color: var(--accent);
    font-size: 2.4rem;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.05em;
    line-height: 1;
  }

  .streak-readout a,
  .panel-header > a,
  .stat-card a {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    width: fit-content;
    min-height: 24px;
    padding: 0 0.15rem;
    font-size: 0.74rem;
    font-weight: 650;
  }

  .stat-grid {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    gap: 0.75rem;
    margin: 0;
  }

  .stat-card {
    display: grid;
    align-content: start;
    gap: 0.4rem;
    min-width: 0;
    padding: 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }

  .distance-card {
    border-top: 3px solid var(--accent);
  }

  .stat-card dd {
    display: grid;
    gap: 0.4rem;
    min-width: 0;
    margin: 0;
  }

  .stat-card strong {
    overflow-wrap: anywhere;
    font-size: clamp(1.05rem, 1.55vw, 1.5rem);
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.04em;
    line-height: 1.1;
  }

  .stat-card small {
    min-height: 2.2em;
    color: var(--text-muted);
    font-size: 0.69rem;
    line-height: 1.4;
  }

  .stat-card a {
    margin-top: 0.2rem;
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

  .dashboard-grid :global(.activity-heatmap) {
    grid-column: 1 / -1;
  }

  .panel {
    min-width: 0;
    padding: 1.15rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }

  .chart-panel {
    grid-column: span 8;
    border-top: 3px solid var(--accent);
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
    font-size: 0.7rem;
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
    font-size: 0.74rem;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .comparison small {
    display: block;
    color: var(--text-muted);
    font-size: 0.64rem;
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
    font-size: 0.7rem;
    font-weight: 700;
    cursor: pointer;
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
    font-size: 0.8rem;
    font-weight: 650;
    text-overflow: ellipsis;
    text-transform: capitalize;
    white-space: nowrap;
  }

  .sport-count {
    color: var(--text-muted);
    font-size: 0.72rem;
    font-variant-numeric: tabular-nums;
  }

  .sport-bar {
    grid-column: 1 / -1;
    display: block;
    height: 0.28rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }

  .sport-bar::after {
    display: block;
    width: var(--activity-share);
    height: 100%;
    border-radius: inherit;
    background: var(--accent);
    content: "";
  }

  .table-wrap {
    overflow-x: auto;
  }

  .table-scroll-hint {
    display: none;
    margin: 0.45rem 0 0;
    color: var(--text-muted);
    font-size: 0.7rem;
  }

  table {
    width: 100%;
    min-width: 34rem;
    border-collapse: collapse;
    font-size: 0.72rem;
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
    font-size: 0.62rem;
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
    font-size: 0.75rem;
    cursor: pointer;
  }

  .status.error {
    color: var(--danger);
  }

  .updating {
    color: var(--text-muted);
    font-size: 0.72rem;
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
    .dashboard-header {
      grid-template-columns: 1fr;
      gap: 1rem;
    }

    .streak-readout {
      max-width: 20rem;
    }

    .chart-panel,
    .sport-panel,
    .table-panel,
    .route-panel {
      grid-column: 1 / -1;
    }
  }

  @media (max-width: 640px) {
    .table-scroll-hint {
      display: block;
    }

    .stat-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 0.5rem;
    }

    .stat-card {
      padding: 0.75rem;
    }

    .stat-card strong {
      font-size: 1.05rem;
    }

    .dashboard-grid {
      gap: 0.75rem;
    }

    .panel {
      padding: 0.9rem;
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
