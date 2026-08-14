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
  <Shell theme={theme.language()}>{@render children()}</Shell>
{:else}
  <main class="content">{@render children()}</main>
  <footer class="footer">Private activity viewer · v{APP_VERSION}</footer>
{/if}
