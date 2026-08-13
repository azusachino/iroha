<script lang="ts">
  let { children } = $props();
</script>

<div class="grapher-metrics">{@render children?.()}</div>

<style>
  .grapher-metrics {
    padding-top: 0.7rem;
    border-top: 3px solid var(--accent);
    font-variant-numeric: tabular-nums;
  }
</style>
