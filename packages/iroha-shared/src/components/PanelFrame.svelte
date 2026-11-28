<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    label,
    title,
    headingLevel = 2,
    meta,
    children,
  }: {
    label?: string;
    title?: string;
    headingLevel?: 2 | 3;
    meta?: Snippet;
    children: Snippet;
  } = $props();

  const headingId = $props.id();
</script>

<section
  class="panel-frame"
  aria-labelledby={title ? headingId : undefined}
  aria-label={title ? undefined : label}
>
  {#if title}
    <header class="panel-frame-header">
      {#if headingLevel === 3}
        <h3 id={headingId} class="panel-frame-title">{title}</h3>
      {:else}
        <h2 id={headingId} class="panel-frame-title">{title}</h2>
      {/if}
      {#if meta}
        <div class="panel-frame-meta">{@render meta()}</div>
      {/if}
    </header>
  {/if}
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

  .panel-frame-header {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-2) var(--space-3);
  }

  .panel-frame-title,
  .panel-frame :global(h2) {
    margin: 0;
    font-size: var(--type-title-small);
    font-weight: 700;
    line-height: 1.25;
  }

  .panel-frame-meta {
    margin-left: auto;
    color: var(--text-muted);
    font-size: var(--type-caption);
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
</style>
