<script lang="ts">
  import { onMount } from "svelte";
  import { BarChart } from "echarts/charts";
  import { GridComponent, TooltipComponent } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";
  import { formatDistance } from "../format/format";
  import { axisRange, formatAxisTick } from "../format/axis";
  import { chartFontSize } from "../theme/chart-typography";
  import { observeChartPresentation } from "../theme/chart-presentation";

  use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

  type Metric = "distance_m" | "activity_count";

  let {
    points,
    metric,
    year,
  }: {
    points: { label: string; value: number }[];
    metric: Metric;
    year: string;
  } = $props();

  let chartContainer = $state<HTMLDivElement>();
  let chart: ECharts | undefined;

  function render() {
    if (!chart || !chartContainer) return;
    const styles = getComputedStyle(chartContainer);
    const text = styles.getPropertyValue("--text").trim();
    const fontSize = chartFontSize(styles);
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const accent = styles.getPropertyValue("--accent").trim() || "#39c5bb";
    const distance = metric === "distance_m";
    const range = axisRange(points.map((point) => point.value), distance ? 0 : 1);

    chart.setOption({
      animation: !window.matchMedia("(prefers-reduced-motion: reduce)").matches,
      animationDuration: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 550,
      animationEasing: "cubicOut",
      grid: { top: 32, right: 12, bottom: 30, left: 48 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "shadow" },
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: border,
        textStyle: { color: text, fontSize },
        formatter: (
          params: Array<{
            axisValue?: string;
            value: number;
            color: string;
          }>,
        ) => {
          const item = params[0];
          if (!item) return "";
          const value = distance
            ? formatDistance(item.value)
            : `${item.value.toLocaleString()} activities`;
          return `${item.axisValue}<br/><span style="color:${item.color}">●</span> ${value}`;
        },
      },
      xAxis: {
        type: "category",
        data: points.map((point) => point.label),
        axisLabel: { color: muted, fontSize },
        axisLine: { lineStyle: { color: border } },
        axisTick: { lineStyle: { color: border } },
      },
      yAxis: {
        type: "value",
        ...range,
        name: distance ? "km" : "activities",
        nameTextStyle: { color: muted, fontSize },
        axisLabel: {
          color: muted,
          fontSize,
          formatter: (value: number) =>
            formatAxisTick(value, range.interval, distance ? 0.001 : 1),
        },
        axisLine: { lineStyle: { color: border } },
        splitLine: { lineStyle: { color: border, opacity: 0.6 } },
      },
      series: [
        {
          type: "bar",
          data: points.map((point) => point.value),
          barMaxWidth: 28,
          itemStyle: {
            color: accent,
            borderRadius: [5, 5, 0, 0],
          },
          emphasis: {
            itemStyle: { color: accent },
          },
        },
      ],
    });
  }

  onMount(() => {
    if (!chartContainer) return;
    chart = init(chartContainer, undefined, { renderer: "canvas" });
    render();
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(chartContainer);
    const stopPresentation = observeChartPresentation(render);
    return () => {
      resize.disconnect();
      stopPresentation();
      chart?.dispose();
    };
  });

  $effect(() => {
    points;
    metric;
    year;
    render();
  });
</script>

<section class="month-chart tile">
  <div class="month-chart-head">
    Monthly {metric === "distance_m" ? "distance" : "activities"} — {year}
  </div>
  <div
    class="chart"
    bind:this={chartContainer}
    role="img"
    aria-label={`Monthly ${metric === "distance_m" ? "distance" : "activity count"} for ${year}`}
  ></div>
</section>

<style>
  .month-chart {
    min-width: 0;
    padding: var(--space-4);
  }

  .month-chart-head {
    margin-bottom: 0.25rem;
    color: var(--text-muted);
    font-size: var(--type-label);
  }

  .chart {
    width: 100%;
    height: 220px;
  }
</style>
