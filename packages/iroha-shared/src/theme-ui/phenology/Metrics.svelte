<script lang="ts">
  import type { MetricsThemeProps } from "../../view-contracts/metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="phenology-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .phenology-metrics {
    padding: 1rem;
    border-radius: 1.5rem;
    background: color-mix(in srgb, var(--accent) 5%, transparent);
  }
</style>
