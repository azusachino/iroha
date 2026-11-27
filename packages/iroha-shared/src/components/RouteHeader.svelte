<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    context,
    titleSnippet,
    children,
    actions,
  }: {
    title: string;
    context: string;
    titleSnippet?: Snippet;
    children?: Snippet;
    actions?: Snippet;
  } = $props();
</script>

<header class="route-header" role="region" aria-label="{title} header">
  <div class="route-copy">
    {#if titleSnippet}
      {@render titleSnippet()}
    {:else}
      <h1>{title}</h1>
    {/if}
    <p>{context}</p>
  </div>
  {#if children}<div class="route-controls">{@render children()}</div>{/if}
  {#if actions}<div class="route-actions">{@render actions()}</div>{/if}
</header>

<style>
  .route-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-3) var(--gutter-page);
    padding-block: var(--space-2) var(--space-10);
    border-bottom: 1px solid var(--border);
    min-width: 0;
  }
  .route-copy {
    flex: 0 1 auto;
    max-width: 100%;
    min-width: 0;
  }
  h1,
  .route-copy :global(h1) {
    margin: 0;
    font-size: var(--grapher-utility-title-size, var(--type-display));
    line-height: 1.15;
    overflow-wrap: anywhere;
  }
  p {
    margin: var(--space-1) 0 0;
    font-size: var(--type-label);
    color: var(--text-muted);
    overflow-wrap: anywhere;
  }
  .route-controls { flex: 1 1 auto; min-width: 0; }
  .route-actions { flex: 0 0 auto; min-width: 0; margin-left: auto; }
  @media (max-width: 640px) {
    .route-copy { flex-basis: 100%; }
    .route-controls { flex-basis: 100%; }
  }
</style>
