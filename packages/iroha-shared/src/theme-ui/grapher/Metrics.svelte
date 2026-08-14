<script lang="ts">
  import type { MetricsThemeProps } from "../../metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="grapher-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .grapher-metrics {
    padding-top: 0.7rem;
    border-top: 3px solid var(--accent);
    font-variant-numeric: tabular-nums;
  }
</style>
