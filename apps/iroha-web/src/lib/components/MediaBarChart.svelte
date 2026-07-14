<script lang="ts">
  import { onMount } from "svelte";
  import { BarChart } from "echarts/charts";
  import { GridComponent, TooltipComponent } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

  let {
    labels,
    values,
    color = "--accent",
    horizontal = false,
    height = "12rem",
    valueSuffix = "",
  }: {
    labels: (string | number)[];
    values: number[];
    color?: string;
    horizontal?: boolean;
    height?: string;
    valueSuffix?: string;
  } = $props();

  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function token(styles: CSSStyleDeclaration, name: string, fallback: string) {
    return styles.getPropertyValue(name).trim() || fallback;
  }

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const muted = token(styles, "--text-muted", "#9aa3b2");
    const border = token(styles, "--border", "#2a2f3a");
    const text = token(styles, "--text", "#e6e8ec");
    const surface2 = token(styles, "--surface-2", "#1f242d");
    const bar = token(styles, color, color);

    const category = {
      type: "category" as const,
      data: labels,
      axisLabel: { color: muted, fontSize: 10 },
      axisLine: { lineStyle: { color: border } },
      axisTick: { show: false },
    };
    const value = {
      type: "value" as const,
      axisLabel: { color: muted, fontSize: 10 },
      splitLine: { lineStyle: { color: border, opacity: 0.28 } },
    };

    chart.setOption(
      {
        animationDuration: 500,
        animationEasing: "cubicOut",
        grid: { left: 2, right: 10, top: 12, bottom: 2, containLabel: true },
        tooltip: {
          trigger: "axis",
          axisPointer: { type: "shadow" },
          backgroundColor: surface2,
          borderColor: border,
          textStyle: { color: text, fontSize: 11 },
          valueFormatter: (v: number) => `${v}${valueSuffix}`,
        },
        xAxis: horizontal ? value : category,
        yAxis: horizontal ? { ...category, inverse: true } : value,
        series: [
          {
            type: "bar",
            data: values,
            barMaxWidth: horizontal ? 15 : 30,
            itemStyle: {
              color: bar,
              borderRadius: horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0],
            },
          },
        ],
      },
      { notMerge: true },
    );
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
    values;
    color;
    horizontal;
    render();
  });
</script>

<div
  class="media-bar-chart"
  bind:this={container}
  style={`height:${height}`}
></div>

<style>
  .media-bar-chart {
    width: 100%;
  }
</style>
