<script lang="ts">
  import type { MetricsThemeProps } from "../../view-contracts/metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="archive-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .archive-metrics {
    padding: 0.8rem;
    border: 3px double var(--border);
    border-radius: 0;
    box-shadow: 3px 3px 0 color-mix(in srgb, var(--text-muted) 22%, transparent);
  }
</style>
