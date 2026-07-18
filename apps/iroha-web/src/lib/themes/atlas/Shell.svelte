<script lang="ts">
  import type { Snippet } from "svelte";
  import ThemeShell from "$lib/themes/ThemeShell.svelte";
  let { children }: { children: Snippet } = $props();
</script>

<ThemeShell variant="atlas">{@render children()}</ThemeShell>
