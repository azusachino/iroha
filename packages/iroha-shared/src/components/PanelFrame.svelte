<script lang="ts">
  import type { Snippet } from "svelte";
  let { label, children }: { label: string; children: Snippet } = $props();
</script>

<section class="panel-frame" aria-label={label}>
  {@render children()}
</section>

<style>
  .panel-frame {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-3);
    min-width: 0;
    padding: var(--space-4);
    border: 1px solid color-mix(in srgb, var(--border) 88%, white 12%);
    border-radius: var(--radius, 14px);
    background: var(--tile-surface, var(--surface));
    box-shadow:
      var(--tile-shadow, none),
      inset 0 1px 0 rgb(255 255 255 / 0.035);
  }
</style>
