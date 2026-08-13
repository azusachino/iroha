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

  export type ActivityDetailVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  let {
    variant,
    activity,
    derivedDistanceM,
    route,
    samplings,
    laps,
    selectedRouteIndex,
    onSelectRoute,
  }: {
    variant: ActivityDetailVariant;
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

<article class={`theme-activity-detail theme-activity-detail-${variant}`}>
  <header class="detail-hero">
    <div>
      <p class="theme-kicker">
        {sportLabel(activity.sport_type)} / field record
      </p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="detail-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="hero-mark" aria-hidden="true">
      <span>{String(tracePoints.length).padStart(2, "0")}</span><small
        >trace points</small
      >
    </div>
  </header>

  <div class="metric-grid">
    <div>
      <span>{swimming ? "GPS distance" : "Distance"}</span><strong
        >{formatDistance(distanceM)}</strong
      >
    </div>
    <div>
      <span>Duration</span><strong>{formatDuration(activity.duration_s)}</strong
      >
    </div>
    <div>
      <span>{swimming ? "Pace / 100m" : "Avg pace"}</span><strong
        >{swimming
          ? formatSwimmingPace(distanceM, activity.duration_s)
          : formatPace(activity.avg_pace_s_per_km)}</strong
      >
    </div>
    <div>
      <span>Avg heart rate</span><strong>{formatHr(activity.avg_hr)}</strong>
    </div>
    <div>
      <span>Elevation</span><strong
        >{formatDistance(activity.elevation_gain_m)}</strong
      >
    </div>
    <div>
      <span>Samples</span><strong>{heartRateSamples.length || "—"}</strong>
    </div>
  </div>

  <div class="record-grid">
    <section class="trace-panel">
      <div class="panel-heading">
        <div>
          <p class="theme-kicker">Evidence stream</p>
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
        {:else}<p>No route geometry was recorded for this session.</p>{/if}
      </div>
      <p class="panel-note">
        {swimming
          ? "GPS fixes remain visible as an open-water trace; no pool intervals are inferred."
          : "The presentation keeps the source trace visible without hiding the underlying record."}
      </p>
    </section>
    <aside class="context-panel">
      <p class="theme-kicker">Session notes</p>
      <h2>What remains</h2>
      <dl>
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
    <section class="laps-panel">
      <div class="panel-heading">
        <div>
          <p class="theme-kicker">Intervals</p>
          <h2>Splits in the source</h2>
        </div>
        <span>{laps.length} laps</span>
      </div>
      <div class="lap-list">
        {#each laps.slice(0, 12) as lap (lap.id)}<div>
            <b>{String(lap.lap_no).padStart(2, "0")}</b><span
              >{formatDistance(lap.distance_m)}</span
            ><span>{formatDuration(lap.duration_s)}</span><strong
              >{formatPace(lap.avg_pace_s_per_km)}</strong
            >
          </div>{/each}
      </div>
    </section>
  {/if}
</article>

<style>
  .theme-activity-detail {
    display: grid;
    gap: 1.25rem;
    min-width: 0;
    max-width: 100%;
  }
  .detail-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding: 0.5rem 0 1.5rem;
  }
  h1,
  h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-weight: 400;
    letter-spacing: -0.05em;
  }
  h1 {
    max-width: 14ch;
    font-size: clamp(2.5rem, 7vw, 5.5rem);
    line-height: 0.92;
  }
  h2 {
    font-size: 1.55rem;
  }
  .theme-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .detail-date,
  .panel-heading > span,
  .panel-note {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .hero-mark {
    display: grid;
    color: var(--accent);
    text-align: right;
  }
  .hero-mark span {
    font:
      3rem Georgia,
      serif;
  }
  .hero-mark small {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .metric-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    min-width: 0;
    border: 1px solid var(--border);
  }
  .metric-grid div {
    display: grid;
    gap: 0.35rem;
    padding: 0.9rem;
    border-right: 1px solid var(--border);
    min-width: 0;
  }
  .metric-grid div:last-child {
    border-right: 0;
  }
  .metric-grid span {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .metric-grid strong {
    font-size: 1rem;
    font-weight: 600;
  }
  .record-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 0.85fr);
    gap: 1.25rem;
    min-width: 0;
  }
  .trace-panel,
  .context-panel,
  .laps-panel {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
    padding: 1.25rem;
    min-width: 0;
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
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
    line-height: 1.5;
  }
  dl {
    display: grid;
    gap: 0;
    margin: 1.25rem 0 0;
  }
  dl div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  dd {
    margin: 0;
    font-size: 0.8rem;
    text-align: right;
  }
  .lap-list {
    display: grid;
    gap: 0.65rem;
    margin-top: 1.25rem;
  }
  .lap-list div {
    display: grid;
    grid-template-columns: 2rem 1fr 1fr auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.65rem;
    font-size: 0.8rem;
  }
  .lap-list b {
    color: var(--accent);
  }
  @media (max-width: 800px) {
    .metric-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 0;
    }
    .record-grid {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 480px) {
    .detail-hero {
      align-items: start;
    }
    .hero-mark {
      display: none;
    }
    .metric-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 1px solid var(--border);
    }
  }
</style>
