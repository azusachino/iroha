<script lang="ts">
  import { onMount } from "svelte";
  import { LineChart } from "echarts/charts";
  import { GridComponent, TooltipComponent } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

  let {
    xValues,
    xLabel,
    pace,
    heartRate,
    elevation,
    paceLabel = "Pace",
    onHover,
  }: {
    xValues: number[];
    xLabel: string;
    pace: (number | null)[];
    heartRate: (number | null)[];
    elevation: (number | null)[];
    paceLabel?: string;
    onHover?: (index: number | null) => void;
  } = $props();

  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const text = styles.getPropertyValue("--text").trim();
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const surface = styles.getPropertyValue("--surface-2").trim();
    const paceColor =
      styles.getPropertyValue("--chart-pace").trim() || "#4f8cff";
    const heartRateColor =
      styles.getPropertyValue("--chart-heart-rate").trim() || "#ff6b6b";
    const elevationColor =
      styles.getPropertyValue("--chart-elevation").trim() || "#3ecf8e";
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    chart.setOption({
      animation: !reducedMotion,
      animationDuration: reducedMotion ? 0 : 500,
      grid: { top: 34, right: 52, bottom: 52, left: 48 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "line", lineStyle: { color: muted, width: 1 } },
        backgroundColor: surface,
        borderColor: border,
        textStyle: { color: text, fontSize: 11 },
        formatter: (
          params: Array<{
            seriesName: string;
            value: [number, number | null];
            color: string;
          }>,
        ) => {
          const x = params[0]?.value?.[0];
          const rows = params
            .filter((item) => item.value?.[1] != null)
            .map(
              (item) =>
                `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${formatValue(item.value[1], item.seriesName)}</strong>`,
            );
          return `${xLabel}: ${formatXAxis(x)}<br/>${rows.join("<br/>")}`;
        },
      },
      xAxis: {
        type: "value",
        name: xLabel,
        nameLocation: "middle",
        nameGap: 30,
        axisLabel: {
          color: muted,
          fontSize: 10,
          formatter: (value: number) => formatXAxis(value),
        },
        axisLine: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.55 } },
      },
      yAxis: [
        {
          type: "value",
          name: paceLabel,
          inverse: true,
          axisLabel: { color: paceColor, fontSize: 10, formatter: formatPace },
          axisLine: { lineStyle: { color: paceColor } },
          splitLine: { lineStyle: { color: border, opacity: 0.35 } },
        },
        {
          type: "value",
          name: "HR",
          position: "right",
          axisLabel: { color: heartRateColor, fontSize: 10 },
          axisLine: { lineStyle: { color: heartRateColor } },
          splitLine: { show: false },
        },
        {
          type: "value",
          name: "Elev.",
          position: "right",
          offset: 42,
          axisLabel: { color: elevationColor, fontSize: 10 },
          axisLine: { lineStyle: { color: elevationColor } },
          splitLine: { show: false },
        },
      ],
      series: [
        makeSeries(paceLabel, pace, paceColor, 0),
        makeSeries("Heart rate", heartRate, heartRateColor, 1),
        makeSeries("Elevation", elevation, elevationColor, 2),
      ].filter((series) => series.data.some((point) => point[1] != null)),
    });
    chart.off("updateAxisPointer");
    chart.on("updateAxisPointer", (rawEvent) => {
      const event = rawEvent as { axesInfo?: Array<{ value?: number }> };
      const x = event.axesInfo?.[0]?.value;
      if (x == null || xValues.length === 0) return;
      let nearest = 0;
      for (let i = 1; i < xValues.length; i++)
        if (Math.abs(xValues[i] - x) < Math.abs(xValues[nearest] - x))
          nearest = i;
      onHover?.(nearest);
    });
    chart.on("globalout", () => onHover?.(null));
  }

  function makeSeries(
    name: string,
    values: (number | null)[],
    color: string,
    yAxisIndex: number,
  ) {
    return {
      name,
      type: "line",
      yAxisIndex,
      showSymbol: false,
      smooth: 0.12,
      connectNulls: false,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      data: xValues.map((x, i) => [x, values[i] ?? null]),
    };
  }

  function formatXAxis(value: number | undefined) {
    return value == null
      ? ""
      : xLabel.includes("Distance")
        ? `${value.toFixed(value < 10 ? 1 : 0)} km`
        : `${value.toFixed(value < 10 ? 1 : 0)} min`;
  }

  function formatPace(value: number) {
    if (!Number.isFinite(value)) return "";
    const min = Math.floor(value / 60);
    return `${min}:${String(Math.round(value % 60)).padStart(2, "0")}`;
  }

  function formatValue(value: number | null, name: string) {
    if (value == null) return "—";
    return name === paceLabel
      ? formatPace(value)
      : `${value.toFixed(0)}${name === "Heart rate" ? " bpm" : " m"}`;
  }

  onMount(() => {
    chart = init(container, undefined, { renderer: "canvas" });
    render();
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(container);
    return () => {
      resize.disconnect();
      chart?.dispose();
    };
  });

  $effect(() => {
    xValues;
    xLabel;
    pace;
    heartRate;
    elevation;
    paceLabel;
    render();
  });
</script>

<div
  class="chart"
  bind:this={container}
  role="img"
  aria-label="Synchronized activity chart. Exact measurements are available in the activity data table."
></div>
<details class="chart-data">
  <summary>View activity data</summary>
  <div class="table-wrap">
    <table>
      <caption>Synchronized activity measurements</caption>
      <thead>
        <tr>
          <th scope="col">{xLabel}</th>
          <th scope="col">{paceLabel}</th>
          <th scope="col">Heart rate</th>
          <th scope="col">Elevation</th>
        </tr>
      </thead>
      <tbody>
        {#each xValues as x, index}
          <tr>
            <th scope="row">{formatXAxis(x)}</th>
            <td>{pace[index] == null ? "No observation" : formatPace(pace[index]!)}</td>
            <td>{heartRate[index] == null ? "No observation" : `${heartRate[index]!.toFixed(0)} bpm`}</td>
            <td>{elevation[index] == null ? "No observation" : `${elevation[index]!.toFixed(0)} m`}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</details>

<style>
  .chart {
    width: 100%;
    height: 330px;
    min-height: 250px;
  }

  .chart-data {
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  .chart-data summary {
    width: fit-content;
    cursor: pointer;
  }

  .table-wrap {
    margin-top: 0.5rem;
    overflow-x: auto;
  }

  table {
    width: 100%;
    min-width: 34rem;
    border-collapse: collapse;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }

  th,
  td {
    padding: 0.35rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: right;
  }

  th:first-child,
  td:first-child {
    text-align: left;
  }

  thead th {
    color: var(--text-muted);
    font-weight: 650;
  }
</style>
