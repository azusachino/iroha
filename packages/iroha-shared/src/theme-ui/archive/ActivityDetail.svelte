<script lang="ts">
  import type { Snippet } from "svelte";
  import type {
    Activity,
    Lap,
    RoutePoint,
    SamplingPoint,
  } from "../../domain/activity";
  import SourceBadge from "../../components/SourceBadge.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSwimmingPace,
  } from "../../format/format";
  import { isSwimming, sportLabel } from "../../domain/sport";
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
    samplings
      .filter((sample) => /heart|(^|_)hr($|_)/i.test(sample.sampling_type))
      .slice()
      .sort((a, b) => Date.parse(a.ts) - Date.parse(b.ts)),
  );

  const BUCKET_COUNT = 20;

  // The signature device: a session core. The recorded stream (heart rate,
  // or speed when no heart-rate stream survives) is split into equal-time
  // windows across the session. A window's THICKNESS is how many raw
  // readings actually fall inside it -- the record is thicker where
  // evidence survives, thinner where it doesn't -- and its TONE is the
  // real mean value in that window. Deposition rate and reading, from the
  // same stream, the way a stratigrapher reads a physical core.
  const evidence = $derived.by(() => {
    let source: { ts?: string; value: number }[];
    let unit: string;
    let label: "heart rate" | "speed";
    if (heartRateSamples.length > 0) {
      source = heartRateSamples.map((s) => ({ ts: s.ts, value: s.value }));
      unit = heartRateSamples[0]?.unit || "bpm";
      label = "heart rate";
    } else {
      const speedPoints = route.filter((point) =>
        Number.isFinite(point.speed_mps),
      );
      source = speedPoints.map((point) => ({
        ts: point.ts,
        value: point.speed_mps as number,
      }));
      unit = "m/s";
      label = "speed";
    }
    if (source.length === 0) return null;

    const startMs = Date.parse(activity.started_at);
    const totalS =
      activity.duration_s ?? activity.moving_time_s ?? source.length;
    const haveElapsed = Number.isFinite(startMs) && totalS > 0;

    const buckets = Array.from({ length: BUCKET_COUNT }, () => ({
      count: 0,
      sum: 0,
    }));
    source.forEach((point, index) => {
      let bucketIndex: number;
      if (haveElapsed && point.ts) {
        const elapsedS = Math.max(0, (Date.parse(point.ts) - startMs) / 1000);
        bucketIndex = Math.min(
          BUCKET_COUNT - 1,
          Math.max(0, Math.floor((elapsedS / totalS) * BUCKET_COUNT)),
        );
      } else {
        bucketIndex = Math.min(
          BUCKET_COUNT - 1,
          Math.floor((index / source.length) * BUCKET_COUNT),
        );
      }
      buckets[bucketIndex].count += 1;
      buckets[bucketIndex].sum += point.value;
    });

    const values = source.map((point) => point.value);
    const min = Math.min(...values);
    const max = Math.max(...values);
    const maxCount = Math.max(1, ...buckets.map((bucket) => bucket.count));

    return {
      unit,
      source: label,
      totalReadings: source.length,
      buckets: buckets.map((bucket, index) => ({
        index,
        count: bucket.count,
        mean: bucket.count > 0 ? bucket.sum / bucket.count : null,
        magnitude: Math.max(bucket.count / maxCount, 0.05),
        pct:
          bucket.count > 0 && max > min
            ? ((bucket.sum / bucket.count - min) / (max - min)) * 100
            : 50,
      })),
    };
  });

  function tone(pct: number): string {
    const clamped = Math.max(0, Math.min(100, pct));
    return `color-mix(in srgb, var(--accent-2) ${clamped}%, var(--accent) ${100 - clamped}%)`;
  }

  const selectedBucket = $derived(
    evidence && selectedRouteIndex != null
      ? (evidence.buckets[selectedRouteIndex] ?? null)
      : null,
  );

  // Derived from real fields (sport, date) rather than a random identifier
  // -- the same accessioning convention used across the collection.
  const accession = $derived(
    `${(activity.sport_type || "REC").slice(0, 3).toUpperCase()} · ${
      activity.started_at
        ? activity.started_at.slice(0, 10).replaceAll("-", ".")
        : "----.--.--"
    }`,
  );
  const swimming = $derived(isSwimming(activity.sport_type));
  const distanceM = $derived(activity.distance_m ?? derivedDistanceM);
</script>

<article class="folio-detail">
  <header class="detail-hero">
    <div>
      <p class="folio-kicker">
        {sportLabel(activity.sport_type)} / field record
      </p>
      <h1>{activity.title || sportLabel(activity.sport_type)}</h1>
      <p class="detail-date">
        {formatDate(activity.started_at, activity.timezone)}
      </p>
    </div>
    <div class="accession-tag">{accession}</div>
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
      <span>Readings</span><strong>{evidence?.totalReadings || "—"}</strong>
    </div>
  </div>

  {@render children?.()}

  <div class="record-grid">
    <section class="core-panel">
      <div class="panel-heading">
        <div>
          <p class="folio-kicker">Evidence stream</p>
          <h2>{swimming ? "Open-water core" : "Session core"}</h2>
        </div>
        <span
          >{selectedBucket && selectedBucket.mean != null
            ? `${Math.round(selectedBucket.mean)} ${evidence?.unit} · ${selectedBucket.count} readings`
            : "Select a stratum to inspect"}</span
        >
      </div>
      {#if evidence && evidence.buckets.some((bucket) => bucket.count > 0)}
        <div
          class="core-log"
          role="img"
          aria-label="Session core built from recorded readings over time"
        >
          <div class="core-strip horizontal">
            {#each evidence.buckets as bucket (bucket.index)}
              <button
                type="button"
                class="core-band"
                class:active={selectedRouteIndex === bucket.index}
                style={`flex-grow: ${bucket.magnitude}; background: ${bucket.count > 0 ? tone(bucket.pct) : "transparent"};`}
                aria-label={`Segment ${bucket.index + 1}: ${bucket.count} readings`}
                onclick={() =>
                  onSelectRoute(
                    selectedRouteIndex === bucket.index ? null : bucket.index,
                  )}
              ></button>
            {/each}
          </div>
        </div>
        <p class="panel-note">
          {swimming
            ? "Open-water evidence: "
            : "Built from "}{evidence.totalReadings}
          {evidence.source} readings across {BUCKET_COUNT} equal-time segments --
          band width is how many readings survive in that segment, band tone is the
          segment's mean value.
        </p>
      {:else}
        <div class="core-log core-log-empty">
          <p>No heart-rate or speed stream was recorded for this session.</p>
        </div>
      {/if}
    </section>
    <aside class="context-panel catalog-card">
      <p class="folio-kicker">Session notes</p>
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
          <p class="folio-kicker">Intervals</p>
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
  .folio-detail {
    display: grid;
    gap: 1.3rem;
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
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    max-width: 14ch;
    font-size: clamp(2.3rem, 6.5vw, 4.6rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.4rem;
  }
  .folio-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  .detail-date,
  .panel-heading > span,
  .panel-note {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.75rem;
  }
  .accession-tag {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.5rem 0.8rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.82rem;
    letter-spacing: 0.04em;
    white-space: nowrap;
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
    font-size: 0.64rem;
    text-transform: uppercase;
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
  .record-grid > * {
    min-width: 0;
  }
  .core-panel,
  .catalog-card,
  .laps-panel {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.25rem;
    min-width: 0;
  }
  .catalog-card {
    padding-left: 1.45rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.1rem;
    width: 4px;
    height: 2.2rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .core-log {
    display: flex;
    height: 15rem;
    margin-top: 1.25rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.8);
  }
  .core-strip.horizontal {
    display: flex;
    flex: 1;
    flex-direction: row;
  }
  .core-strip.horizontal .core-band {
    flex-shrink: 0;
    height: 100%;
    align-self: flex-end;
    border-top: 0;
    border-left: 1px solid var(--bg);
    padding: 0;
    cursor: pointer;
    opacity: 0.8;
  }
  .core-strip.horizontal .core-band:first-child {
    border-left: 0;
  }
  .core-strip.horizontal .core-band:hover,
  .core-strip.horizontal .core-band.active {
    opacity: 1;
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  .core-log-empty {
    display: grid;
    place-items: center;
  }
  .core-log-empty p {
    margin: 0;
    padding: 1rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  dd {
    margin: 0;
    font-family: var(--font-mono);
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
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  .lap-list b {
    color: var(--accent);
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
      flex-direction: column;
    }
    .metric-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 1px solid var(--border);
    }
  }
</style>
