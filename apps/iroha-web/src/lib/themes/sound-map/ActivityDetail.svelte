<script lang="ts">
  import type { Snippet } from "svelte";
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
    samplings
      .filter((sample) => /heart|(^|_)hr($|_)/i.test(sample.sampling_type))
      .slice()
      .sort((a, b) => Date.parse(a.ts) - Date.parse(b.ts)),
  );

  // The signature device: a real amplitude waveform, the same kind an audio
  // editor draws over a track. Bar height is each sample's deviation from
  // the session mean, mirrored around a centerline like a standard waveform
  // render -- honest because the bars ARE the recorded stream (heart rate
  // first, speed as a fallback when no heart-rate stream was captured), not
  // a decorative sine.
  const waveform = $derived.by(() => {
    if (heartRateSamples.length > 0) {
      const values = heartRateSamples.map((sample) => sample.value);
      const startMs = Date.parse(activity.started_at);
      return {
        source: "heart rate" as const,
        unit: heartRateSamples[0]?.unit || "bpm",
        points: heartRateSamples.map((sample, index) => ({
          index,
          value: sample.value,
          elapsedS: Number.isFinite(startMs)
            ? Math.max(0, (Date.parse(sample.ts) - startMs) / 1000)
            : null,
        })),
        mean: values.reduce((sum, value) => sum + value, 0) / values.length,
      };
    }
    const speedPoints = route.filter((point) =>
      Number.isFinite(point.speed_mps),
    );
    if (speedPoints.length > 0) {
      const values = speedPoints.map((point) => point.speed_mps as number);
      return {
        source: "speed" as const,
        unit: "m/s",
        points: speedPoints.map((point, index) => ({
          index,
          value: point.speed_mps as number,
          elapsedS: null as number | null,
        })),
        mean: values.reduce((sum, value) => sum + value, 0) / values.length,
      };
    }
    return null;
  });
  const swimming = $derived(isSwimming(activity.sport_type));
  const distanceM = $derived(activity.distance_m ?? derivedDistanceM);

  const spread = $derived(
    waveform
      ? Math.max(
          0.001,
          ...waveform.points.map((point) =>
            Math.abs(point.value - waveform.mean),
          ),
        )
      : 1,
  );

  const selectedPoint = $derived(
    waveform && selectedRouteIndex != null
      ? (waveform.points[selectedRouteIndex] ?? null)
      : null,
  );

  function elapsedLabel(seconds: number | null): string {
    if (seconds == null) return "";
    const total = Math.round(seconds);
    const m = Math.floor(total / 60);
    const s = total % 60;
    return `${m}:${String(s).padStart(2, "0")}`;
  }
</script>

<article class="mix-detail">
  <header class="detail-hero">
    <div>
      <p class="mix-kicker">{sportLabel(activity.sport_type)} · field record</p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="detail-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="hero-mark" aria-hidden="true">
      <span>{String(waveform?.points.length ?? 0).padStart(2, "0")}</span>
      <small>signal<br />samples</small>
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
      <span>Samples</span><strong>{waveform?.points.length || "—"}</strong>
    </div>
  </div>

  {@render children?.()}

  <div class="record-grid">
    <section class="wave-panel">
      <div class="panel-heading">
        <div>
          <p class="mix-kicker">Evidence stream</p>
          <h2>{swimming ? "Open-water signal" : "Signal waveform"}</h2>
        </div>
        <span
          >{selectedPoint
            ? `${Math.round(selectedPoint.value)} ${waveform?.unit}${elapsedLabel(selectedPoint.elapsedS) ? ` @ ${elapsedLabel(selectedPoint.elapsedS)}` : ""}`
            : "Select a bar to inspect"}</span
        >
      </div>
      {#if waveform && waveform.points.length}
        <div
          class="wave-track"
          role="img"
          aria-label="Session intensity waveform"
        >
          {#each waveform.points as point (point.index)}
            <button
              class:active={selectedRouteIndex === point.index}
              style={`--h: ${Math.max(4, (Math.abs(point.value - waveform.mean) / spread) * 100)}%;`}
              aria-label={`Select sample ${point.index + 1}`}
              onclick={() =>
                onSelectRoute(
                  selectedRouteIndex === point.index ? null : point.index,
                )}
            ></button>
          {/each}
        </div>
        <p class="panel-note">
          {swimming
            ? "GPS and heart-rate evidence from "
            : "Built from "}{waveform.points.length}
          {waveform.source} samples · bar height is each sample's distance from the
          session mean ({Math.round(waveform.mean)}
          {waveform.unit}), mirrored around a centerline like a track waveform.
        </p>
      {:else}
        <div class="wave-track wave-track-empty">
          <p>No heart-rate or speed stream was recorded for this session.</p>
        </div>
      {/if}
    </section>
    <aside class="context-panel">
      <p class="mix-kicker">Session notes</p>
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
          <p class="mix-kicker">Intervals</p>
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
</article>

<style>
  .mix-detail {
    display: grid;
    gap: 1.35rem;
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
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 15ch;
    font-size: clamp(2.1rem, 6vw, 4.4rem);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.4rem;
  }
  .mix-kicker {
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
    padding: 0.5rem 0.8rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    color: var(--accent);
    text-align: right;
  }
  .hero-mark span {
    font-size: 2.4rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .hero-mark small {
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    line-height: 1.15;
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
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .metric-grid strong {
    font-size: 1rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .record-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.55fr) minmax(0, 0.85fr);
    gap: 1.25rem;
    min-width: 0;
  }
  .wave-panel,
  .context-panel,
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
  .wave-track {
    display: flex;
    align-items: center;
    gap: 2px;
    height: 15rem;
    margin-top: 1.25rem;
    padding: 0 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
    background: repeating-linear-gradient(
      0deg,
      transparent 0 calc(50% - 1px),
      color-mix(in srgb, var(--accent) 25%, transparent) calc(50% - 1px)
        calc(50% + 1px),
      transparent calc(50% + 1px) 100%
    );
  }
  .wave-track button {
    flex: 1 1 0;
    min-width: 1px;
    height: var(--h);
    padding: 0;
    border: 0;
    background: var(--accent);
    opacity: 0.55;
    cursor: pointer;
  }
  .wave-track button.active {
    background: var(--accent-2);
    opacity: 1;
  }
  .wave-track-empty {
    display: grid;
    place-items: center;
    background: none;
  }
  .wave-track-empty p {
    margin: 0;
    padding: 1rem;
    color: var(--text-muted);
    font-size: 0.8rem;
    text-align: center;
  }
  .panel-note {
    margin-top: 1rem;
    line-height: 1.55;
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
    font-variant-numeric: tabular-nums;
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
    font-variant-numeric: tabular-nums;
  }
  .lap-list b {
    color: var(--accent);
    font-weight: 700;
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
