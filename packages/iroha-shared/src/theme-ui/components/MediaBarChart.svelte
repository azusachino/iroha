<script lang="ts">
  import { onMount } from "svelte";
  import { chartSignature } from "./chart-signature";
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
    title,
  }: {
    labels: (string | number)[];
    values: number[];
    color?: string;
    horizontal?: boolean;
    height?: string;
    valueSuffix?: string;
    title?: string;
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
        animation: !window.matchMedia("(prefers-reduced-motion: reduce)")
          .matches,
        animationDuration: window.matchMedia("(prefers-reduced-motion: reduce)")
          .matches
          ? 0
          : 500,
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

  let lastSignature: string | undefined;
  $effect(() => {
    const next = chartSignature(labels, values, color, horizontal);
    if (next === lastSignature) return;
    lastSignature = next;
    render();
  });
</script>

<div
  class="media-bar-chart"
  bind:this={container}
  role={title ? "img" : undefined}
  aria-label={title}
  style={`height:${height}`}
></div>
{#if title}<table aria-label={`${title} — exact counts`}>
    <caption>{title} — exact counts</caption>
    <thead
      ><tr><th scope="col">Category</th><th scope="col">Count</th></tr></thead
    >
    <tbody
      >{#each labels as label, index}<tr
          ><th scope="row">{label}</th><td>{values[index]}</td></tr
        >{/each}</tbody
    >
  </table>{/if}

<style>
  table {
    width: 100%;
    border-collapse: collapse;
    margin-top: var(--space-3);
    font-size: var(--type-label);
  }
  caption {
    text-align: left;
    padding-block: 0.5rem;
    color: var(--text-muted);
  }
  th,
  td {
    text-align: left;
    padding: 0.5rem 0.5rem;
    border-bottom: 1px solid var(--border);
  }
  th:last-child,
  td:last-child {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .media-bar-chart {
    width: 100%;
  }
</style>
