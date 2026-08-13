<script lang="ts">
  let { children } = $props();
</script>

<div class="archive-metrics">{@render children?.()}</div>

<style>
  .archive-metrics {
    padding: 0.8rem;
    border: 3px double var(--border);
    border-radius: 0;
    box-shadow: 3px 3px 0 color-mix(in srgb, var(--text-muted) 22%, transparent);
  }
</style>
