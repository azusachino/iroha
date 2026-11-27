<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    text,
    children,
  }: {
    text?: string;
    children?: Snippet;
  } = $props();
</script>

<p class="empty-note">
  {#if children}
    {@render children()}
  {:else if text}
    {text}
  {/if}
</p>

<style>
  .empty-note {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
  }
</style>
