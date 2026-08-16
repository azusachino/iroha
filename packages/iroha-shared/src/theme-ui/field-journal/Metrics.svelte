<script lang="ts">
  import type { MetricsThemeProps } from "../../view-contracts/metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="journal-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .journal-metrics {
    padding: 0.8rem;
    border: 1px dashed var(--border);
    box-shadow: 4px 4px 0 color-mix(in srgb, var(--accent-2) 16%, transparent);
  }
</style>
