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
    padding: var(--pad-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--tile-surface, var(--surface));
    box-shadow: var(--tile-shadow), var(--tile-rim);
  }

  .panel-frame :global(h2) {
    font-size: var(--type-title-small);
  }
</style>
