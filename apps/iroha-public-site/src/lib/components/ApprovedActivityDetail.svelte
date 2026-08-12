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
  import type {
    ActivityDetail,
    ActivityDetailLap,
    ActivityDetailRoutePoint,
    RouteFeatureCollection,
  } from "$lib/types";

  let { detail, backHref }: { detail: ActivityDetail; backHref: string } =
    $props();
  let chartContainer = $state<HTMLDivElement>();
  let zoneChartContainer = $state<HTMLDivElement>();
  let lapsChartContainer = $state<HTMLDivElement>();
  let chart: echarts.ECharts | null = null;
  let zoneChart: echarts.ECharts | null = null;
  let lapsChart: echarts.ECharts | null = null;

  const route = $derived(detail.route);
  const sport = $derived(detail.activity.sport_type.toLowerCase());
  const isSwimming = $derived(sport.includes("swim"));
  const supportsDistanceSplits = $derived(
    /run|walk|hike|ride|cycl|swim/.test(sport),
  );
  const paceUnitMeters = $derived(isSwimming ? 100 : 1000);
  const paceLabel = $derived(isSwimming ? "Pace /100m" : "Pace /km");
  const heartRateSamples = $derived(
    detail.samplings.filter((sample) =>
      /heart|(^|_)hr($|_)/i.test(sample.sampling_type),
    ),
  );
  const sourceLaps = $derived(
    detail.laps.filter(
      (lap) =>
        lap.distance_m != null &&
        lap.distance_m > 0 &&
        lap.duration_s != null &&
        lap.duration_s > 0,
    ),
  );

  type ProcessedPoint = ActivityDetailRoutePoint & {
    distance_m: number;
    speed_mps?: number;
    heart_rate?: number;
  };
  type DisplayLap = ActivityDetailLap & {
    distance_m: number;
    duration_s: number;
    avg_pace_s_per_km: number;
  };

  function haversineMeters(
    a: ActivityDetailRoutePoint,
    b: ActivityDetailRoutePoint,
  ): number {
    const earthRadius = 6_371_000;
    const radians = Math.PI / 180;
    const dLat = (b.lat - a.lat) * radians;
    const dLon = (b.lon - a.lon) * radians;
    const lat1 = a.lat * radians;
    const lat2 = b.lat * radians;
    const h =
      Math.sin(dLat / 2) ** 2 +
      Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) ** 2;
    return 2 * earthRadius * Math.asin(Math.sqrt(h));
  }

  const processedRoute = $derived.by<ProcessedPoint[]>(() => {
    let distance = 0;
    const points = route.map((point, index) => {
      if (point.distance_m != null) {
        distance = point.distance_m;
      } else if (index > 0) {
        distance += haversineMeters(route[index - 1], point);
      }
      return { ...point, distance_m: point.distance_m ?? distance };
    });
    const samples = [...heartRateSamples].sort(
      (a, b) => Date.parse(a.ts) - Date.parse(b.ts),
    );
    return points.map((point, index) => {
      let speed = point.speed_mps;
      if (speed == null && index > 0 && point.ts && points[index - 1].ts) {
        const seconds =
          (Date.parse(point.ts) - Date.parse(points[index - 1].ts!)) / 1000;
        const meters = point.distance_m - points[index - 1].distance_m;
        if (seconds > 0 && meters >= 0) speed = meters / seconds;
      }
      let heartRate = point.heart_rate;
      if (heartRate == null && point.ts && samples.length > 0) {
        const timestamp = Date.parse(point.ts);
        let closest = samples[0];
        let closestDistance = Math.abs(Date.parse(closest.ts) - timestamp);
        for (const sample of samples) {
          const sampleDistance = Math.abs(Date.parse(sample.ts) - timestamp);
          if (sampleDistance < closestDistance) {
            closest = sample;
            closestDistance = sampleDistance;
          }
        }
        if (closestDistance <= 15_000) heartRate = closest.value;
      }
      return { ...point, speed_mps: speed, heart_rate: heartRate };
    });
  });

  const xValues = $derived(
    processedRoute.map((point) => point.distance_m / 1000),
  );
  const paceValues = $derived(
    processedRoute.map((point) => {
      if (point.speed_mps == null || point.speed_mps <= 0) return null;
      const pace = paceUnitMeters / point.speed_mps;
      return pace <= 1_200 ? pace : null;
    }),
  );
  const heartRateValues = $derived(
    processedRoute.length > 0
      ? processedRoute.map((point) => point.heart_rate ?? null)
      : heartRateSamples.map((sample) => sample.value),
  );
  const elevationValues = $derived(
    processedRoute.map((point) => point.elevation_m ?? null),
  );
  const measuredHeartRate = $derived(
    heartRateValues.filter(
      (value): value is number => value != null && value > 0,
    ),
  );
  const heartRateZones = $derived.by(() => {
    if (measuredHeartRate.length === 0) return [];
    const maxHr = Math.max(...measuredHeartRate);
    return [
      { label: "Easy", min: 0, max: 0.7, color: "#3ecf8e" },
      { label: "Steady", min: 0.7, max: 0.8, color: "#f5c451" },
      { label: "Tempo", min: 0.8, max: 0.9, color: "#ff9f43" },
      { label: "Hard", min: 0.9, max: 2, color: "#ff6b6b" },
    ]
      .map((zone) => ({
        ...zone,
        count: measuredHeartRate.filter(
          (value) => value <= maxHr * zone.max && value > maxHr * zone.min,
        ).length,
      }))
      .filter((zone) => zone.count > 0);
  });

  const displayLaps = $derived.by<DisplayLap[]>(() => {
    if (sourceLaps.length > 0) return sourceLaps as DisplayLap[];
    if (!supportsDistanceSplits) return [];
    if (processedRoute.length < 2) return [];
    const laps: DisplayLap[] = [];
    let startIndex = 0;
    let lapNo = 1;
    for (let index = 1; index < processedRoute.length; index++) {
      const distance =
        processedRoute[index].distance_m -
        processedRoute[startIndex].distance_m;
      const isLast = index === processedRoute.length - 1;
      if (
        distance < paceUnitMeters &&
        !(isLast && distance > paceUnitMeters * 0.1)
      )
        continue;
      const start = processedRoute[startIndex];
      const end = processedRoute[index];
      const duration =
        start.ts && end.ts
          ? (Date.parse(end.ts) - Date.parse(start.ts)) / 1000
          : 0;
      if (distance > 0 && duration > 0) {
        const hrValues = processedRoute
          .slice(startIndex, index + 1)
          .map((point) => point.heart_rate)
          .filter((value): value is number => value != null);
        laps.push({
          id: `derived-lap-${lapNo}`,
          lap_no: lapNo,
          distance_m: distance,
          duration_s: duration,
          avg_pace_s_per_km: duration / (distance / paceUnitMeters),
          avg_hr:
            hrValues.length > 0
              ? hrValues.reduce((sum, value) => sum + value, 0) /
                hrValues.length
              : undefined,
        });
        lapNo += 1;
        startIndex = index;
      }
    }
    return laps;
  });
  const lapsAreDerived = $derived(sourceLaps.length === 0);

  function formatActivityPace(value?: number): string {
    const formatted = formatPace(value);
    return isSwimming ? formatted.replace("/km", "/100m") : formatted;
  }

  function formatSummaryPace(): string {
    if (
      isSwimming &&
      detail.activity.distance_m &&
      detail.activity.duration_s
    ) {
      return formatPace(
        detail.activity.duration_s / (detail.activity.distance_m / 100),
      ).replace("/km", "/100m");
    }
    return formatActivityPace(detail.activity.avg_pace_s_per_km);
  }

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

  function lineSeries(
    name: string,
    values: (number | null)[],
    color: string,
    yAxisIndex: number,
  ): echarts.SeriesOption {
    return {
      name,
      type: "line",
      yAxisIndex,
      showSymbol: false,
      smooth: 0.12,
      connectNulls: false,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      data: xValues.map((x, index) => [x, values[index] ?? null]),
    };
  }

  function renderChart() {
    if (!chart || !chartContainer || processedRoute.length === 0) return;
    const styles = getComputedStyle(chartContainer);
    const text = styles.getPropertyValue("--text").trim();
    const muted = styles.getPropertyValue("--text-muted").trim() || "#68707a";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const series = [
      lineSeries(paceLabel, paceValues, "#4f8cff", 0),
      lineSeries("Heart rate", heartRateValues, "#ff6b6b", 1),
      lineSeries("Elevation", elevationValues, "#3ecf8e", 2),
    ].filter((item) =>
      (item.data as [number, number | null][]).some(
        (point) => point[1] != null,
      ),
    );
    chart.setOption({
      animation: false,
      grid: { left: 58, right: 92, top: 58, bottom: 54 },
      tooltip: {
        trigger: "axis",
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: border,
        textStyle: { color: text, fontSize: 11 },
      },
      legend: {
        top: 0,
        left: "center",
        itemGap: 18,
        textStyle: { color: muted },
      },
      xAxis: {
        type: "value",
        name: "Distance (km)",
        nameLocation: "middle",
        nameGap: 32,
        axisLabel: { color: muted },
        axisLine: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.55 } },
      },
      yAxis: [
        {
          type: "value",
          name: paceLabel,
          inverse: true,
          axisLabel: {
            color: "#4f8cff",
            formatter: (value: number) => formatActivityPace(value),
          },
          axisLine: { lineStyle: { color: "#4f8cff" } },
          splitLine: { lineStyle: { color: border, opacity: 0.35 } },
        },
        {
          type: "value",
          name: "HR",
          position: "right",
          axisLabel: { color: "#ff6b6b" },
          axisLine: { lineStyle: { color: "#ff6b6b" } },
          splitLine: { show: false },
        },
        {
          type: "value",
          name: "Elev.",
          position: "right",
          offset: 42,
          axisLabel: { color: "#3ecf8e" },
          axisLine: { lineStyle: { color: "#3ecf8e" } },
          splitLine: { show: false },
        },
      ],
      series,
    });
  }

  function renderZoneChart() {
    if (!zoneChart || !zoneChartContainer || measuredHeartRate.length === 0)
      return;
    const styles = getComputedStyle(zoneChartContainer);
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    zoneChart.setOption({
      animation: false,
      grid: { left: 0, right: 0, top: 8, bottom: 8 },
      xAxis: { type: "value", max: measuredHeartRate.length, show: false },
      yAxis: { type: "category", data: ["Heart rate"], show: false },
      tooltip: {
        trigger: "item",
        borderColor: border,
        formatter: (params: { seriesName: string; value: number }) =>
          `${params.seriesName}<br/>${params.value} samples (${Math.round((params.value / measuredHeartRate.length) * 100)}%)`,
      },
      series: heartRateZones.map((zone) => ({
        name: zone.label,
        type: "bar",
        stack: "zones",
        barWidth: "58%",
        data: [zone.count],
        itemStyle: { color: zone.color },
      })),
    });
  }

  function renderLapsChart() {
    if (!lapsChart || !lapsChartContainer || displayLaps.length === 0) return;
    const styles = getComputedStyle(lapsChartContainer);
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const muted = styles.getPropertyValue("--text-muted").trim() || "#68707a";
    lapsChart.setOption({
      animation: false,
      grid: { left: 62, right: 72, top: 12, bottom: 38 },
      xAxis: {
        type: "value",
        name: paceLabel,
        nameLocation: "middle",
        nameGap: 28,
        min: 0,
        axisLabel: {
          color: muted,
          formatter: (value: number) => formatActivityPace(value),
        },
        axisLine: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.45 } },
      },
      yAxis: {
        type: "category",
        inverse: true,
        data: displayLaps.map((lap) => `Split ${lap.lap_no}`),
        axisLabel: { color: muted },
        axisLine: { lineStyle: { color: border } },
      },
      tooltip: {
        trigger: "item",
        borderColor: border,
        formatter: (params: { dataIndex: number; value: number }) => {
          const lap = displayLaps[params.dataIndex];
          return `Split ${lap.lap_no}<br/>${formatActivityPace(params.value)}<br/>${formatDistance(lap.distance_m)} · ${formatDuration(lap.duration_s)}${lap.avg_hr != null ? `<br/>${formatHr(lap.avg_hr)}` : ""}`;
        },
      },
      series: [
        {
          name: "Pace",
          type: "bar",
          barWidth: "55%",
          itemStyle: { color: "#0b9d98", borderRadius: [0, 4, 4, 0] },
          label: {
            show: true,
            position: "right",
            color: "#0b807c",
            formatter: (params: { value: number }) =>
              formatActivityPace(params.value),
          },
          data: displayLaps.map((lap) => lap.avg_pace_s_per_km),
        },
      ],
    });
  }

  onMount(() => {
    if (chartContainer) chart = echarts.init(chartContainer);
    if (zoneChartContainer) zoneChart = echarts.init(zoneChartContainer);
    if (lapsChartContainer) lapsChart = echarts.init(lapsChartContainer);
    renderChart();
    renderZoneChart();
    renderLapsChart();
    const resize = new ResizeObserver(() => {
      chart?.resize();
      zoneChart?.resize();
      lapsChart?.resize();
    });
    if (chartContainer) resize.observe(chartContainer);
    if (zoneChartContainer) resize.observe(zoneChartContainer);
    if (lapsChartContainer) resize.observe(lapsChartContainer);
    return () => {
      resize.disconnect();
      chart?.dispose();
      zoneChart?.dispose();
      lapsChart?.dispose();
      chart = null;
      zoneChart = null;
      lapsChart = null;
    };
  });

  $effect(() => {
    detail;
    processedRoute;
    paceValues;
    heartRateValues;
    elevationValues;
    heartRateZones;
    measuredHeartRate;
    displayLaps;
    renderChart();
    renderZoneChart();
    renderLapsChart();
  });
</script>

<article class="grapher-detail tile">
  <a class="back-link" href={backHref}>← Back to archive</a>
  <header class="detail-hero">
    <div>
      <p class="kicker">Grapher / {detail.activity.sport_type} report</p>
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
      <span>{paceLabel}</span><strong>{formatSummaryPace()}</strong>
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

  {#if route.length > 1}
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
  {/if}

  {#if route.length > 1}
    <section class="panel chart-panel">
      <div class="panel-heading">
        <div>
          <p class="kicker">Evidence stream</p>
          <h2>Pace, heart rate, elevation</h2>
        </div>
        <span>Distance · {processedRoute.length.toLocaleString()} points</span>
      </div>
      <div
        class="chart"
        bind:this={chartContainer}
        aria-label="Activity pace, heart rate, and elevation chart"
      ></div>
    </section>
  {/if}

  {#if heartRateZones.length}
    <section class="panel zone-panel">
      <div class="panel-heading">
        <div>
          <p class="kicker">Intensity</p>
          <h2>Heart-rate zones</h2>
        </div>
        <span>recorded distribution</span>
      </div>
      <div
        class="zone-chart"
        bind:this={zoneChartContainer}
        aria-label="Heart-rate zone distribution chart"
      ></div>
      <div class="zone-legend">
        {#each heartRateZones as zone}
          <span
            ><i style={`background: ${zone.color}`}></i>{zone.label}<b
              >{Math.round((zone.count / measuredHeartRate.length) * 100)}%</b
            ></span
          >
        {/each}
      </div>
    </section>
  {/if}

  {#if displayLaps.length}
    <section class="panel laps-panel">
      <div class="panel-heading">
        <div>
          <p class="kicker">Intervals</p>
          <h2>{isSwimming ? "100 m splits" : "1 km splits"}</h2>
        </div>
        <span
          >{displayLaps.length} splits · {lapsAreDerived
            ? "derived from the route"
            : "from the activity record"}</span
        >
      </div>
      <div
        class="laps-chart"
        style={`height: ${Math.max(12, displayLaps.length * 2.6 + 3)}rem`}
        bind:this={lapsChartContainer}
        aria-label="Split pace chart"
      ></div>
      <div class="table-wrap">
        <table>
          <thead
            ><tr
              ><th>Split</th><th>Distance</th><th>Duration</th><th>Pace</th><th
                >Avg HR</th
              ></tr
            ></thead
          >
          <tbody>
            {#each displayLaps as lap (lap.id)}
              <tr
                ><td><strong>{lap.lap_no}</strong></td><td
                  >{formatDistance(lap.distance_m)}</td
                ><td>{formatDuration(lap.duration_s)}</td><td
                  >{formatActivityPace(lap.avg_pace_s_per_km)}</td
                ><td>{formatHr(lap.avg_hr)}</td></tr
              >
            {/each}
          </tbody>
        </table>
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
    height: 24rem;
  }
  .zone-chart {
    width: 100%;
    height: 5.5rem;
  }
  .zone-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem 1.25rem;
    margin-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .zone-legend span {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }
  .zone-legend i {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }
  .zone-legend b {
    color: var(--text);
    font-weight: 600;
  }
  .laps-chart {
    width: 100%;
    min-height: 15rem;
    margin-bottom: 1.2rem;
  }
  .table-wrap {
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.8rem;
  }
  th,
  td {
    padding: 0.7rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }
  td:last-child,
  th:last-child {
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
    .map {
      height: 20rem;
    }
    .chart {
      height: 22rem;
    }
    .panel-heading {
      align-items: start;
      flex-direction: column;
    }
  }
</style>
