<script lang="ts">
  let { children } = $props();
</script>

<div class="sound-metrics">{@render children?.()}</div>

<style>
  .sound-metrics {
    padding-inline: 0.8rem;
    border-inline: 3px solid var(--accent-2);
    box-shadow: inset 0 -3px 0
      color-mix(in srgb, var(--accent) 28%, transparent);
  }
</style>
