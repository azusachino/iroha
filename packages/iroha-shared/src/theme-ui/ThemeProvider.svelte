<script lang="ts">
  import type { Snippet } from "svelte";
  import type { DesignLanguage } from "../themes";
  import { getThemeDefinition } from "./registry";
  import { provideTheme } from "./context.svelte";

  let {
    language,
    onSelect,
    children,
  }: {
    language: DesignLanguage;
    onSelect: (language: DesignLanguage) => void;
    children?: Snippet;
  } = $props();

  const definition = $derived(getThemeDefinition(language));

  provideTheme({
    language: () => language,
    definition: () => definition,
    select: (next) => onSelect(next),
  });
</script>

{@render children?.()}
