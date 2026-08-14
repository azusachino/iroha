<script lang="ts">
  import { onMount } from "svelte";
  import { LineChart } from "echarts/charts";
  import {
    AxisPointerComponent,
    GridComponent,
    TooltipComponent,
  } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([
    LineChart,
    AxisPointerComponent,
    GridComponent,
    TooltipComponent,
    CanvasRenderer,
  ]);

  export interface SmallMultiple {
    label: string;
    values: (number | null)[];
    color: string;
    unit?: string;
  }

  let { labels, charts }: { labels: string[]; charts: SmallMultiple[] } =
    $props();
  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const text = styles.getPropertyValue("--text").trim();
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const count = Math.max(charts.length, 1);
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    const columns = count > 2 ? 2 : count;
    const rows = Math.ceil(count / columns);
    const grid = charts.map((_, index) => ({
      left: `${(index % columns) * (100 / columns) + 5}%`,
      top: `${Math.floor(index / columns) * (100 / rows) + 8}%`,
      width: `${100 / columns - 10}%`,
      height: `${100 / rows - 18}%`,
    }));
    chart.clear();
    chart.setOption({
      animation: !reducedMotion,
      animationDuration: reducedMotion ? 0 : 450,
      grid,
      axisPointer: { link: [{ xAxisIndex: "all" }], snap: true },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: border,
        textStyle: { color: text, fontSize: 11 },
      },
      xAxis: charts.map((_, index) => ({
        type: "category",
        gridIndex: index,
        data: labels,
        boundaryGap: false,
        axisLabel: {
          show: index >= charts.length - columns,
          color: muted,
          fontSize: 9,
          interval: Math.max(0, Math.floor(labels.length / 4) - 1),
        },
        axisLine: { lineStyle: { color: border } },
        axisTick: { show: false },
        splitLine: { show: false },
      })),
      yAxis: charts.map((item, index) => ({
        type: "value",
        gridIndex: index,
        name: item.unit ? `${item.label} (${item.unit})` : item.label,
        nameTextStyle: { color: item.color, fontSize: 10, fontWeight: 650 },
        axisLabel: { color: muted, fontSize: 9 },
        axisLine: { show: true, lineStyle: { color: item.color } },
        splitLine: { lineStyle: { color: border, opacity: 0.35 } },
      })),
      series: charts.map((item, index) => ({
        name: item.label,
        type: "line",
        xAxisIndex: index,
        yAxisIndex: index,
        showSymbol: false,
        smooth: 0.18,
        lineStyle: { color: item.color, width: 2 },
        itemStyle: { color: item.color },
        areaStyle: { color: item.color, opacity: 0.08 },
        data: item.values.map((value) =>
          value == null || !Number.isFinite(value) ? null : value,
        ),
      })),
    });
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
    labels;
    charts;
    render();
  });
</script>

<div
  class="small-multiples"
  bind:this={container}
  role="img"
  aria-label="Daily trends with synchronized crosshair. Exact values are available in the trend data table."
></div>
<details class="chart-data">
  <summary>View trend data</summary>
  <div class="table-wrap">
    <table>
      <caption>Daily trend values</caption>
      <thead>
        <tr>
          <th scope="col">Period</th>
          {#each charts as item}<th scope="col">{item.label}</th>{/each}
        </tr>
      </thead>
      <tbody>
        {#each labels as label, index}
          <tr>
            <th scope="row">{label}</th>
            {#each charts as item}
              <td>{item.values[index] == null ? "No observation" : `${item.values[index]}${item.unit ? ` ${item.unit}` : ""}`}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</details>

<style>
  .small-multiples {
    width: 100%;
    height: 300px;
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
    min-width: 32rem;
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
