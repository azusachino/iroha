<script lang="ts">
  import { onMount } from "svelte";
  import { init, use } from "echarts/core";
  import { PieChart } from "echarts/charts";
  import { LegendComponent, TooltipComponent } from "echarts/components";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";

  use([PieChart, LegendComponent, TooltipComponent, CanvasRenderer]);

  type Stage = { name: string; value: number; color: string };

  let {
    stages,
    selectedStage,
    onStageSelect,
    onStageHover,
  }: {
    stages: Stage[];
    selectedStage: string;
    onStageSelect?: (stage: string) => void;
    onStageHover?: (stage: string | null) => void;
  } = $props();

  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    chart.setOption({
      animation: !reducedMotion,
      tooltip: {
        trigger: "item",
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: styles.getPropertyValue("--border").trim(),
        textStyle: { color: styles.getPropertyValue("--text").trim() },
        formatter: (params: { name: string; value: number; percent: number }) =>
          `${params.name}<br/><strong>${formatMinutes(params.value)}</strong> · ${params.percent}%`,
      },
      legend: {
        show: true,
        bottom: 0,
        left: "center",
        itemWidth: 8,
        itemHeight: 8,
        itemGap: 8,
        textStyle: {
          color: styles.getPropertyValue("--text-muted").trim(),
          fontSize: 10,
        },
        data: stages.map((stage) => stage.name),
      },
      series: [
        {
          type: "pie",
          radius: ["48%", "70%"],
          center: ["50%", "42%"],
          avoidLabelOverlap: true,
          itemStyle: {
            borderColor: styles.getPropertyValue("--surface").trim(),
            borderWidth: 3,
          },
          label: { show: false },
          emphasis: { scale: true, scaleSize: 5, label: { show: false } },
          animationType: "expansion",
          animationDuration: reducedMotion ? 0 : 650,
          animationEasing: "cubicOut",
          animationDurationUpdate: reducedMotion ? 0 : 500,
          animationEasingUpdate: "cubicInOut",
          data: stages.map((stage) => ({
            name: stage.name,
            value: stage.value,
            itemStyle: { color: stage.color },
          })),
        },
      ],
    });
    chart.dispatchAction({
      type: "highlight",
      seriesIndex: 0,
      name: selectedStage,
    });
  }

  function formatMinutes(seconds: number): string {
    const minutes = Math.round(seconds / 60);
    return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
  }

  onMount(() => {
    chart = init(container, undefined, { renderer: "canvas" });
    chart.on("click", (params) => onStageSelect?.(String(params.name)));
    chart.on("mouseover", (params) => onStageHover?.(String(params.name)));
    chart.on("mouseout", () => onStageHover?.(null));
    render();
    const resize = new ResizeObserver(() => chart?.resize());
    resize.observe(container);
    return () => {
      resize.disconnect();
      chart?.dispose();
    };
  });

  $effect(() => {
    stages;
    selectedStage;
    render();
  });
</script>

<div
  class="architecture-chart"
  bind:this={container}
  role="img"
  aria-label="Interactive sleep stage composition. Exact stage totals are available in the sleep data table."
></div>
<details class="chart-data">
  <summary>View sleep stage data</summary>
  <table>
    <caption>Sleep stage composition</caption>
    <thead>
      <tr><th scope="col">Stage</th><th scope="col">Duration</th></tr>
    </thead>
    <tbody>
      {#each stages as stage}
        <tr><th scope="row">{stage.name}</th><td>{formatMinutes(stage.value)}</td></tr>
      {/each}
    </tbody>
  </table>
</details>

<style>
  .architecture-chart {
    width: 11rem;
    height: 11rem;
    flex: 0 0 11rem;
  }
  @media (max-width: 640px) {
    .architecture-chart {
      width: 9rem;
      height: 9rem;
      flex-basis: 9rem;
    }
  }
  .chart-data {
    width: min(100%, 18rem);
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .chart-data summary {
    width: fit-content;
    cursor: pointer;
  }
  table {
    width: 100%;
    margin-top: 0.5rem;
    border-collapse: collapse;
    color: var(--text);
  }
  th,
  td {
    padding: 0.35rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  thead th {
    color: var(--text-muted);
    font-weight: 650;
  }
</style>
