<script lang="ts">
  import { onMount } from "svelte";
  import { LineChart as EchartsLineChart } from "echarts/charts";
  import {
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    TooltipComponent,
  } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([
    EchartsLineChart,
    DataZoomComponent,
    GridComponent,
    LegendComponent,
    TooltipComponent,
    CanvasRenderer,
  ]);

  export interface ChartSeries {
    label: string;
    values: (number | null)[];
    stroke: string;
  }

  let {
    title,
    xValues,
    xLabel,
    series,
  }: {
    title: string;
    xValues: number[];
    xLabel: string;
    series: ChartSeries[];
  } = $props();

  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const text = styles.getPropertyValue("--text").trim();
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    chart.setOption({
      animationDuration: 700,
      animationDurationUpdate: 450,
      animationEasing: "cubicOut",
      animationEasingUpdate: "cubicInOut",
      title: {
        text: title,
        left: 0,
        top: 0,
        textStyle: { color: text, fontSize: 14, fontWeight: 650 },
      },
      legend: {
        show: series.length > 1,
        top: 0,
        right: 0,
        itemWidth: 10,
        itemHeight: 3,
        textStyle: { color: muted, fontSize: 11 },
        data: series.map((item) => item.label),
      },
      grid: {
        top: series.length > 1 ? 42 : 30,
        right: 16,
        bottom: 56,
        left: 48,
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross", crossStyle: { color: muted } },
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
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
                `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${formatValue(item.value[1])}</strong>`,
            );
          return `${xLabel}: ${formatXAxis(x, xLabel)}<br/>${rows.join("<br/>")}`;
        },
      },
      xAxis: {
        type: "value",
        name: xLabel,
        nameLocation: "middle",
        nameGap: 28,
        axisLabel: {
          color: muted,
          fontSize: 10,
          formatter: (value: number) => formatXAxis(value, xLabel),
        },
        axisLine: { lineStyle: { color: border } },
        axisTick: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.65 } },
      },
      yAxis: {
        axisLabel: { color: muted, fontSize: 10 },
        axisLine: { lineStyle: { color: border } },
        axisTick: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.65 } },
      },
      dataZoom: [
        {
          type: "inside",
          xAxisIndex: 0,
          zoomOnMouseWheel: true,
          moveOnMouseMove: true,
        },
        {
          type: "slider",
          xAxisIndex: 0,
          height: 14,
          bottom: 4,
          borderColor: border,
          backgroundColor: styles.getPropertyValue("--surface-2").trim(),
          fillerColor: "rgba(92, 141, 255, 0.18)",
          textStyle: { color: muted, fontSize: 9 },
        },
      ],
      series: series.map((item) => ({
        name: item.label,
        type: "line",
        showSymbol: false,
        smooth: 0.16,
        connectNulls: false,
        lineStyle: { color: item.stroke, width: 2 },
        itemStyle: { color: item.stroke },
        emphasis: { focus: "series", lineStyle: { width: 3 } },
        data: xValues.map((x, index) => [x, item.values[index] ?? null]),
      })),
    });
  }

  function formatXAxis(value: number | undefined, label: string): string {
    if (value == null || !Number.isFinite(value)) return "";
    if (label.toLowerCase().includes("distance"))
      return `${value.toFixed(value < 10 ? 1 : 0)} km`;
    if (label.toLowerCase().includes("time"))
      return `${value.toFixed(value < 10 ? 1 : 0)} min`;
    return Number.isInteger(value) ? String(value) : value.toFixed(1);
  }

  function formatValue(value: number | null): string {
    if (value == null) return "—";
    return Number.isInteger(value) ? String(value) : value.toFixed(1);
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
    title;
    xValues;
    xLabel;
    series;
    render();
  });
</script>

<div class="chart" bind:this={container} aria-label={`${title} chart`}></div>

<style>
  .chart {
    width: 100%;
    height: 280px;
    min-height: 220px;
  }
</style>
