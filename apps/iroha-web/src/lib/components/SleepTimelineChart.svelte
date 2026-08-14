<script lang="ts">
  import { onMount } from "svelte";
  import { BarChart } from "echarts/charts";
  import { GridComponent, TooltipComponent } from "echarts/components";
  import { init, use } from "echarts/core";
  import { CanvasRenderer } from "echarts/renderers";
  import type { ECharts } from "echarts/core";
  import { sleepStageColor, sleepStageLabel } from "$lib/sleep-stages";

  use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

  type Segment = { stage: string; started_at: string; ended_at: string };

  let {
    segments,
    sessionKind = "main",
  }: { segments: Segment[]; sessionKind?: "main" | "nap" } = $props();
  let container: HTMLDivElement;
  let chart: ECharts | undefined;

  const sessionLabel = $derived(sessionKind === "nap" ? "Nap" : "Main sleep");

  function duration(segment: Segment): number {
    return Math.max(
      0,
      (new Date(segment.ended_at).getTime() -
        new Date(segment.started_at).getTime()) /
        1000,
    );
  }

  function formatDuration(seconds: number): string {
    const minutes = Math.round(seconds / 60);
    return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
  }

  // The chart itself has no visible axis (it's a single stacked row, not a
  // series to read point-by-point) -- without this, the only way to learn
  // what a color means is to hover each segment one at a time. Totals are
  // summed per stage since the same stage can recur as separate segments
  // through the night.
  const legend = $derived.by(() => {
    const totals = new Map<string, number>();
    for (const segment of segments) {
      totals.set(
        segment.stage,
        (totals.get(segment.stage) ?? 0) + duration(segment),
      );
    }
    return [...totals.entries()]
      .sort((a, b) => b[1] - a[1])
      .map(([stage, seconds]) => ({
        stage,
        label: sleepStageLabel(stage),
        color: sleepStageColor(stage),
        seconds,
      }));
  });

  function render() {
    if (!chart) return;
    const styles = getComputedStyle(container);
    const total = segments.reduce((sum, segment) => sum + duration(segment), 0);
    chart.setOption({
      animationDuration: 550,
      animationDurationUpdate: 450,
      animationEasing: "cubicOut",
      animationEasingUpdate: "cubicInOut",
      grid: { top: 12, right: 4, bottom: 12, left: 4 },
      xAxis: { type: "value", max: Math.max(1, total), show: false },
      yAxis: { type: "category", data: [sessionLabel], show: false },
      tooltip: {
        trigger: "item",
        backgroundColor: styles.getPropertyValue("--surface-2").trim(),
        borderColor: styles.getPropertyValue("--border").trim(),
        textStyle: {
          color: styles.getPropertyValue("--text").trim(),
          fontSize: 12,
        },
        formatter: (params: { seriesName: string; value: number }) =>
          `${sleepStageLabel(params.seriesName)}<br/><strong>${formatDuration(params.value)}</strong>`,
      },
      series: segments.map((segment) => ({
        name: segment.stage,
        type: "bar",
        stack: "sleep",
        barWidth: 42,
        data: [duration(segment)],
        itemStyle: {
          color: sleepStageColor(segment.stage),
          borderColor: styles.getPropertyValue("--surface").trim(),
          borderWidth: 1,
        },
        emphasis: {
          itemStyle: {
            shadowBlur: 14,
            shadowColor: sleepStageColor(segment.stage),
          },
        },
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
    segments;
    render();
  });
</script>

<div
  class="timeline-chart"
  bind:this={container}
  aria-label={`Interactive ${sessionLabel.toLowerCase()} sleep stage timeline`}
></div>
<ul class="timeline-legend">
  {#each legend as item (item.stage)}
    <li>
      <span class="dot" style={`background:${item.color}`}></span>
      <span class="lbl">{item.label}</span>
      <span class="val">{formatDuration(item.seconds)}</span>
    </li>
  {/each}
</ul>

<style>
  .timeline-chart {
    width: 100%;
    height: 5.5rem;
  }
  .timeline-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 1rem;
    margin: 0.75rem 0 0;
    padding: 0;
    list-style: none;
  }
  .timeline-legend li {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8rem;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
  }
  .lbl {
    color: var(--text-muted);
  }
  .val {
    font-weight: 650;
  }
</style>
