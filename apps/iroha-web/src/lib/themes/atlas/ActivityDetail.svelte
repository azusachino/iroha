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

  // Plot the real recorded lat/lon as a normalized polyline instead of a
  // synthetic waveform: atlas is the "places and routes" language, so the
  // trace panel should read as an actual (unlabelled, scale-free) plot of
  // where the activity happened, not a decorative stand-in.
  const PAD = 10;
  const SIZE = 100 - PAD * 2;
  const projected = $derived.by(() => {
    if (!tracePoints.length) {
      return {
        points: [] as { seq: number; x: number; y: number }[],
        path: "",
        minLat: 0,
        maxLat: 0,
        minLon: 0,
        maxLon: 0,
      };
    }
    let minLat = Infinity;
    let maxLat = -Infinity;
    let minLon = Infinity;
    let maxLon = -Infinity;
    for (const point of tracePoints) {
      minLat = Math.min(minLat, point.lat);
      maxLat = Math.max(maxLat, point.lat);
      minLon = Math.min(minLon, point.lon);
      maxLon = Math.max(maxLon, point.lon);
    }
    const meanLat = (minLat + maxLat) / 2;
    const lonScale = Math.cos((meanLat * Math.PI) / 180) || 1;
    const latSpan = Math.max(maxLat - minLat, 0.00005);
    const lonSpan = Math.max((maxLon - minLon) * lonScale, 0.00005);
    const span = Math.max(latSpan, lonSpan);
    const xOffset = (SIZE - (lonSpan / span) * SIZE) / 2;
    const yOffset = (SIZE - (latSpan / span) * SIZE) / 2;
    const points = tracePoints.slice(0, 160).map((point) => ({
      seq: point.seq,
      x: PAD + xOffset + (((point.lon - minLon) * lonScale) / span) * SIZE,
      y: PAD + yOffset + (1 - (point.lat - minLat) / span) * SIZE,
    }));
    const path = points
      .map(
        (point, index) =>
          `${index === 0 ? "M" : "L"}${point.x.toFixed(2)},${point.y.toFixed(2)}`,
      )
      .join(" ");
    return { points, path, minLat, maxLat, minLon, maxLon };
  });

  function coord(value: number, positive: string, negative: string): string {
    return `${Math.abs(value).toFixed(3)}°${value >= 0 ? positive : negative}`;
  }
</script>

<article class="atlas-plot">
  <header class="plot-hero">
    <div>
      <p class="atlas-kicker">
        {sportLabel(activity.sport_type)} · route plate
      </p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="plot-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="grid-ref" aria-label="Recorded trace points">
      <span>{String(tracePoints.length).padStart(3, "0")}</span>
      <small>fixes logged</small>
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

  <div class="plot-grid">
    <section class="atlas-plate trace-plate">
      <div class="panel-heading">
        <div>
          <p class="atlas-kicker">Plotted trace</p>
          <h2>{swimming ? "Open-water plate" : "Route plate"}</h2>
        </div>
        <span
          >{selectedRouteIndex == null
            ? "Select a fix to inspect"
            : `Fix ${selectedRouteIndex + 1}`}</span
        >
      </div>
      <div
        class="trace-map"
        aria-label="Route trace, plotted from recorded coordinates"
      >
        {#if tracePoints.length}
          <span class="trace-compass" aria-hidden="true">N ↑</span>
          <svg
            viewBox="0 0 100 100"
            preserveAspectRatio="none"
            aria-hidden="true"
          >
            <path class="trace-line" d={projected.path} />
          </svg>
          {#each projected.points as point, index (point.seq)}
            <button
              class:active={selectedRouteIndex === index}
              class:endpoint={index === 0 ||
                index === projected.points.length - 1}
              style={`--x: ${point.x}%; --y: ${point.y}%;`}
              aria-label={`Select route fix ${index + 1}`}
              onclick={() =>
                onSelectRoute(selectedRouteIndex === index ? null : index)}
            ></button>
          {/each}
        {:else}
          <p class="trace-empty">
            No route geometry was recorded for this session.
          </p>
        {/if}
      </div>
      {#if tracePoints.length}
        <dl class="coord-readout">
          <div>
            <dt>Latitude span</dt>
            <dd>
              {coord(projected.minLat, "N", "S")} – {coord(
                projected.maxLat,
                "N",
                "S",
              )}
            </dd>
          </div>
          <div>
            <dt>Longitude span</dt>
            <dd>
              {coord(projected.minLon, "E", "W")} – {coord(
                projected.maxLon,
                "E",
                "W",
              )}
            </dd>
          </div>
        </dl>
      {/if}
      <p class="panel-note">
        {swimming
          ? "Plotted directly from the recorded GPS fixes — an open-water trace, not a pool interval model."
          : "Plotted directly from the recorded fixes — the shape is real, the source trace stays visible rather than hidden behind a summary."}
      </p>
    </section>
    <aside class="atlas-plate notes-plate">
      <p class="atlas-kicker">Field notes</p>
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
    <section class="atlas-plate laps-plate">
      <div class="panel-heading">
        <div>
          <p class="atlas-kicker">Intervals</p>
          <h2>Splits in the source</h2>
        </div>
        <span>{laps.length} laps</span>
      </div>
      <LapChart {laps} {swimming} />
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

  <footer class="atlas-source">
    Source: imported route and sample records · presentation only
  </footer>
</article>

<style>
  .atlas-plot {
    display: grid;
    gap: 1.5rem;
    min-width: 0;
    max-width: 100%;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 15ch;
    font-size: clamp(2.2rem, 6vw, 4rem);
    line-height: 1;
  }
  h2 {
    font-size: 1.3rem;
  }
  .plot-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding: 0.5rem 0 1.5rem;
  }
  .plot-date {
    margin: 0.8rem 0 0;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.85rem;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 0.85rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.3rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .metric-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    min-width: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
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
    font-family: var(--font-mono);
    font-size: 0.65rem;
  }
  .metric-grid strong {
    font-family: var(--font-mono);
    font-size: 1rem;
    font-weight: 600;
  }
  .plot-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 0.85fr);
    gap: 1.25rem;
    min-width: 0;
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .trace-plate,
  .notes-plate,
  .laps-plate {
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
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  .trace-map {
    position: relative;
    min-height: 16rem;
    margin-top: 1.25rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
    background:
      repeating-linear-gradient(
        0deg,
        transparent 0 2.1rem,
        color-mix(in srgb, var(--accent) 12%, transparent) 2.1rem 2.16rem
      ),
      repeating-linear-gradient(
        90deg,
        transparent 0 3.2rem,
        color-mix(in srgb, var(--accent) 10%, transparent) 3.2rem 3.26rem
      );
  }
  .trace-compass {
    position: absolute;
    top: 0.6rem;
    right: 0.7rem;
    z-index: 1;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.68rem;
  }
  .trace-map svg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
  }
  .trace-line {
    fill: none;
    stroke: var(--accent-2);
    stroke-width: 0.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .trace-map button {
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
    opacity: 0.5;
  }
  .trace-map button.endpoint {
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.9;
    background: var(--accent-2);
  }
  .trace-map button.active {
    width: 0.9rem;
    height: 0.9rem;
    opacity: 1;
    box-shadow: 0 0 0 0.3rem color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .trace-empty {
    padding: 1rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .coord-readout {
    display: grid;
    gap: 0;
    margin: 1rem 0 0;
  }
  .coord-readout div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px dashed var(--border);
    padding: 0.6rem 0;
  }
  .coord-readout dt {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .coord-readout dd {
    margin: 0;
    font-family: var(--font-mono);
    font-size: 0.78rem;
    text-align: right;
  }
  .panel-note {
    margin-top: 1rem;
    color: var(--text-muted);
    font-size: 0.76rem;
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
    border-top: 1px dashed var(--border);
    padding: 0.75rem 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
    font-size: 0.8rem;
    text-align: right;
  }
  .lap-list {
    display: grid;
    gap: 0.6rem;
    margin-top: 1.25rem;
  }
  .lap-list div {
    display: grid;
    grid-template-columns: 2rem 1fr 1fr auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.6rem;
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  .lap-list b {
    color: var(--accent);
    font-weight: 600;
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.64rem;
  }
  @media (max-width: 800px) {
    .metric-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 0;
    }
    .plot-grid {
      grid-template-columns: 1fr;
    }
  }
  @media (max-width: 480px) {
    .plot-hero {
      align-items: start;
    }
    .grid-ref {
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
