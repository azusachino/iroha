<script lang="ts">
  // Generic themed ECharts bar chart for the per-theme Daily/Sleep pattern
  // views. Those were previously hand-rolled div/SVG bars with no chart
  // library backing -- no hover, no tooltip, and (on the Sleep page) an
  // x-axis that truncated every label to an identical, unreadable prefix.
  // This gives every theme the same real interactivity; per-theme color and
  // orientation keep some of each theme's own identity.
  import { onMount } from "svelte";
  import { BarChart as EchartsBarChart, LineChart } from "echarts/charts";
  import {
    GridComponent,
    LegendComponent,
    TooltipComponent,
  } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([
    EchartsBarChart,
    LineChart,
    GridComponent,
    LegendComponent,
    TooltipComponent,
    CanvasRenderer,
  ]);

  export interface BarSeries {
    name: string;
    values: (number | null)[];
    color?: string;
    formatter?: (value: number) => string;
  }

  let {
    categories,
    primary,
    secondary,
    orientation = "vertical",
    primaryType = "bar",
    activeIndex = null,
    onBarClick,
    height = 260,
  }: {
    categories: string[];
    primary: BarSeries;
    secondary?: BarSeries;
    orientation?: "vertical" | "horizontal";
    primaryType?: "bar" | "line";
    activeIndex?: number | null;
    onBarClick?: (index: number) => void;
    height?: number;
  } = $props();

  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function defaultFormatter(value: number): string {
    return Number.isInteger(value) ? String(value) : value.toFixed(1);
  }

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const text = styles.getPropertyValue("--text").trim();
    const muted = styles.getPropertyValue("--text-muted").trim() || "#9aa3b2";
    const border = styles.getPropertyValue("--border").trim() || "#2a2f3a";
    const accent = styles.getPropertyValue("--accent").trim() || "#5c8dff";
    const primaryColor = primary.color || accent;
    const primaryFormat = primary.formatter || defaultFormatter;
    const secondaryFormat = secondary?.formatter || defaultFormatter;
    const categoryAxis = {
      type: "category" as const,
      data: categories,
      axisLabel: { color: muted, fontSize: 10 },
      axisLine: { lineStyle: { color: border } },
      axisTick: { show: false },
    };
    const valueAxis = {
      type: "value" as const,
      axisLabel: {
        color: muted,
        fontSize: 10,
        formatter: (value: number) => primaryFormat(value),
      },
      axisLine: { lineStyle: { color: border } },
      splitLine: { lineStyle: { color: border, opacity: 0.5 } },
    };

    chart.setOption({
      animation: !window.matchMedia("(prefers-reduced-motion: reduce)").matches,
      animationDuration: window.matchMedia("(prefers-reduced-motion: reduce)")
        .matches
        ? 0
        : 500,
      animationEasing: "cubicOut",
      grid:
        orientation === "horizontal"
          ? { top: 12, right: 24, bottom: 12, left: 90 }
          : { top: 12, right: secondary ? 44 : 16, bottom: 32, left: 48 },
      legend: {
        show: !!secondary,
        top: 0,
        right: 0,
        itemWidth: 10,
        itemHeight: 3,
        textStyle: { color: muted, fontSize: 11 },
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "shadow" },
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: border,
        textStyle: { color: text, fontSize: 11 },
        formatter: (
          params: Array<{
            axisValue: string;
            seriesName: string;
            value: number;
            color: string;
          }>,
        ) => {
          const rows = params.map((item) => {
            const fmt =
              item.seriesName === primary.name
                ? primaryFormat
                : secondaryFormat;
            return `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${fmt(item.value)}</strong>`;
          });
          return `${params[0]?.axisValue}<br/>${rows.join("<br/>")}`;
        },
      },
      xAxis: orientation === "horizontal" ? valueAxis : categoryAxis,
      yAxis: orientation === "horizontal" ? categoryAxis : valueAxis,
      series: [
        {
          name: primary.name,
          type: primaryType,
          data: primary.values.map((value, index) => ({
            value,
            itemStyle: {
              color: primaryColor,
              opacity: activeIndex == null || activeIndex === index ? 1 : 0.45,
            },
          })),
          barMaxWidth: 28,
          smooth: primaryType === "line" ? 0.25 : undefined,
          showSymbol: primaryType === "line",
          symbolSize: primaryType === "line" ? 7 : undefined,
          lineStyle:
            primaryType === "line"
              ? { color: primaryColor, width: 3 }
              : undefined,
          areaStyle:
            primaryType === "line"
              ? { color: primaryColor, opacity: 0.12 }
              : undefined,
          label:
            orientation === "horizontal" && primaryType === "bar"
              ? {
                  show: true,
                  position: "right" as const,
                  color: text,
                  fontSize: 10,
                  formatter: (params: { value: number }) =>
                    primaryFormat(params.value),
                }
              : undefined,
          itemStyle: {
            color: primaryColor,
            borderRadius:
              primaryType === "bar"
                ? orientation === "horizontal"
                  ? [0, 7, 7, 0]
                  : [7, 7, 0, 0]
                : undefined,
          },
          emphasis: { itemStyle: { color: primaryColor } },
        },
        ...(secondary
          ? [
              {
                name: secondary.name,
                type: "line" as const,
                data: secondary.values,
                showSymbol: false,
                lineStyle: {
                  color: secondary.color || muted,
                  width: 2,
                  type: "dashed" as const,
                },
                itemStyle: { color: secondary.color || muted },
              },
            ]
          : []),
      ],
    });
  }

  onMount(() => {
    chart = init(container, undefined, { renderer: "canvas" });
    render();
    if (onBarClick) {
      chart.on(
        "click",
        (params: { componentType: string; dataIndex: number }) => {
          if (params.componentType === "series") onBarClick(params.dataIndex);
        },
      );
    }
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(container);
    return () => {
      resize.disconnect();
      chart?.dispose();
    };
  });

  $effect(() => {
    categories;
    primary;
    secondary;
    orientation;
    primaryType;
    activeIndex;
    render();
  });
</script>

<div
  class="bar-chart"
  style={`--h:${height}px`}
  bind:this={container}
  role="img"
  aria-label={`${primary.name} chart`}
></div>

<style>
  .bar-chart {
    width: 100%;
    height: var(--h);
  }
</style>
