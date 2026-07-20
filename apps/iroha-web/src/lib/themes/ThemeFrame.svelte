<script lang="ts">
  import type { Snippet } from "svelte";
  import { useTheme } from "$lib/themes/context.svelte";

  let { children }: { children: Snippet } = $props();
  const theme = useTheme();
  const Shell = $derived(theme.definition().components?.shell);
</script>

{#if Shell}
  <Shell>{@render children()}</Shell>
{:else}
  <main class="content">{@render children()}</main>
  <footer class="footer">Private activity viewer</footer>
{/if}
