<script lang="ts">
  import type { MetricsThemeProps } from "../../view-contracts/metrics-view";

  let { children, theme }: MetricsThemeProps = $props();
</script>

<div class="atlas-metrics" data-theme={theme}>{@render children?.()}</div>

<style>
  .atlas-metrics {
    padding-left: 0.8rem;
    border-left: 3px solid var(--accent);
    background-image: linear-gradient(
      90deg,
      color-mix(in srgb, var(--accent) 7%, transparent) 1px,
      transparent 1px
    );
    background-size: 18px 100%;
  }
</style>
