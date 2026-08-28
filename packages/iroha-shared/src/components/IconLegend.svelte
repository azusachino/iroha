<script lang="ts">
  import type { PanelRow } from "./metric-panel";

  // A DOM strip beside an ECharts chart, not inside it -- ECharts draws
  // axis labels to canvas, so a Svelte icon component can never mount
  // there directly. This sits above the chart instead: one chip per row,
  // same icon/color/label the chart's bars already use, so the mapping
  // reads at a glance without needing the chart's own hover tooltip or a
  // trip to the TABLE tab.
  let { rows }: { rows: PanelRow[] } = $props();
</script>

{#if rows.some((row) => row.icon)}
  <ul class="icon-legend" aria-hidden="true">
    {#each rows as row}
      <li>
        {#if row.icon}
          <row.icon
            class="legend-icon"
            size={13}
            style={row.colorVar ? `color: var(${row.colorVar})` : undefined}
          />
        {/if}
        <span>{row.label}</span>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .icon-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 0.7rem;
    margin: 0 0 0.6rem;
    padding: 0;
    list-style: none;
  }

  .icon-legend li {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  :global(.legend-icon) {
    flex: 0 0 auto;
  }
</style>
