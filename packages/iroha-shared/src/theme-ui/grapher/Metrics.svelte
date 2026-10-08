<script lang="ts">
  import type { MetricsThemeProps } from "../../view-contracts/metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="grapher-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .grapher-metrics {
    --accent: var(--sport-run);
    padding-top: 0.7rem;
    border-top: 3px solid var(--accent);
    font-variant-numeric: tabular-nums;
  }

  .grapher-metrics :global(.metric-panel) {
    padding: var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow);
  }
</style>
