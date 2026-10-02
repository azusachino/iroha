<script lang="ts">
  let {
    unit,
    method,
    coverage,
    sourceKinds = [],
  }: {
    unit: string;
    method: string;
    coverage?: { expected_periods: number; observed_periods: number };
    sourceKinds?: string[];
  } = $props();
</script>

<dl class="metric-metadata">
  <div><dt>Unit</dt><dd>{unit}</dd></div>
  <div><dt>Method</dt><dd class="mono">{method}</dd></div>
  {#if coverage}
    <div>
      <dt>Coverage</dt>
      <dd>{coverage.observed_periods}/{coverage.expected_periods} periods</dd>
    </div>
  {/if}
  {#if sourceKinds.length}
    <div><dt>Sources</dt><dd>{sourceKinds.join(", ")}</dd></div>
  {/if}
</dl>

<style>
  .metric-metadata {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-4);
    margin: 0;
    color: var(--text-muted, #778090);
    font-size: var(--type-caption);
  }

  .metric-metadata div { display: inline-flex; gap: var(--space-1); }
  .metric-metadata dt { font-weight: 700; }
  .metric-metadata dd { margin: 0; font-variant-numeric: tabular-nums; }
  .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>
