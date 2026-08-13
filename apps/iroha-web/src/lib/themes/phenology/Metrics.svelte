<script lang="ts">
  let { children } = $props();
</script>

<div class="phenology-metrics">{@render children?.()}</div>

<style>
  .phenology-metrics {
    padding: 1rem;
    border-radius: 1.5rem;
    background: color-mix(in srgb, var(--accent) 5%, transparent);
  }
</style>
