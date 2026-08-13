<script lang="ts">
  import type { Activity, Lap, RoutePoint, SamplingPoint } from "$lib/api";
  import SourceBadge from "@iroha/shared/SourceBadge.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSwimmingPace,
  } from "$lib/format";
  import { isSwimming, sportLabel } from "$lib/sport";
  import LapChart from "$lib/components/LapChart.svelte";

  let {
    activity,
    derivedDistanceM,
    route,
    samplings,
    laps,
    selectedRouteIndex,
    onSelectRoute,
  }: {
    activity: Activity;
    derivedDistanceM?: number;
    route: RoutePoint[];
    samplings: SamplingPoint[];
    laps: Lap[];
    selectedRouteIndex: number | null;
    onSelectRoute: (index: number | null) => void;
  } = $props();

  const heartRateSamples = $derived(
    samplings.filter((sample) =>
      /heart|(^|_)hr($|_)/i.test(sample.sampling_type),
    ),
  );
  const tracePoints = $derived(
    route.filter(
      (point) => Number.isFinite(point.lat) && Number.isFinite(point.lon),
    ),
  );
  const swimming = $derived(isSwimming(activity.sport_type));
  const distanceM = $derived(activity.distance_m ?? derivedDistanceM);
</script>

<article class="journal-entry-detail">
  <header class="entry-hero">
    <div>
      <p class="journal-kicker">{sportLabel(activity.sport_type)} · entry</p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="entry-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="entry-stamp" aria-hidden="true">
      <span>{String(tracePoints.length).padStart(2, "0")}</span>
      <small>trace<br />points</small>
    </div>
  </header>

  <div class="journal-rule"><span>the numbers</span></div>

  <dl class="entry-metrics">
    <div>
      <dt>{swimming ? "GPS distance" : "Distance"}</dt>
      <dd>{formatDistance(distanceM)}</dd>
    </div>
    <div>
      <dt>Duration</dt>
      <dd>{formatDuration(activity.duration_s)}</dd>
    </div>
    <div>
      <dt>{swimming ? "Pace / 100m" : "Avg pace"}</dt>
      <dd>
        {swimming
          ? formatSwimmingPace(distanceM, activity.duration_s)
          : formatPace(activity.avg_pace_s_per_km)}
      </dd>
    </div>
    <div>
      <dt>Avg heart rate</dt>
      <dd>{formatHr(activity.avg_hr)}</dd>
    </div>
    <div>
      <dt>Elevation</dt>
      <dd>{formatDistance(activity.elevation_gain_m)}</dd>
    </div>
    <div>
      <dt>Samples</dt>
      <dd>{heartRateSamples.length || "—"}</dd>
    </div>
  </dl>

  <div class="entry-grid">
    <section class="trace-card">
      <div class="panel-heading">
        <div>
          <p class="journal-kicker">Evidence stream</p>
          <h2>{swimming ? "Open-water trace" : "Route trace"}</h2>
        </div>
        <span
          >{selectedRouteIndex == null
            ? "Move across the record"
            : `Point ${selectedRouteIndex + 1}`}</span
        >
      </div>
      <div class="trace-field" aria-label="Route trace points">
        {#if tracePoints.length}
          {#each tracePoints.slice(0, 160) as point, index (point.seq)}
            <button
              class:active={selectedRouteIndex === index}
              style={`--x: ${(index / Math.max(tracePoints.length - 1, 1)) * 100}%; --y: ${50 + Math.sin(index / 4) * 28}%;`}
              aria-label={`Select route point ${index + 1}`}
              onclick={() =>
                onSelectRoute(selectedRouteIndex === index ? null : index)}
            ></button>
          {/each}
        {:else}
          <p>No route geometry was recorded for this session.</p>
        {/if}
      </div>
      <p class="panel-note">
        {swimming
          ? "GPS fixes show the open-water path. No pool intervals are inferred from this record."
          : "The presentation keeps the source trace visible without hiding the underlying record."}
      </p>
    </section>
    <aside class="notes-card">
      <p class="journal-kicker">Session notes</p>
      <h2>What remains</h2>
      <dl class="notes-list">
        <div>
          <dt>Moving time</dt>
          <dd>{formatDuration(activity.moving_time_s)}</dd>
        </div>
        <div>
          <dt>Maximum heart rate</dt>
          <dd>{formatHr(activity.max_hr)}</dd>
        </div>
        <div>
          <dt>Source</dt>
          <dd><SourceBadge source={activity.source_kind} /></dd>
        </div>
        <div>
          <dt>Laps available</dt>
          <dd>{laps.length || "None"}</dd>
        </div>
      </dl>
    </aside>
  </div>

  {#if laps.length}
    <section class="laps-ledger">
      <div class="panel-heading">
        <div>
          <p class="journal-kicker">Intervals</p>
          <h2>Splits in the source</h2>
        </div>
        <span>{laps.length} laps</span>
      </div>
      <LapChart {laps} {swimming} />
      <div class="ledger-scroll">
        <table>
          <thead>
            <tr>
              <th>Lap</th>
              <th>Distance</th>
              <th>Duration</th>
              <th>Pace</th>
            </tr>
          </thead>
          <tbody>
            {#each laps.slice(0, 12) as lap (lap.id)}
              <tr>
                <td>{String(lap.lap_no).padStart(2, "0")}</td>
                <td>{formatDistance(lap.distance_m)}</td>
                <td>{formatDuration(lap.duration_s)}</td>
                <td>{formatPace(lap.avg_pace_s_per_km)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>
  {/if}

  <footer class="journal-source">
    <span>Source: imported activity record</span>
    <span>Presentation only · no readiness score inferred</span>
  </footer>
</article>

<style>
  .journal-entry-detail {
    display: grid;
    gap: 1.5rem;
    min-width: 0;
    max-width: 100%;
  }
  .journal-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    max-width: 15ch;
    font-size: clamp(2.3rem, 6vw, 4.4rem);
    line-height: 0.94;
  }
  h2 {
    font-size: 1.4rem;
  }
  .entry-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: start;
  }
  .entry-date {
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.95rem;
  }
  .entry-stamp {
    display: grid;
    width: 5.5rem;
    aspect-ratio: 1;
    place-items: center;
    align-content: center;
    border: 1px solid var(--accent);
    border-radius: 50%;
    color: var(--accent);
    text-align: center;
    transform: rotate(-6deg);
  }
  .entry-stamp span {
    font-family: var(--font-serif);
    font-size: 1.3rem;
  }
  .entry-stamp small {
    margin-top: 0.2rem;
    font-size: 0.5rem;
    letter-spacing: 0.1em;
    line-height: 1.1;
    text-transform: uppercase;
  }
  .journal-rule {
    display: flex;
    align-items: center;
    gap: 1rem;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.8rem;
    font-style: italic;
  }
  .journal-rule::after {
    content: "";
    height: 1px;
    flex: 1;
    background: var(--border);
  }
  .entry-metrics {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    min-width: 0;
    margin: 0;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .entry-metrics div {
    display: grid;
    gap: 0.35rem;
    padding: 0.9rem;
    border-right: 1px solid var(--border);
    min-width: 0;
  }
  .entry-metrics div:last-child {
    border-right: 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-serif);
    font-size: 1rem;
  }
  .entry-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 0.85fr);
    gap: 1.25rem;
    min-width: 0;
  }
  .trace-card,
  .notes-card,
  .laps-ledger {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
    padding: 1.25rem;
    min-width: 0;
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .trace-field {
    position: relative;
    min-height: 15rem;
    margin-top: 1.25rem;
    overflow: hidden;
    border: 1px solid var(--border);
    background:
      repeating-linear-gradient(
        0deg,
        transparent 0 2.1rem,
        color-mix(in srgb, var(--accent) 14%, transparent) 2.1rem 2.16rem
      ),
      repeating-linear-gradient(
        90deg,
        transparent 0 3.2rem,
        color-mix(in srgb, var(--accent) 12%, transparent) 3.2rem 3.26rem
      );
  }
  .trace-field button {
    position: absolute;
    left: var(--x);
    top: var(--y);
    width: 0.35rem;
    height: 0.35rem;
    padding: 0;
    transform: translate(-50%, -50%);
    border: 0;
    border-radius: 50%;
    background: var(--accent);
    opacity: 0.55;
  }
  .trace-field button.active {
    width: 0.8rem;
    height: 0.8rem;
    opacity: 1;
    box-shadow: 0 0 0 0.3rem color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .trace-field p {
    padding: 1rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .panel-note {
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.6;
  }
  .notes-list {
    display: grid;
    gap: 0;
    margin: 1.25rem 0 0;
  }
  .notes-list div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-bottom: 1px dotted var(--border);
    padding: 0.7rem 0;
  }
  .notes-list dd {
    text-align: right;
  }
  .ledger-scroll {
    overflow-x: auto;
    margin-top: 1.25rem;
  }
  table {
    width: 100%;
    min-width: 26rem;
    border-collapse: collapse;
    font-size: 0.8rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.7rem 0.5rem;
    text-align: left;
  }
  td:first-child {
    color: var(--accent);
    font-family: var(--font-serif);
  }
  .journal-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 800px) {
    .entry-metrics {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .entry-metrics div:nth-child(3) {
      border-right: 0;
    }
    .entry-grid {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 480px) {
    .entry-hero {
      align-items: start;
    }
    .entry-stamp {
      display: none;
    }
    .entry-metrics {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .entry-metrics div:nth-child(3) {
      border-right: 1px solid var(--border);
    }
    .journal-source {
      display: block;
    }
  }
</style>
