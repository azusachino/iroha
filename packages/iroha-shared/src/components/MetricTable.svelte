<script lang="ts">
  import type { PanelRow } from "./metric-panel";

  let {
    rows,
    unit,
    rowHeader = "Period",
  }: {
    rows: PanelRow[];
    unit: string;
    rowHeader?: string;
  } = $props();

  const hasBreakdown = $derived(rows.some((row) => row.breakdown));
  const hasObserved = $derived(rows.some((row) => row.observed != null));
</script>

<div class="metric-table-wrap">
  <table>
    <caption class="sr-only">Exact values</caption>
    <thead
      ><tr
        ><th>{rowHeader}</th>{#if hasBreakdown}<th>Breakdown</th>{/if}<th
          >Value ({unit})</th
        >{#if hasObserved}<th>Observed days</th>{/if}</tr
      ></thead
    >
    <tbody>
      {#each rows as row}
        <tr>
          <th scope="row">{row.label}</th>
          {#if hasBreakdown}<td>{row.breakdown ?? "—"}</td>{/if}
          <td>{row.display}</td>
          {#if hasObserved}<td>{row.observed ?? "—"}</td>{/if}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .metric-table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; color: var(--text, #1b2430); font-size: 0.78rem; }
  th, td { padding: 0.55rem 0.65rem; border-bottom: 1px solid var(--border, #d6dbe3); text-align: left; }
  thead th { color: var(--text-muted, #778090); font-size: 0.68rem; letter-spacing: 0.06em; text-transform: uppercase; }
  tbody th { font-weight: 650; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
</style>
