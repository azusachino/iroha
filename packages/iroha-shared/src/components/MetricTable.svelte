<script lang="ts">
  import type { Snippet } from "svelte";
  import { isOutOfRange, type PanelRow } from "./metric-panel";

  let {
    rows = [],
    unit = "",
    rowHeader = "Period",
    caption,
    ariaLabel,
    scrollLabel,
    minWidth,
    nowrap = false,
    numericAlign = false,
    dense = false,
    empty,
    children,
  }: {
    rows?: PanelRow[];
    unit?: string;
    rowHeader?: string;
    caption?: string;
    ariaLabel?: string;
    scrollLabel?: string;
    minWidth?: string;
    nowrap?: boolean;
    numericAlign?: boolean;
    dense?: boolean;
    empty?: string | Snippet;
    children?: Snippet;
  } = $props();

  const hasBreakdown = $derived(rows.some((row) => row.breakdown));
  const hasObserved = $derived(rows.some((row) => row.observed != null));
  // CoverageBars (the chart view) conveys a row's out-of-range status via
  // color and is aria-hidden -- this table is the accessible fallback, so
  // that status has to exist here as text too, not just a color a screen
  // reader can't see.
  const hasRange = $derived(rows.some((row) => row.range));

  function rangeStatus(row: PanelRow): string {
    if (!row.range || row.value == null) return "—";
    const { min, max } = row.range;
    if (row.value < min) return `${min}–${max} (below range)`;
    if (row.value > max) return `${min}–${max} (above range)`;
    return `${min}–${max} (in range)`;
  }
  const wrapLabel = $derived(scrollLabel ?? ariaLabel ?? caption ?? "Exact values");
</script>

{#if children}
  <div
    class="metric-table-wrap"
    role="region"
    tabindex="0"
    aria-label={wrapLabel}
  >
    <table
      aria-label={ariaLabel}
      class:nowrap
      class:numeric-align={numericAlign}
      class:dense
      style:--metric-table-min-width={minWidth}
    >
      {#if caption}<caption class="sr-only">{caption}</caption>{/if}
      {@render children()}
    </table>
  </div>
{:else if rows.length === 0 && empty}
  {#if typeof empty === "string"}
    <p class="metric-table-empty">{empty}</p>
  {:else}
    {@render empty()}
  {/if}
{:else}
  <div
    class="metric-table-wrap"
    role="region"
    tabindex="0"
    aria-label={wrapLabel}
  >
    <table
      aria-label={ariaLabel}
      class:nowrap
      class:numeric-align={numericAlign}
      class:dense
      style:--metric-table-min-width={minWidth}
    >
      <caption class="sr-only">{caption ?? "Exact values"}</caption>
      <thead
        ><tr
          ><th>{rowHeader}</th>{#if hasBreakdown}<th>Breakdown</th>{/if}<th
            >Value ({unit})</th
          >{#if hasRange}<th>Reference range</th>{/if}{#if hasObserved}<th
              >Observed days</th
            >{/if}</tr
        ></thead
      >
      <tbody>
        {#each rows as row}
          <tr>
            <th scope="row"
              >{#if row.icon}<row.icon
                  class="row-icon"
                  size={14}
                  aria-hidden="true"
                />{/if}{row.label}</th
            >
            {#if hasBreakdown}<td>{row.breakdown ?? "—"}</td>{/if}
            <td>{row.display}</td>
            {#if hasRange}<td class:danger={isOutOfRange(row)}
                >{rangeStatus(row)}</td
              >{/if}
            {#if hasObserved}<td>{row.observed ?? "—"}</td>{/if}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style>
  .metric-table-wrap { overflow-x: auto; border-radius: var(--radius-sm, 4px); }
  .metric-table-wrap:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 3px; }
  table {
    width: 100%;
    min-width: var(--metric-table-min-width, 0);
    border-collapse: collapse;
    color: var(--text, #1b2430);
    font-size: var(--type-label);
  }
  table.dense { font-size: var(--type-caption); }
  th, td {
    padding: var(--space-2) var(--space-3);
    border-bottom: 1px solid var(--border, #d6dbe3);
    text-align: left;
  }
  table.nowrap :is(th, td) { white-space: nowrap; }
  table.numeric-align :is(th, td):not(:first-child) {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  thead th {
    color: var(--text-muted, #778090);
    font-size: var(--type-caption);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  tbody th { font-weight: 650; }
  td.danger { color: var(--danger, #c0392b); font-weight: 650; }
  .metric-table-empty {
    margin: 0;
    color: var(--text-muted, #778090);
    font-size: var(--type-label);
  }
  :global(.row-icon) { margin-right: 0.35rem; color: var(--text-muted, #778090); vertical-align: -0.15em; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
</style>
