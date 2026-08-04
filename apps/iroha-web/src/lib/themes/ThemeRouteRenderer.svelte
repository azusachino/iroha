<script lang="ts">
  import type { Component } from "svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import type { ThemeRoute } from "$lib/themes/types";

  let {
    route,
    props,
  }: {
    route: ThemeRoute;
    props: Record<string, unknown>;
  } = $props();

  const theme = useTheme();
  const Renderer = $derived(
    theme.definition().components?.[route] as
      Component<Record<string, unknown>> | undefined,
  );
</script>

{#if Renderer}
  {#if route === "activity-detail"}
    <p class="detail-back"><a href="/activities">← Back to Motion</a></p>
  {/if}
  <Renderer {...props} />
{/if}
