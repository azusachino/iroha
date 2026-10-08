<script lang="ts">
  import { onMount } from "svelte";
  import { axisRange, formatAxisTick } from "../format/axis";
  import { chartFontSize } from "../theme/chart-typography";
  import {
    attachChartInstance,
    observeChartPresentation,
  } from "../theme/chart-presentation";
  import { chartSignature } from "../theme-ui/components/chart-signature";
  import { LineChart } from "echarts/charts";
  import {
    GridComponent,
    LegendComponent,
    TooltipComponent,
  } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";
  import type { ActivitySummaryBucket } from "../domain/activity";
  import { formatDistance } from "../format/format";

  use([
    LineChart,
    GridComponent,
    LegendComponent,
    TooltipComponent,
    CanvasRenderer,
  ]);

  const MONTHS = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];

  let {
    byMonth,
    year,
    sportName,
    embedded = false,
  }: { byMonth: ActivitySummaryBucket[]; year: string; sportName?: string; embedded?: boolean } =
    $props();

  interface YearSeries {
    year: string;
    cumulative: (number | null)[];
    lastIdx: number;
    unknownFrom?: number;
  }

  let chartContainer = $state<HTMLDivElement>();
  let chart: ECharts | undefined;

  function seriesFor(y: string): YearSeries | null {
    const monthly = new Array(12).fill(0);
    let seen = false;
    let lastIdx = -1;
    let unknownFrom: number | undefined;
    for (const bucket of byMonth) {
      if (!bucket.key.startsWith(`${y}-`)) continue;
      const idx = Number(bucket.key.slice(5, 7)) - 1;
      if (idx < 0 || idx > 11) continue;
      monthly[idx] = bucket.distance_m;
      if (bucket.distance_unknown_count > 0)
        unknownFrom = Math.min(unknownFrom ?? idx, idx);
      seen = true;
      if (idx > lastIdx) lastIdx = idx;
    }
    if (!seen) return null;
    let running = 0;
    return {
      year: y,
      cumulative: monthly.map((value, index) => {
        if (
          index > lastIdx ||
          (unknownFrom !== undefined && index >= unknownFrom)
        )
          return null;
        running += value;
        return running;
      }),
      lastIdx,
      unknownFrom,
    };
  }

  const current = $derived(seriesFor(year));
  const prior = $derived(seriesFor(String(Number(year) - 1)));
  const delta = $derived.by(() => {
    if (!current || !prior) return null;
    const idx = current.lastIdx;
    const cur = current.cumulative[idx];
    const previous = prior.cumulative[Math.min(idx, prior.lastIdx)];
    if (cur == null || previous == null) return null;
    const diff = cur - previous;
    return { diff, ahead: diff >= 0, month: MONTHS[idx] };
  });

  function render() {
    if (!chart || !chartContainer) return;
    const styles = getComputedStyle(chartContainer);
    const text = styles.getPropertyValue("--text").trim();
    const fontSize = chartFontSize(styles);
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const accent = styles.getPropertyValue("--sport-run").trim() || "#4f8cff";
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    const range = axisRange([...(current?.cumulative ?? []), ...(prior?.cumulative ?? [])]);
    // This scope owns the full option; merging retains removed years and legend state.
    chart.setOption({
      animation: !reducedMotion,
      animationDuration: reducedMotion ? 0 : 800,
      animationDurationUpdate: reducedMotion ? 0 : 550,
      animationEasing: "cubicOut",
      animationEasingUpdate: "cubicInOut",
      legend: {
        show: Boolean(prior),
        top: 0,
        right: 0,
        itemWidth: 12,
        itemHeight: 3,
        textStyle: { color: muted, fontSize },
        data: [year, ...(prior ? [String(Number(year) - 1)] : [])],
      },
      grid: { top: prior ? 34 : 32, right: 16, bottom: 30, left: 44 },
      tooltip: {
        trigger: "axis",
        axisPointer: {
          type: "line",
          lineStyle: { color: muted, opacity: 0.7 },
        },
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: border,
        textStyle: { color: text, fontSize },
        formatter: (
          params: Array<{
            axisValue?: string;
            seriesName: string;
            value: number | null;
            color: string;
          }>,
        ) => {
          const month = params[0]?.axisValue ?? "";
          const rows = params
            .map((item) =>
              item.value == null
                ? ""
                : `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${formatDistance(item.value)}</strong>`,
            )
            .filter(Boolean);
          return `${month}<br/>${rows.join("<br/>")}`;
        },
      },
      xAxis: {
        type: "category",
        data: MONTHS,
        boundaryGap: false,
        axisLabel: { color: muted, fontSize },
        axisLine: { lineStyle: { color: border } },
        axisTick: { lineStyle: { color: border } },
      },
      yAxis: {
        type: "value",
        ...range,
        name: "km",
        nameTextStyle: { color: muted, fontSize },
        axisLabel: {
          color: muted,
          fontSize,
          formatter: (value: number) => formatAxisTick(value, range.interval, 0.001),
        },
        axisLine: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.6 } },
      },
      series: [
        ...(prior
          ? [
              {
                name: String(Number(year) - 1),
                type: "line",
                data: prior.cumulative,
                showSymbol: false,
                connectNulls: false,
                smooth: 0.18,
                lineStyle: { color: muted, width: 1.5, type: "dashed" },
                itemStyle: { color: muted },
                emphasis: { focus: "series", lineStyle: { width: 2 } },
              },
            ]
          : []),
        ...(current
          ? [
              {
                name: year,
                type: "line",
                data: current.cumulative,
                showSymbol: false,
                connectNulls: false,
                smooth: 0.18,
                lineStyle: { color: accent, width: 2.5 },
                itemStyle: { color: accent },
                areaStyle: {
                  color: {
                    type: "linear",
                    x: 0,
                    y: 0,
                    x2: 0,
                    y2: 1,
                    colorStops: [
                      { offset: 0, color: accent },
                      { offset: 1, color: "transparent" },
                    ],
                  },
                  opacity: 0.35,
                },
                emphasis: { focus: "series", lineStyle: { width: 3 } },
              },
            ]
          : []),
      ],
    }, { notMerge: true });
  }

  onMount(() => {
    if (!chartContainer) return;
    chart = init(chartContainer, undefined, { renderer: "canvas" });
    attachChartInstance(chartContainer, chart);
    render();
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(chartContainer);
    const stopPresentation = observeChartPresentation(render);
    return () => {
      stopPresentation();
      resize.disconnect();
      attachChartInstance(chartContainer, undefined);
      chart?.dispose();
    };
  });

  let lastSignature: string | undefined;
  $effect(() => {
    const next = chartSignature(byMonth, year, current, prior);
    if (next === lastSignature) return;
    lastSignature = next;
    render();
  });
</script>

<div class="year-progress" class:tile={!embedded} class:embedded>
  <div class="header">
    <div class="title">
      Cumulative {sportName ? `${sportName.toLowerCase()} ` : ""}distance — {year}
    </div>
    {#if delta}
      <div class="delta" class:ahead={delta.ahead} class:behind={!delta.ahead}>
        <span class="arrow">{delta.ahead ? "▲" : "▼"}</span>{formatDistance(
          Math.abs(delta.diff),
        )}<span class="delta-sub"
          >{delta.ahead ? "ahead of" : "behind"}
          {Number(year) - 1} at {delta.month}</span
        >
      </div>
    {/if}
  </div>
  {#if !current}
    <p class="muted">No distance recorded for {year}.</p>
  {:else}
    <div
      class="chart"
      bind:this={chartContainer}
      role="img"
      aria-label={`Cumulative distance for ${year}`}
    ></div>
    {#if current.unknownFrom !== undefined}
      <p class="muted">
        Cumulative distance is unavailable from {MONTHS[current.unknownFrom]} because
        one or more activity distances are unknown.
      </p>
    {/if}
    <details class="chart-data">
      <summary>View cumulative distance data</summary>
      <table>
        <caption>Cumulative distance by month</caption>
        <thead>
          <tr>
            <th scope="col">Month</th>
            <th scope="col">{year}</th>
            {#if prior}<th scope="col">{Number(year) - 1}</th>{/if}
          </tr>
        </thead>
        <tbody>
          {#each MONTHS as month, index}
            <tr>
              <th scope="row">{month}</th>
              <td
                >{current?.unknownFrom !== undefined &&
                index >= current.unknownFrom
                  ? "Distance unknown"
                  : current?.cumulative[index] == null
                    ? "No observation"
                    : formatDistance(current.cumulative[index]!)}</td
              >
              {#if prior}<td
                  >{prior.unknownFrom !== undefined &&
                  index >= prior.unknownFrom
                    ? "Distance unknown"
                    : prior.cumulative[index] == null
                      ? "No observation"
                      : formatDistance(prior.cumulative[index]!)}</td
                >{/if}
            </tr>
          {/each}
        </tbody>
      </table>
    </details>
  {/if}
</div>

<style>
  .year-progress {
    min-width: 0;
    padding: var(--space-4);
  }
  .year-progress.embedded { padding: 0; }
  .header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-2);
  }
  .title {
    font-size: var(--type-label);
    color: var(--text-muted);
  }
  .delta {
    display: inline-flex;
    align-items: baseline;
    gap: var(--space-1);
    font-size: var(--type-label);
    font-weight: 700;
  }
  .delta.ahead {
    color: var(--sport-walk);
  }
  .delta.behind {
    color: var(--sport-other);
  }
  .delta .arrow {
    font-size: var(--type-caption);
  }
  .delta-sub {
    font-size: var(--type-caption);
    font-weight: 500;
    color: var(--text-muted);
  }
  .chart-data {
    margin-top: var(--space-2);
    color: var(--text-muted);
    font-size: var(--type-label);
  }
  .chart-data summary {
    width: fit-content;
    cursor: pointer;
  }
  table {
    width: 100%;
    margin-top: var(--space-2);
    border-collapse: collapse;
    color: var(--text);
  }
  th,
  td {
    padding: var(--space-2);
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
  .chart {
    width: 100%;
    height: 260px;
  }
  @media (max-width: 640px) {
    .header {
      align-items: flex-start;
      flex-direction: column;
      gap: var(--space-1);
    }
    .chart {
      height: 230px;
    }
  }
</style>
