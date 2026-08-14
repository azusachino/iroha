<script lang="ts">
  import type { Snippet } from "svelte";
  import type {
    Activity,
    Lap,
    RoutePoint,
    SamplingPoint,
  } from "../../activity";
  import SourceBadge from "../../SourceBadge.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSwimmingPace,
  } from "../../format";
  import { isSwimming, sportLabel } from "../../sport";
  import LapChart from "../components/LapChart.svelte";

  let {
    activity,
    derivedDistanceM,
    route,
    samplings,
    laps,
    selectedRouteIndex,
    onSelectRoute,
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

  // Phyllotaxis spiral: the same construction that spaces a sunflower's
  // seeds or a fern's fronds, angle_i = i * golden angle, radius_i grows
  // with sqrt(i). It keeps the source order visible (seed to edge) as an
  // unfolding pattern instead of a decorative sine wave or a real map --
  // honest about not having geometry, distinct from a generic placeholder.
  const GOLDEN_ANGLE = 137.50776;
  const CENTER = 50;
  const MAX_RADIUS = 44;
  const spiral = $derived.by(() => {
    const points = tracePoints.slice(0, 200);
    const n = points.length;
    if (n === 0) return [] as { seq: number; x: number; y: number }[];
    const scale = MAX_RADIUS / Math.sqrt(Math.max(n - 1, 1));
    return points.map((point, index) => {
      const radius = scale * Math.sqrt(index);
      const angle = (index * GOLDEN_ANGLE * Math.PI) / 180;
      return {
        seq: point.seq,
        x: CENTER + radius * Math.cos(angle),
        y: CENTER + radius * Math.sin(angle),
      };
    });
  });
</script>

<article class="bloom-detail">
  <header class="detail-hero">
    <div>
      <p class="bloom-kicker">
        {sportLabel(activity.sport_type)} · field record
      </p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="detail-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="hero-mark" aria-hidden="true">
      <span>{String(tracePoints.length).padStart(2, "0")}</span>
      <small>trace<br />points</small>
    </div>
  </header>

  <dl class="metric-grid">
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

  {@render children?.()}

  <div class="record-grid">
    <section class="spiral-panel">
      <div class="panel-heading">
        <div>
          <p class="bloom-kicker">Evidence, unfolding</p>
          <h2>{swimming ? "Open-water spiral" : "Growth spiral"}</h2>
        </div>
        <span
          >{selectedRouteIndex == null
            ? "Select a point to inspect"
            : `Point ${selectedRouteIndex + 1}`}</span
        >
      </div>
      <div
        class="spiral-field"
        aria-label="Route trace laid out as a growth spiral, in recorded order"
      >
        {#if spiral.length}
          <svg viewBox="0 0 100 100" aria-hidden="true">
            <circle cx={CENTER} cy={CENTER} r="1.1" class="spiral-seed" />
          </svg>
          {#each spiral as point, index (point.seq)}
            <button
              class:active={selectedRouteIndex === index}
              class:endpoint={index === spiral.length - 1}
              style={`--x: ${point.x}%; --y: ${point.y}%;`}
              aria-label={`Select route point ${index + 1}`}
              onclick={() =>
                onSelectRoute(selectedRouteIndex === index ? null : index)}
            ></button>
          {/each}
        {:else}
          <p class="spiral-empty">
            No route geometry was recorded for this session.
          </p>
        {/if}
      </div>
      <p class="panel-note">
        {swimming
          ? "GPS fixes are placed in recorded order around the spiral. This is an open-water trace; no pool intervals are inferred."
          : "Points are placed in the order they were recorded, spiralling outward from the first fix — the shape traces sequence, not geography."}
      </p>
    </section>
    <aside class="notes-panel">
      <p class="bloom-kicker">Session notes</p>
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
    <section class="laps-panel">
      <div class="panel-heading">
        <div>
          <p class="bloom-kicker">Intervals</p>
          <h2>Splits in the source</h2>
        </div>
        <span>{laps.length} laps</span>
      </div>
      <LapChart {laps} {swimming} />
      <div class="lap-list">
        {#each laps.slice(0, 12) as lap (lap.id)}
          <div>
            <b>{String(lap.lap_no).padStart(2, "0")}</b>
            <span>{formatDistance(lap.distance_m)}</span>
            <span>{formatDuration(lap.duration_s)}</span>
            <strong>{formatPace(lap.avg_pace_s_per_km)}</strong>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <footer class="bloom-source">
    Source: imported activity record · presentation only
  </footer>
</article>

<style>
  .bloom-detail {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
    min-width: 0;
    max-width: 100%;
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 400;
    letter-spacing: -0.02em;
  }
  h1 {
    max-width: 14ch;
    font-size: clamp(2.3rem, 6vw, 4.6rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.4rem;
  }
  .detail-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: start;
    border-bottom: 1px solid var(--border);
    padding: 0.4rem 0 1.5rem;
  }
  .detail-date {
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    font-style: italic;
  }
  .hero-mark {
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
  .hero-mark span {
    font-style: italic;
    font-size: 1.35rem;
  }
  .hero-mark small {
    margin-top: 0.2rem;
    font-size: 0.5rem;
    letter-spacing: 0.1em;
    line-height: 1.1;
    text-transform: uppercase;
  }
  .metric-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    min-width: 0;
    margin: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
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
  dt {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  dd {
    margin: 0;
    font-size: 0.92rem;
  }
  .record-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 0.85fr);
    gap: 1.25rem;
    min-width: 0;
  }
  .spiral-panel,
  .notes-panel,
  .laps-panel {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
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
  .spiral-field {
    position: relative;
    min-height: 16rem;
    margin-top: 1.25rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
    background: radial-gradient(
      circle at 50% 50%,
      color-mix(in srgb, var(--accent) 8%, transparent),
      transparent 65%
    );
  }
  .spiral-field svg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
  }
  .spiral-seed {
    fill: color-mix(in srgb, var(--accent-2) 60%, transparent);
  }
  .spiral-field button {
    position: absolute;
    left: var(--x);
    top: var(--y);
    width: 0.4rem;
    height: 0.4rem;
    padding: 0;
    transform: translate(-50%, -50%);
    border: 0;
    border-radius: 50%;
    background: var(--accent);
    opacity: 0.55;
  }
  .spiral-field button.endpoint {
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.9;
    background: var(--accent-2);
  }
  .spiral-field button.active {
    width: 0.85rem;
    height: 0.85rem;
    opacity: 1;
    box-shadow: 0 0 0 0.3rem color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .spiral-empty {
    padding: 1rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .panel-note {
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.55;
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
    border-top: 1px dotted var(--border);
    padding: 0.8rem 0;
  }
  .notes-list dt {
    font-size: 0.75rem;
  }
  .notes-list dd {
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
    font-weight: 400;
  }
  .bloom-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 1024px) {
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
  @media (max-width: 640px) {
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
