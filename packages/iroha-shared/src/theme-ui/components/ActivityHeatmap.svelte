<script lang="ts">
  import { onMount } from "svelte";
  import type { ActivityActiveDay } from "../../domain/activity";
  import { buildActivityHeatmap } from "./activity-heatmap";

  const WEEKDAYS = ["Mon", "", "Wed", "", "Fri", "", ""];
  const LEGEND_LEVELS = [0, 1, 2, 3, 4];

  let {
    days,
    endDay,
    title = "Activity history",
    embedded = false,
  }: {
    days: ActivityActiveDay[];
    endDay: string;
    title?: string;
    embedded?: boolean;
  } = $props();

  const model = $derived(buildActivityHeatmap(days, endDay));
  const firstDay = $derived(model.weeks.flat().find((cell) => cell.date)?.date);
  let heatmapGrid: HTMLDivElement;
  let hasHorizontalOverflow = $state(false);

  function focusHeatmapGrid() {
    heatmapGrid?.focus({ preventScroll: true });
  }

  onMount(() => {
    const updateOverflow = () => {
      hasHorizontalOverflow = heatmapGrid.scrollWidth > heatmapGrid.clientWidth + 1;
    };
    const observer = new ResizeObserver(updateOverflow);
    observer.observe(heatmapGrid);
    updateOverflow();
    return () => observer.disconnect();
  });
</script>

<section class="activity-heatmap" class:embedded aria-labelledby="activity-heatmap-title">
  <header class="heatmap-header">
    <div>
      <p class="kicker">Movement / daily density</p>
      <h2 id="activity-heatmap-title">{title}</h2>
      <p>
        {model.activeDayCount.toLocaleString()} active days ·
        {model.totalActivityCount.toLocaleString()} activity records ·
        {firstDay} to {endDay}
      </p>
    </div>
    <div class="legend" aria-label="Activity count intensity">
      <span>Less</span>
      {#each LEGEND_LEVELS as level}
        <span class="cell" data-level={level} aria-hidden="true"></span>
      {/each}
      <span>More</span>
    </div>
  </header>

  {#if hasHorizontalOverflow}
    <button
      class="scroll-hint"
      type="button"
      aria-controls="activity-heatmap-grid"
      onclick={focusHeatmapGrid}
    >
      Focus the heatmap to scroll through all 365 days.
    </button>
  {/if}
  <div
    id="activity-heatmap-grid"
    bind:this={heatmapGrid}
    class="heatmap-grid"
    role="img"
    tabindex="-1"
    aria-label={`${title}, last 365 days ending ${endDay}: ${model.activeDayCount} active days and ${model.totalActivityCount} activity records. Scroll horizontally to explore every day.`}
  >
    <div class="weekday-labels" aria-hidden="true">
      {#each WEEKDAYS as day}
        <span>{day}</span>
      {/each}
    </div>
    <div class="weeks" aria-hidden="true">
      {#each model.weeks as week, index (index)}
        <div class="week">
          {#each week as cell, cellIndex (cell.date ?? `empty-${index}-${cellIndex}`)}
            <span
              class="cell"
              class:empty={!cell.date}
              data-level={cell.level}
              title={cell.date
                ? `${cell.date}: ${cell.count} ${cell.count === 1 ? "activity" : "activities"}`
                : undefined}
            ></span>
          {/each}
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .activity-heatmap {
    min-width: 0;
    overflow: hidden;
    padding: 1.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface);
    box-shadow: var(--tile-shadow);
  }

  .activity-heatmap.embedded {
    padding: 0;
    border: 0;
    border-radius: 0;
    background: transparent;
    box-shadow: none;
  }
  .heatmap-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.1rem;
  }

  .kicker {
    margin: 0 0 0.35rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-size: 1.12rem;
    letter-spacing: -0.035em;
  }

  .heatmap-header p:last-child {
    margin-top: 0.3rem;
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  .scroll-hint {
    display: inline-flex;
    align-items: center;
    min-height: 44px;
    margin: 0 0 0.35rem;
    padding: 0 0.25rem;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font: inherit;
    font-size: 0.7rem;
    text-align: left;
    text-decoration: underline;
    text-underline-offset: 0.15em;
    cursor: pointer;
  }

  .scroll-hint:focus,
  .scroll-hint:focus-visible {
    outline: 2px solid var(--color-focus, var(--accent));
    outline-offset: 2px;
  }

  .heatmap-grid {
    display: flex;
    gap: 0.45rem;
    min-width: 0;
    overflow-x: auto;
    padding: 0.25rem 0 0.4rem;
    scrollbar-width: thin;
  }

  .weekday-labels,
  .week {
    display: grid;
    grid-template-rows: repeat(7, 0.68rem);
    gap: 0.2rem;
  }

  .weekday-labels {
    flex: 0 0 1.55rem;
    color: var(--text-muted);
    font-size: 0.58rem;
    line-height: 0.68rem;
    text-align: right;
  }

  .weeks {
    display: flex;
    flex: 1 1 auto;
    justify-content: space-between;
    gap: 0.2rem;
  }

  .cell {
    display: block;
    flex: 0 0 auto;
    width: 0.68rem;
    height: 0.68rem;
    border-radius: 2px;
    background: var(--surface-2);
  }

  .cell.empty {
    visibility: hidden;
  }

  .cell[data-level="1"] {
    background: color-mix(in srgb, var(--accent) 25%, var(--surface-2));
  }

  .cell[data-level="2"] {
    background: color-mix(in srgb, var(--accent) 45%, var(--surface-2));
  }

  .cell[data-level="3"] {
    background: color-mix(in srgb, var(--accent) 68%, var(--surface-2));
  }

  .cell[data-level="4"] {
    background: var(--accent);
  }

  .legend {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    gap: 0.28rem;
    padding-top: 0.25rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    white-space: nowrap;
  }

  @media (max-width: 640px) {
    .activity-heatmap {
      padding: 1rem;
    }

    .heatmap-header {
      flex-direction: column;
    }
  }
</style>
