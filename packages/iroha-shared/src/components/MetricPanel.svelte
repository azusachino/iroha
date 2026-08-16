<script lang="ts">
  // The one panel contract every metric-bearing composition uses: the theme's
  // own chart, exact table parity, visible provenance, and a CSV of exactly
  // the rows displayed. Themes own hierarchy and chart grammar; this owns the
  // truthfulness chrome so it cannot drift per theme.
  import type { Snippet } from "svelte";
  import MetricMetadata from "./MetricMetadata.svelte";
  import MetricTable from "./MetricTable.svelte";
  import { panelCsv, type PanelCoverage, type PanelRow } from "./metric-panel";

  let {
    metricId,
    label,
    unit,
    method,
    coverage,
    sourceKinds = [],
    rowHeader = "Period",
    rows,
    period,
    children,
  }: {
    metricId: string;
    label: string;
    unit: string;
    method: string;
    coverage?: PanelCoverage;
    sourceKinds?: string[];
    rowHeader?: string;
    rows: PanelRow[];
    period?: string;
    children: Snippet;
  } = $props();

  let view = $state<"chart" | "table">("chart");

  function downloadCsv() {
    const blob = new Blob([panelCsv(metricId, unit, rowHeader, rows)], {
      type: "text/csv;charset=utf-8",
    });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = `${metricId}${period ? `-${period}` : ""}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
  }
</script>

<div class="metric-panel" aria-label={label} data-metric={metricId}>
  <div class="metric-panel-body">
    {#if view === "chart"}
      {@render children()}
    {:else}
      <MetricTable {rows} {unit} {rowHeader} />
    {/if}
  </div>
  <div class="metric-panel-foot">
    <MetricMetadata {unit} {method} {coverage} {sourceKinds} />
    <div class="metric-panel-actions" role="group" aria-label="{label} view">
      <button
        type="button"
        aria-pressed={view === "chart"}
        onclick={() => (view = "chart")}>Chart</button
      >
      <button
        type="button"
        aria-pressed={view === "table"}
        onclick={() => (view = "table")}>Table</button
      >
      <button type="button" onclick={downloadCsv} disabled={!rows.length}
        >CSV</button
      >
    </div>
  </div>
</div>

<style>
  .metric-panel {
    display: grid;
    gap: 0.6rem;
    min-width: 0;
  }
  .metric-panel-body {
    min-width: 0;
  }
  .metric-panel-foot {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem 1rem;
  }
  .metric-panel-actions {
    display: flex;
    gap: 0.3rem;
  }
  button {
    min-height: 1.9rem;
    padding: 0.25rem 0.6rem;
    border: 1px solid var(--border, #d6dbe3);
    border-radius: calc(var(--radius, 8px) - 4px);
    background: var(--surface-2, transparent);
    color: var(--text-muted, #778090);
    font: inherit;
    font-size: 0.68rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    cursor: pointer;
  }
  button[aria-pressed="true"] {
    border-color: var(--accent, #2f6fed);
    color: var(--accent, #2f6fed);
  }
  button:disabled {
    cursor: default;
    opacity: 0.5;
  }
</style>
