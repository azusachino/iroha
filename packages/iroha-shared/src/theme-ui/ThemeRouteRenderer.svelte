<script lang="ts">
  import type { Component, Snippet } from "svelte";
  import type { ThemeRoute } from "../themes";
  import { useTheme } from "./context.svelte";

  let {
    route,
    props,
    children,
  }: {
    route: ThemeRoute;
    props: Record<string, unknown>;
    children?: Snippet;
  } = $props();

  const theme = useTheme();
  const Renderer = $derived(
    theme.definition().components?.[route] as
      Component<Record<string, unknown>> | undefined,
  );
</script>

{#if Renderer}
  <Renderer {...props} theme={theme.language()} {children} />
{/if}
