<script lang="ts">
  import type { Snippet } from "svelte";
  import type {
    Activity,
    Lap,
    RoutePoint,
    SamplingPoint,
  } from "../../activity";
  import SourceBadge from "../../SourceBadge.svelte";
  import LapChart from "../components/LapChart.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSwimmingPace,
  } from "../../format";
  import { isSwimming, sportLabel } from "../../sport";

  let {
    activity,
    derivedDistanceM,
    route,
    samplings,
    laps,
    children,
  }: {
    activity: Activity;
    derivedDistanceM?: number;
    route: RoutePoint[];
    samplings: SamplingPoint[];
    laps: Lap[];
    selectedRouteIndex: number | null;
    onSelectRoute: (index: number | null) => void;
    children?: Snippet;
  } = $props();

  const swimming = $derived(isSwimming(activity.sport_type));
  const distance = $derived(activity.distance_m ?? derivedDistanceM);
</script>

<article class="grapher-detail" aria-labelledby="grapher-detail-title">
  <header class="detail-header">
    <div>
      <p class="kicker">Motion / record detail</p>
      <h1 id="grapher-detail-title">
        {activity.title || sportLabel(activity.sport_type)}
      </h1>
      <p>
        {formatDate(activity.started_at, activity.timezone)} · {sportLabel(
          activity.sport_type,
        )}
      </p>
    </div>
    <SourceBadge source={activity.source_kind} />
  </header>
  <div class="metrics">
    <div><span>Distance</span><strong>{formatDistance(distance)}</strong></div>
    <div>
      <span>Duration</span><strong>{formatDuration(activity.duration_s)}</strong
      >
    </div>
    <div>
      <span>{swimming ? "Pace / 100m" : "Average pace"}</span><strong
        >{swimming
          ? formatSwimmingPace(distance, activity.duration_s)
          : formatPace(activity.avg_pace_s_per_km)}</strong
      >
    </div>
    <div>
      <span>Average HR</span><strong>{formatHr(activity.avg_hr)}</strong>
    </div>
    <div>
      <span>Elevation</span><strong
        >{formatDistance(activity.elevation_gain_m)}</strong
      >
    </div>
  </div>
  {@render children?.()}
  {#if laps.length}<section class="chart-panel">
      <header>
        <p class="kicker">Measured intervals</p>
        <h2>Lap chart</h2>
      </header>
      <LapChart {laps} {swimming} />
      <div class="lap-table">
        <table>
          <thead
            ><tr><th>Lap</th><th>Distance</th><th>Duration</th><th>Pace</th></tr
            ></thead
          ><tbody
            >{#each laps as lap (lap.id)}<tr
                ><td>{lap.lap_no}</td><td>{formatDistance(lap.distance_m)}</td
                ><td>{formatDuration(lap.duration_s)}</td><td
                  >{formatPace(lap.avg_pace_s_per_km)}</td
                ></tr
              >{/each}</tbody
          >
        </table>
      </div>
    </section>{/if}
  <section class="evidence-grid">
    <div>
      <p class="kicker">Source measurements</p>
      <h2>Exact record</h2>
      <dl>
        <div>
          <dt>Samples</dt>
          <dd>{samplings.length || "—"}</dd>
        </div>
        <div>
          <dt>Moving time</dt>
          <dd>{formatDuration(activity.moving_time_s)}</dd>
        </div>
        <div>
          <dt>Maximum HR</dt>
          <dd>{formatHr(activity.max_hr)}</dd>
        </div>
        <div>
          <dt>Route points</dt>
          <dd>{route.length || "—"}</dd>
        </div>
      </dl>
    </div>
  </section>
</article>

<style>
  .grapher-detail {
    display: grid;
    gap: 1rem;
    font-family: var(--font-mono);
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    max-width: 16ch;
    font-family: var(--font-sans);
    font-size: clamp(2.8rem, 8vw, 7rem);
    letter-spacing: -0.12em;
    line-height: 0.82;
  }
  h2 {
    font-family: var(--font-sans);
    font-size: 1.2rem;
  }
  .kicker {
    margin-bottom: 0.45rem;
    color: var(--accent);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .detail-header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
    border-bottom: 3px solid var(--text);
    padding-bottom: 1.5rem;
  }
  .detail-header p:last-child {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .metrics {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    border-block: 1px solid var(--border);
  }
  .metrics div {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.85rem;
    border-right: 1px solid var(--border);
  }
  .metrics div:last-child {
    border: 0;
  }
  .metrics span,
  dt {
    color: var(--text-muted);
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .metrics strong {
    font-family: var(--font-sans);
    font-size: 1.1rem;
    letter-spacing: -0.05em;
  }
  .chart-panel,
  .evidence-grid > div {
    border: 1px solid var(--border);
    background: var(--surface);
    padding: 1rem;
  }
  .chart-panel {
    border-top: 4px solid var(--accent);
  }
  .chart-panel > header {
    margin-bottom: 0.5rem;
  }
  .lap-table {
    overflow-x: auto;
    border-top: 1px solid var(--border);
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.72rem;
  }
  th,
  td {
    padding: 0.65rem 0.35rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.62rem;
    text-transform: uppercase;
  }
  .evidence-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  dl {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.75rem;
    margin: 1rem 0 0;
  }
  dd {
    margin: 0.25rem 0 0;
    font-size: 0.85rem;
  }
  @media (max-width: 768px) {
    .detail-header,
    .evidence-grid {
      display: grid;
    }
    .metrics {
      grid-template-columns: repeat(2, 1fr);
    }
    .metrics div:nth-child(even) {
      border-right: 0;
    }
    .metrics div:nth-child(n + 3) {
      border-top: 1px solid var(--border);
    }
  }
</style>
