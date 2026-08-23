<script lang="ts">
  import type { Snippet } from "svelte";
  import { APP_VERSION } from "$lib/config";
  import { useTheme } from "$lib/themes/context.svelte";

  let { children }: { children: Snippet } = $props();
  const theme = useTheme();
  const Shell = $derived(
    theme.definition().components?.shell as unknown as
      import("svelte").Component<Record<string, unknown>> | undefined,
  );
</script>

{#if Shell}
  <Shell theme={theme.language()}>
    <main id="main-content" class="theme-content" tabindex="-1">
      {@render children()}
    </main>
  </Shell>
{:else}
  <main id="main-content" class="content" tabindex="-1">
    {@render children()}
  </main>
  <footer class="footer">Private activity viewer · v{APP_VERSION}</footer>
{/if}

<style>
  .theme-content,
  .content {
    min-width: 0;
    scroll-margin-top: 8rem;
  }
</style>
