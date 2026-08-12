<script lang="ts">
  import { onMount } from "svelte";
  import * as echarts from "echarts";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatElevation,
    formatHr,
    formatPace,
  } from "$lib/format";
  import RoutesMap from "$lib/components/RoutesMap.svelte";
  import type { ActivityDetail, RouteFeatureCollection } from "$lib/types";

  let { detail, backHref }: { detail: ActivityDetail; backHref: string } =
    $props();

  let chartContainer: HTMLDivElement;
  let chart: echarts.ECharts | null = null;
  const route = $derived(detail.route);
  const heartRateSamples = $derived(
    detail.samplings.filter((sample) =>
      /heart|(^|_)hr($|_)/i.test(sample.sampling_type),
    ),
  );
  const validLaps = $derived(
    detail.laps.filter(
      (lap) =>
        lap.distance_m != null &&
        lap.distance_m > 0 &&
        lap.duration_s != null &&
        lap.duration_s > 0,
    ),
  );
  const routeFeatures = $derived.by<RouteFeatureCollection>(() => ({
    type: "FeatureCollection",
    features: [
      {
        type: "Feature",
        geometry: {
          type: "LineString",
          coordinates: route.map((point) => [point.lon, point.lat]),
        },
        properties: {
          activity_id: detail.activity.id,
          sport_type: detail.activity.sport_type,
          year: detail.activity.started_at.slice(0, 4),
        },
      },
    ],
  }));

  const useDistanceAxis = $derived(
    route.length > 0 &&
      route.every((point) => point.distance_m != null) &&
      heartRateSamples.length === 0,
  );

  function elapsedMinutes(ts: string | undefined, fallback: number): number {
    const start = detail.activity.started_at
      ? Date.parse(detail.activity.started_at)
      : 0;
    if (!ts || !start) return fallback;
    return (Date.parse(ts) - start) / 60000;
  }

  function routeXValue(point: (typeof route)[number]): number {
    if (useDistanceAxis && point.distance_m != null) {
      return point.distance_m / 1000;
    }
    return elapsedMinutes(point.ts, point.seq);
  }

  function renderChart() {
    if (!chart || route.length === 0) return;
    const pace = route
      .map((point) => [
        routeXValue(point),
        point.speed_mps && point.speed_mps > 0 ? 1000 / point.speed_mps : null,
      ])
      .filter(([, value]) => value != null);
    const heartRate = heartRateSamples.map((sample, index) => [
      elapsedMinutes(sample.ts, index),
      sample.value,
    ]);
    const elevation = route
      .map((point) => [routeXValue(point), point.elevation_m ?? null])
      .filter(([, value]) => value != null);

    const yAxis: echarts.YAXisComponentOption[] = [];
    const series: echarts.SeriesOption[] = [];
    const addSeries = (
      name: string,
      data: (number | null)[][],
      axisName: string,
    ) => {
      if (data.length === 0) return;
      const axisIndex = yAxis.length;
      yAxis.push({
        type: "value",
        name: axisName,
        position: axisIndex === 1 ? "right" : "left",
        offset: axisIndex > 1 ? 42 : 0,
      });
      series.push({
        name,
        type: "line",
        yAxisIndex: axisIndex,
        showSymbol: false,
        data,
        connectNulls: false,
      });
    };

    addSeries("Pace", pace, "Pace (s/km)");
    addSeries("Heart rate", heartRate, "Heart rate (bpm)");
    addSeries("Elevation", elevation, "Elevation (m)");

    if (series.length === 0) return;
    chart.setOption({
      animation: false,
      grid: { left: 58, right: 58, top: 52, bottom: 52 },
      tooltip: { trigger: "axis" },
      legend: {
        top: 0,
        left: "center",
        itemGap: 18,
        textStyle: { color: "#68707a" },
      },
      xAxis: {
        type: "value",
        name: useDistanceAxis ? "Distance (km)" : "Elapsed time (min)",
        nameLocation: "middle",
        nameGap: 32,
      },
      yAxis,
      series,
    });
  }

  onMount(() => {
    chart = echarts.init(chartContainer);
    renderChart();
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(chartContainer);
    return () => {
      resize.disconnect();
      chart?.dispose();
      chart = null;
    };
  });

  $effect(() => {
    detail;
    renderChart();
  });
</script>

<article class="grapher-detail tile">
  <a class="back-link" href={backHref}>← Back to archive</a>
  <header class="detail-hero">
    <div>
      <p class="kicker">{detail.activity.sport_type} / field record</p>
      <h1>{detail.activity.title || detail.activity.sport_type}</h1>
      <p class="muted">
        {formatDate(detail.activity.started_at, detail.activity.timezone)}
      </p>
    </div>
    <div class="trace-count">
      <strong>{route.length.toLocaleString()}</strong>
      <span>trace points</span>
    </div>
  </header>

  <div class="metric-grid">
    <div>
      <span>Distance</span><strong
        >{formatDistance(detail.activity.distance_m)}</strong
      >
    </div>
    <div>
      <span>Duration</span><strong
        >{formatDuration(detail.activity.duration_s)}</strong
      >
    </div>
    <div>
      <span>Moving time</span><strong
        >{formatDuration(detail.activity.moving_time_s)}</strong
      >
    </div>
    <div>
      <span>Avg pace</span><strong
        >{formatPace(detail.activity.avg_pace_s_per_km)}</strong
      >
    </div>
    <div>
      <span>Elevation</span><strong
        >{formatElevation(detail.activity.elevation_gain_m)}</strong
      >
    </div>
    <div>
      <span>Avg heart rate</span><strong
        >{formatHr(detail.activity.avg_hr)}</strong
      >
    </div>
  </div>

  <section class="panel map-panel">
    <div class="panel-heading">
      <div>
        <p class="kicker">Geography</p>
        <h2>Route trace</h2>
      </div>
      <span>Owner-approved detail</span>
    </div>
    <div class="map"><RoutesMap data={routeFeatures} /></div>
  </section>

  <section class="panel chart-panel">
    <div class="panel-heading">
      <div>
        <p class="kicker">Evidence stream</p>
        <h2>Activity streams</h2>
      </div>
      <span>{useDistanceAxis ? "Distance" : "Elapsed time"}</span>
    </div>
    <div
      class="chart"
      bind:this={chartContainer}
      aria-label="Activity pace, heart rate, and elevation chart"
    ></div>
  </section>

  {#if validLaps.length}
    <section class="panel laps-panel">
      <div class="panel-heading">
        <div>
          <p class="kicker">Intervals</p>
          <h2>Splits in the source</h2>
        </div>
        <span>{validLaps.length} splits</span>
      </div>
      <div class="laps">
        {#each validLaps as lap (lap.id)}
          <div>
            <b>{String(lap.lap_no).padStart(2, "0")}</b><span
              >{formatDistance(lap.distance_m)}</span
            ><span>{formatDuration(lap.duration_s)}</span><strong
              >{formatPace(lap.avg_pace_s_per_km)}</strong
            >
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <p class="detail-note muted">
    This activity is published in full by explicit owner decision. Other
    activities remain on the sanitized public projection.
  </p>
</article>

<style>
  .grapher-detail {
    display: grid;
    gap: 1.25rem;
    padding: 1.25rem;
  }
  .back-link {
    font-size: 0.85rem;
    font-weight: 700;
  }
  .detail-hero {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding: 0.5rem 0 1.5rem;
  }
  .kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.14em;
    text-transform: uppercase;
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
    font-size: 1.35rem;
  }
  .trace-count {
    display: grid;
    color: var(--accent);
    text-align: right;
  }
  .trace-count strong {
    font:
      3rem Georgia,
      serif;
  }
  .trace-count span,
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .metric-grid {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    border: 1px solid var(--border);
  }
  .metric-grid div {
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    padding: 0.9rem;
    border-right: 1px solid var(--border);
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
  }
  .panel {
    min-width: 0;
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .map {
    height: 25rem;
  }
  .chart {
    width: 100%;
    height: 22rem;
  }
  .laps {
    display: grid;
    gap: 0.5rem;
  }
  .laps div {
    display: grid;
    grid-template-columns: 3rem repeat(3, 1fr);
    gap: 1rem;
    padding: 0.7rem 0;
    border-bottom: 1px solid var(--border);
    font-size: 0.82rem;
  }
  .laps strong {
    text-align: right;
  }
  .detail-note {
    margin: 0;
    font-size: 0.82rem;
  }
  @media (max-width: 800px) {
    .metric-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 0;
    }
  }
  @media (max-width: 500px) {
    .detail-hero {
      align-items: start;
      flex-direction: column;
    }
    .trace-count {
      text-align: left;
    }
    .metric-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
    .metric-grid div:nth-child(3) {
      border-right: 1px solid var(--border);
    }
    .metric-grid div:nth-child(even) {
      border-right: 0;
    }
  }
</style>
